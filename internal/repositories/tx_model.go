package repositories

type AdminCreateUserTxModel struct {
	Email          string
	Password       string
	Name           string
	OrganizationID string
	Role           string
}

type ResetUserPasswordTxModel struct {
	UserID       string
	PasswordHash string
	ResetToken   string
}

type AdminUpdateCourseTxModel struct {
	CourseId     string
	Sku          string
	Title        string
	Description  string
	Language     string
	CategoryID   string
	ThumbnailUrl string
	UpdatedBy    string
}
