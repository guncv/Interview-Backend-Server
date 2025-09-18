package entities

type CreateUserIssueReportReq struct {
	Description string `json:"description" validate:"required,min=1,max=1000"`
	CategoryID  string `json:"category_id" validate:"required,uuid"`
}

type ListUserIssueReportsResp struct {
	Data []UserIssueReport `json:"data"`
}

type UserIssueReport struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	IsEditable   bool   `json:"is_editable"`
	Acknowledged bool   `json:"acknowledged"`
	CommentCount int32  `json:"comment_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type UpdateUserIssueReportByIDReq struct {
	Description string `json:"description" validate:"required,min=1,max=1000"`
	CategoryID  string `json:"category_id" validate:"required,uuid"`
}
