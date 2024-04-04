package uker

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Global interface
type Middlewares interface {
	// Generate a valid JWT
	//
	// @param id uint: User id.
	//
	// @param keeplogin bool: Param to extend jwt valid time.
	//
	// @return (string, error): generated jwt & error if exists
	GenerateJWT(id string, keeplogin bool, ipAddress string) (string, http.Cookie, error)

	// Middleware to validate if user is authenticated with a valid JWT
	//
	// @return http.Handler: Handler function used as middleware
	IsAuthenticated(next http.Handler) http.Handler

	// Middleware to validate if user is not authenticated
	//
	// @return http.Handler: Handler function used as middleware
	NotAuthenticated(next http.Handler) http.Handler

	// Middleware to check if user is authenticated with a valid JWT
	//
	// @return http.Handler: Handler function used as middleware
	OptionalAuthenticated(next http.Handler) http.Handler
}

// Local struct to be implmented
type middlewares_implementation struct {
	secret string
}

// External contructor
func NewMiddlewares(jwtKey string) Middlewares {
	return &middlewares_implementation{
		secret: jwtKey,
	}
}

func (m *middlewares_implementation) NotAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(JWT_COOKIE_NAME)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			return []byte(m.secret), nil
		})

		if err != nil || !token.Valid {
			next.ServeHTTP(w, r)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		data := claims[JWT_CLAIM_KEY_DATA].(map[string]interface{})
		id := claims[JWT_CLAIM_KEY_ISSUER].(string)
		ip := data[JWT_CLAIM_KEY_IP].(string)

		if id == "" || (ip != r.Context().Value(HTTP_HEADER_CLOUDFLARE_USERIP) && ip != r.RemoteAddr) {
			next.ServeHTTP(w, r)
			return
		}

		if id == "" {
			next.ServeHTTP(w, r)
			return
		}

		errorOutPut(w, http.StatusUnauthorized, ERROR_MIDDLEWARE_NOT_AUTHENTICATED_ROUTE)
	})
}

func (m *middlewares_implementation) IsAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(JWT_COOKIE_NAME)
		if err != nil {
			errorOutPut(w, http.StatusUnauthorized, ERROR_MIDDLEWARE_INVALID_COOKIE)
			return
		}

		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			return []byte(m.secret), nil
		})

		if err != nil || !token.Valid {
			errorOutPut(w, http.StatusUnauthorized, ERROR_MIDDLEWARE_INVALID_JWT)
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		data := claims[JWT_CLAIM_KEY_DATA].(map[string]interface{})
		ip := data[JWT_CLAIM_KEY_IP].(string)

		id := claims[JWT_CLAIM_KEY_ISSUER].(string)

		if id == "" || (ip != r.Context().Value(HTTP_HEADER_CLOUDFLARE_USERIP) && ip != r.RemoteAddr) {
			errorOutPut(w, http.StatusUnauthorized, ERROR_MIDDLEWARE_INVALID_JWT_USER)
			return
		}

		ctx := context.WithValue(r.Context(), CONTEXT_VALUE_USERID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *middlewares_implementation) OptionalAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(JWT_COOKIE_NAME)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			return []byte(m.secret), nil
		})

		if err != nil || !token.Valid {
			next.ServeHTTP(w, r)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		data := claims[JWT_CLAIM_KEY_DATA].(map[string]interface{})
		ip := data[JWT_CLAIM_KEY_IP].(string)
		id := claims[JWT_CLAIM_KEY_ISSUER].(string)

		if id == "" || (ip != r.Context().Value(HTTP_HEADER_CLOUDFLARE_USERIP) && ip != r.RemoteAddr) {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), CONTEXT_VALUE_USERID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *middlewares_implementation) GenerateJWT(id string, keeplogin bool, ipAddress string) (string, http.Cookie, error) {
	// Generate date depending on keeplogin
	date := time.Hour * 24 // JWT Have 1 day of duration

	if keeplogin {
		date = time.Hour * 24 * 7 // JWT Have 1 week of duration
	}

	// Creating custom claims
	claims := jwt.MapClaims{
		"iss": id,
		"exp": jwt.NewNumericDate(time.Now().Add(date)).Unix(),
		"data": map[string]string{
			"ip": ipAddress,
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(m.secret))
	if err != nil {
		return "", http.Cookie{}, err
	}

	cookie := http.Cookie{
		Name:     JWT_COOKIE_NAME,
		Path:     "/",
		Value:    token,
		MaxAge:   int(date.Abs().Seconds()),
		HttpOnly: true,
	}

	return token, cookie, nil
}
