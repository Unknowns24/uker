package uker

type ContextKey string

// context constants
const (
	CONTEXT_VALUE_USERID ContextKey = "user-id"
)

// http error constants
const (
	// Shared http errors
	ERROR_HTTP_BAD_REQUEST    = "ERRN000"
	ERROR_HTTP_MISSING_DATA   = "ERRN001"
	ERROR_HTTP_INVALID_JSON   = "ERRN002"
	ERROR_HTTP_INVALID_BASE64 = "ERRN003"
	ERROR_HTTP_MALFORMED_DATA = "ERRN004"
	ERROR_HTTP_MISSING_PARAMS = "ERRN005"

	// Multipart specific errors
	ERROR_HTTP_MULTIPARTFORM_INVALID_FORM  = "ERRN010"
	ERROR_HTTP_MULTIPARTFORM_MISSING_FILES = "ERRN011"
)

// middleware error constants
const (
	ERROR_MIDDLEWARE_INVALID_COOKIE           = "ERRN500"
	ERROR_MIDDLEWARE_INVALID_JWT              = "ERRN501"
	ERROR_MIDDLEWARE_INVALID_JWT_USER         = "ERRN502"
	ERROR_MIDDLEWARE_NO_AUTHENTICATED_USER    = "ERRN503"
	ERROR_MIDDLEWARE_INSUFFICIENT_PERMISSIONS = "ERRN504"
	ERROR_MIDDLEWARE_NOT_AUTHENTICATED_ROUTE  = "ERRN505"
)

// http header constants
const (
	HTTP_HEADER_CLOUDFLARE_USERIP = "Cf-Connecting-Ip"
)

// request constants
const (
	REQUEST_KEY_DATA    = "data"
	REQUEST_KEY_MESSAGE = "message"
)

// pagination constants
const (
	PAGINATION_ORDER_ASC      = "asc"
	PAGINATION_ORDER_DESC     = "desc"
	PAGINATION_QUERY_SORT     = "sort"
	PAGINATION_QUERY_PAGE     = "page"
	PAGINATION_QUERY_SEARCH   = "search"
	PAGINATION_QUERY_PERPAGE  = "per_page"
	PAGINATION_QUERY_SORT_DIR = "sort_dir"
)

// middleware contants
const (
	JWT_COOKIE_NAME      = "jwt"
	JWT_CLAIM_KEY_IP     = "ip"
	JWT_CLAIM_KEY_DATA   = "data"
	JWT_CLAIM_KEY_ISSUER = "iss"
)

// logger constants
const (
	LOGGER_LEVEL_INFO  = "info"
	LOGGER_LEVEL_WARN  = "warn"
	LOGGER_LEVEL_ERROR = "error"
	LOGGER_LEVEL_DEBUG = "debug"
	LOGGER_LEVEL_FATAL = "fatal"
)

// struct tag
const (
	UKER_STRUCT_TAG          = "uker"
	UKER_STRUCT_TAG_REQUIRED = "required"
)
