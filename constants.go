package uker

type ContextKey string

// context constants
const (
	CONTEXT_VALUE_USERID ContextKey = "user-id"
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
	PAGINATION_ORDER_ASC         = "asc"
	PAGINATION_ORDER_DESC        = "desc"
	PAGINATION_QUERY_SORT        = "sort"
	PAGINATION_QUERY_PAGE        = "page"
	PAGINATION_QUERY_SEARCH      = "search"
	PAGINATION_QUERY_PERPAGE     = "per_page"
	PAGINATION_QUERY_SORT_DIR    = "sort_dir"
	PAGINATION_QUERY_WHERE_FIELD = "where_field"
	PAGINATION_QUERY_WHERE_VALUE = "where_value"
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
