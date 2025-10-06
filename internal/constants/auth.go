package constants

// User Role Constants
const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

// Auth Constants
var (
	AuthorizationHeaderKey  ContextKey = "authorization"
	AuthorizationTypeBearer ContextKey = "bearer"
	AuthorizationPayloadKey ContextKey = "authorization_payload"
	RefreshTokenCookieKey   ContextKey = "refresh_token"
	NewAccessTokenKey       ContextKey = "new_access_token"
	XAccessTokenHeaderKey   ContextKey = "X-Access-Token"
	AuthContextKey          ContextKey = "auth_context"
	RoleKey                 ContextKey = "x-active-role"
)
