package constants

// http error constants
const (
	ERROR_HTTP_BAD_REQUEST    = "ERRN100"
	ERROR_HTTP_MISSING_DATA   = "ERRN101"
	ERROR_HTTP_INVALID_JSON   = "ERRN102"
	ERROR_HTTP_INVALID_BASE64 = "ERRN103"
)

// middleware error constants
const (
	ERROR_MIDDLEWARE_INVALID_JWT      = "ERRN500"
	ERROR_MIDDLEWARE_INVALID_JWT_USER = "ERRN501"
)