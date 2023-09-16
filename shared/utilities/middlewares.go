package utilities

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/unknowns24/uker/shared/constants"
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
type Middlewares interface {
	GenerateJWT(id uint, keeplogin bool) (string, error)
	IsAuthenticated(c *fiber.Ctx) error
}

// Local struct to be implmented
type middlewares struct{}

// External contructor
func NewMiddlewares(jwtKey string) Middlewares {
	jwt_key = jwtKey
	return &middlewares{}
}

// Middleware to validate if user is authenticated with a valid JWT
//
// @param c *fiber.Ctx: Current fiber context.
//
// @return error: error on authentication
func (m *middlewares) IsAuthenticated(c *fiber.Ctx) error {
	cookie := c.Cookies(jwt_cookie_name)

	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwt_key), nil
	})

	if err != nil || !token.Valid {
		return endOutPut(c, fiber.StatusUnauthorized, constants.ERROR_MIDDLEWARE_INVALID_JWT, nil)
	}

	claims := token.Claims.(jwt.MapClaims)

	data := claims[jwt_claim_key_data].(map[string]interface{})
	ip := data[jwt_claim_key_ip].(string)

	id, err := strconv.ParseUint(claims[jwt_claim_key_issuer].(string), 10, 32)

	if err != nil {
		return endOutPut(c, fiber.StatusUnauthorized, constants.ERROR_MIDDLEWARE_INVALID_JWT, nil)
	}

	if id == 0 || (ip != c.Get("client-ip", c.IP())) {
		return endOutPut(c, fiber.StatusUnauthorized, constants.ERROR_MIDDLEWARE_INVALID_JWT_USER, nil)
	}

	c.Context().SetUserValue("userId", uint(id))
	return c.Next()
}

// Generate a valid JWT
//
// @param id uint: User id.
//
// @param keeplogin bool: Param to extend jwt valid time.
//
// @return (string, error): generated jwt & error if exists
func (m *middlewares) GenerateJWT(id uint, keeplogin bool) (string, error) {
	payload := jwt.RegisteredClaims{}
	payload.Subject = strconv.Itoa(int(id))
	payload.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 24)) // JWT Have 1 day of duration

	if keeplogin {
		payload.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)) // JWT Have 1 week of duration
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString([]byte(jwt_key))

	return token, err
}
