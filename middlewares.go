package uker

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// variable to store jwt key
var jwt_key string

// Global interface
type Middlewares interface {
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
	// @param next http.Handler: Current fiber context.
	//
	// @return http.Handler: error on authentication
	IsAuthenticated(next http.Handler) http.Handler
}

// Local struct to be implmented
type middlewares_implementation struct{}

// External contructor
func NewMiddlewares(jwtKey string) Middlewares {
	jwt_key = jwtKey
	return &middlewares_implementation{}
}

func (m *middlewares_implementation) IsAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(JWT_COOKIE_NAME)
		if err != nil {
			http.Error(w, ERROR_MIDDLEWARE_INVALID_COOKIE, http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwt_key), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, ERROR_MIDDLEWARE_INVALID_JWT, http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		data := claims[JWT_CLAIM_KEY_DATA].(map[string]interface{})
		ip := data[JWT_CLAIM_KEY_IP].(string)

		id, err := strconv.ParseUint(claims[JWT_CLAIM_KEY_ISSUER].(string), 10, 32)
		if err != nil {
			http.Error(w, ERROR_MIDDLEWARE_INVALID_JWT, http.StatusUnauthorized)
			return
		}

		if id == 0 || (ip != r.Context().Value(HTTP_HEADER_NGINX_USERIP) && ip != r.RemoteAddr) {
			http.Error(w, ERROR_MIDDLEWARE_INVALID_JWT_USER, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), CONTEXT_VALUE_USERID, uint(id))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
