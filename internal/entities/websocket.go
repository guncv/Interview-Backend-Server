package entities

type OpenWsConnectionRequest struct {
	SessionToken string `json:"session_token" validate:"required,uuid"`
}

type RedisSessionToken struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
}
