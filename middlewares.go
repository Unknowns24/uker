package uker

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/unknowns24/uker/proto"
)

// middleware contants
const (
	jwt_cookie_name      = "jwt"
	jwt_claim_key_ip     = "ip"
	jwt_claim_key_data   = "data"
	jwt_claim_key_issuer = "iss"
)

// variable to store jwt key
var jwt_key string

// Global interface
type middlewares interface {
	// Generate a valid JWT
	//
	// @param id uint: User id.
	//
	// @param keeplogin bool: Param to extend jwt valid time.
	//
	// @return (string, error): generated jwt & error if exists
	GenerateJWT(id uint, keeplogin bool) (string, error)

	// Middleware to validate if user is authenticated with a valid JWT
	//
	// @param c *fiber.Ctx: Current fiber context.
	//
	// @return error: error on authentication
	IsAuthenticated(c *fiber.Ctx) error

	// Middleware to validate if user has required permissions
	//
	// @param authService proto.AuthServiceClient: authService with stablished connection to make the request.
	//
	// @param permissions []string: Array with the required permissions that user needs to have.
	//
	// @return func(c *fiber.Ctx) error: fiber middleware function to use on the
	HasPermissions(authService proto.AuthServiceClient, permissions []string) func(c *fiber.Ctx) error

	// Middleware to validate if user has at least one of the required permissions
	//
	// @param authService proto.AuthServiceClient: authService with stablished connection to make the request.
	//
	// @param permissions []string: Array with the required permissions that user needs to have.
	//
	// @return func(c *fiber.Ctx) error: fiber middleware function to use on the
	HasAtLeastOnePermission(authService proto.AuthServiceClient, permissions []string) func(c *fiber.Ctx) error
}

// Local struct to be implmented
type middlewares_implementation struct{}

// External contructor
func NewMiddlewares(jwtKey string) middlewares {
	jwt_key = jwtKey
	return &middlewares_implementation{}
}

func (m *middlewares_implementation) IsAuthenticated(c *fiber.Ctx) error {
	cookie := c.Cookies(jwt_cookie_name)

	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwt_key), nil
	})

	if err != nil || !token.Valid {
		return endOutPut(c, fiber.StatusUnauthorized, ERROR_MIDDLEWARE_INVALID_JWT, nil)
	}

	claims := token.Claims.(jwt.MapClaims)

	data := claims[jwt_claim_key_data].(map[string]interface{})
	ip := data[jwt_claim_key_ip].(string)

	id, err := strconv.ParseUint(claims[jwt_claim_key_issuer].(string), 10, 32)

	if err != nil {
		return endOutPut(c, fiber.StatusUnauthorized, ERROR_MIDDLEWARE_INVALID_JWT, nil)
	}

	if id == 0 || (ip != c.Get(HTTP_HEADER_NGINX_USERIP, c.IP())) {
		return endOutPut(c, fiber.StatusUnauthorized, ERROR_MIDDLEWARE_INVALID_JWT_USER, nil)
	}

	c.Context().SetUserValue(CONTEXT_VALUE_USERID, uint(id))
	return c.Next()
}

func (m *middlewares_implementation) GenerateJWT(id uint, keeplogin bool) (string, error) {
	payload := jwt.RegisteredClaims{}
	payload.Subject = strconv.Itoa(int(id))
	payload.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 24)) // JWT Have 1 day of duration

	if keeplogin {
		payload.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)) // JWT Have 1 week of duration
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString([]byte(jwt_key))

	return token, err
}

func (m *middlewares_implementation) HasPermissions(authService proto.AuthServiceClient, permissions []string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Get userId from context
		userId := c.Context().UserValue(CONTEXT_VALUE_USERID).(uint)

		if userId == 0 {
			return endOutPut(c, fiber.StatusUnauthorized, ERROR_MIDDLEWARE_NO_AUTHENTICATED_USER, nil)
		}

		// Validate if user have every permission
		for _, perm := range permissions {
			havePermRes, err := authService.HavePermission(context.Background(), &proto.HavePermReq{
				UserId:     uint64(userId),
				Permission: perm,
			})

			//TODO: check the way HavePermissions return error
			if err != nil {
				return c.SendString(err.Error())
			}

			if !havePermRes.HavePermission {
				return endOutPut(c, fiber.StatusBadRequest, ERROR_MIDDLEWARE_INSUFICIENT_PERMISSIONS, nil)
			}
		}

		return c.Next()
	}
}

func (m *middlewares_implementation) HasAtLeastOnePermission(authService proto.AuthServiceClient, permissions []string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Get userId from context
		userId := c.Context().UserValue(CONTEXT_VALUE_USERID).(uint)

		if userId == 0 {
			return endOutPut(c, fiber.StatusUnauthorized, ERROR_MIDDLEWARE_NO_AUTHENTICATED_USER, nil)
		}

		// Validate if user have at least one permission
		for _, perm := range permissions {
			havePermRes, err := authService.HavePermission(context.Background(), &proto.HavePermReq{
				UserId:     uint64(userId),
				Permission: perm,
			})

			//TODO: check the way HavePermissions return error
			if err != nil {
				return c.SendString(err.Error())
			}

			if havePermRes.HavePermission {
				return c.Next()
			}
		}

		return endOutPut(c, fiber.StatusBadRequest, ERROR_MIDDLEWARE_INSUFICIENT_PERMISSIONS, nil)
	}
}
