package entities

import (
	"mime/multipart"
	"time"
)

type CreateResumeWithRequirementsRequest struct {
	File            *multipart.FileHeader `form:"file" binding:"required"`
	Position        string                `form:"position" binding:"required"`
	Company         string                `form:"company" binding:"required"`
	WorkType        string                `form:"work_type" binding:"required"`
	JobRequirements string                `form:"job_requirements" binding:"required"`
	InterviewType   string                `form:"interview_type" binding:"required"`
	Language        string                `form:"language" binding:"required"`
}

type GetListResumeResponse struct {
	DefaultResume GetListResumeByIdResponse   `json:"default_resume"`
	Resumes       []GetListResumeByIdResponse `json:"resumes"`
}

type GetListResumeByIdResponse struct {
	ID        string    `json:"id"`
	FileName  string    `json:"file_name"`
	MimeType  string    `json:"mime_type"`
	ByteSize  int32     `json:"byte_size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SwitchDefaultResumeRequest struct {
	ResumeID string `json:"resume_id" binding:"required"`
}
