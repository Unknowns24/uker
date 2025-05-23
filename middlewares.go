package uker

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// These are the default errors returned by this middleware
var defaultErrors = &MiddlewareErrors{
	NotAuthenticatedRoute: &ResponseStatus{
		Type:        ERROR,
		Code:        "ERROR_MIDDLEWARE_NOT_AUTHENTICATED_ROUTE",
		Description: "This route is for not authenticated users",
	},

	InvalidJWTCookie: &ResponseStatus{
		Type:        ERROR,
		Code:        "ERROR_MIDDLEWARE_INVALID_COOKIE",
		Description: "The cookie in the request is expired or not valid",
	},

	InvalidJWTUser: &ResponseStatus{
		Type:        ERROR,
		Code:        "ERROR_MIDDLEWARE_INVALID_JWT",
		Description: "User inside JWT is invalid",
	},

	InvalidJWT: &ResponseStatus{
		Type:        ERROR,
		Code:        "ERROR_MIDDLEWARE_INVALID_JWT_USER",
		Description: "JWT is not valid",
	},
}

// Global interface
type Middlewares interface {
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
	errors *MiddlewareErrors
}

type MiddlewareErrors struct {
	NotAuthenticatedRoute any
	InvalidJWTCookie      any
	InvalidJWTUser        any
	InvalidJWT            any
}

type MiddlewareOptions struct {
	Errors MiddlewareErrors
}

// External contructor
func NewMiddlewares(jwtKey string, opts *MiddlewareOptions) Middlewares {
	errors := defaultErrors

	if opts != nil {
		if opts.Errors.NotAuthenticatedRoute != nil {
			errors.NotAuthenticatedRoute = opts.Errors.NotAuthenticatedRoute
		}

		if opts.Errors.InvalidJWTCookie != nil {
			errors.InvalidJWTCookie = opts.Errors.InvalidJWTCookie
		}

		if opts.Errors.InvalidJWTUser != nil {
			errors.InvalidJWTUser = opts.Errors.InvalidJWTUser
		}

		if opts.Errors.InvalidJWT != nil {
			errors.InvalidJWT = opts.Errors.InvalidJWT
		}
	}

	return &middlewares_implementation{
		secret: jwtKey,
		errors: errors,
	}
}

func (m *middlewares_implementation) NotAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(JWT_COOKIE_NAME)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(m.secret), nil
		})

		if err != nil || !token.Valid {
			next.ServeHTTP(w, r)
			return
		}

		if claims.ExpiresAt.Before(time.Now()) {
			next.ServeHTTP(w, r)
			return
		}

		if claims.Subject == "" {
			next.ServeHTTP(w, r)
			return
		}

		errorOutPut(w, http.StatusUnauthorized, m.errors.NotAuthenticatedRoute)
	})
}

func (m *middlewares_implementation) IsAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(JWT_COOKIE_NAME)
		if err != nil {
			errorOutPut(w, http.StatusUnauthorized, m.errors.InvalidJWTCookie)
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(m.secret), nil
		})

		if err != nil || !token.Valid {
			errorOutPut(w, http.StatusUnauthorized, m.errors.InvalidJWT)
			return
		}

		if claims.ExpiresAt.Before(time.Now()) {
			errorOutPut(w, http.StatusUnauthorized, m.errors.InvalidJWT)
			return
		}

		if claims.Subject == "" {
			errorOutPut(w, http.StatusUnauthorized, m.errors.InvalidJWTUser)
			return
		}

		ctx := context.WithValue(r.Context(), CONTEXT_VALUE_USERID, claims.Subject)
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

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(m.secret), nil
		})

		if err != nil || !token.Valid {
			next.ServeHTTP(w, r)
			return
		}

		if claims.ExpiresAt.Before(time.Now()) {
			next.ServeHTTP(w, r)
			return
		}

		if claims.Subject == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), CONTEXT_VALUE_USERID, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
