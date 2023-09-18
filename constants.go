package uker

// http error constants
const (
	// Shared http errors
	ERROR_HTTP_BAD_REQUEST    = "ERRN000"
	ERROR_HTTP_MISSING_DATA   = "ERRN001"
	ERROR_HTTP_INVALID_JSON   = "ERRN002"
	ERROR_HTTP_INVALID_BASE64 = "ERRN003"

	// Multipart specific errors
	ERROR_HTTP_MULTIPARTFORM_INVALID_FORM  = "ERRN004"
	ERROR_HTTP_MULTIPARTFORM_MISSING_FILES = "ERRN005"
)

// middleware error constants
const (
	ERROR_MIDDLEWARE_INVALID_JWT      = "ERRN500"
	ERROR_MIDDLEWARE_INVALID_JWT_USER = "ERRN501"
)

// http header constants
const (
	HTTP_HEADER_NGINX_USERIP = "X-Real-IP"
)

// context constants
const (
	CONTEXT_VALUE_USERID = "user-id"
)
