package entities

type CreateReviewCommentReq struct {
	SessionID string `json:"session_id" binding:"required"`
	Rating    int    `json:"rating" binding:"required"`
	Comment   string `json:"comment" binding:"required"`
}
