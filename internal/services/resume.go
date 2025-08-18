package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type ResumeService interface {
	CreateResumeWithRequirements(ctx context.Context, req *entities.CreateResumeWithRequirementsRequest) error
	// GetListResume(ctx context.Context) (*entities.GetListResumeResponse, error)
}

type resumeService struct {
	log         *log.Logger
	resumeRepo  repositories.ResumeReposity
	authContext middleware.AuthContext
	s3Storage   aws.S3Storage
	validator   utils.Validator
}

func NewResumeService(
	l *log.Logger,
	resumeRepo repositories.ResumeReposity,
	authContext middleware.AuthContext,
	s3Storage aws.S3Storage,
	validator utils.Validator,
) ResumeService {
	return &resumeService{
		log:         l,
		resumeRepo:  resumeRepo,
		authContext: authContext,
		s3Storage:   s3Storage,
		validator:   validator,
	}
}

func (s *resumeService) CreateResumeWithRequirements(ctx context.Context, req *entities.CreateResumeWithRequirementsRequest) error {
	s.log.InfoWithID(ctx, "[Service: CreateResume] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateResume] Error getting auth context", err)
		return err
	}

	if !s.validator.IsAllowedResumeContentType(ctx, req.File) {
		s.log.ErrorWithID(ctx, "[Service: CreateResume] Invalid file type", errors.New("invalid file type"))
		return app_error.New(errors.New("invalid file type"), app_error.ErrCodeResumeInvalidFileContentType)
	}

	if req.File.Size > int64(constants.ResumeMaxFileSize) {
		s.log.ErrorWithID(ctx, "[Service: CreateResume] File size is too large", errors.New("file size is too large"))
		return app_error.New(errors.New("file size is too large"), app_error.ErrCodeResumeInvalidFileSize)
	}

	isDefaultResume, err := s.resumeRepo.CheckIsDefaultResumeExistsByUserID(ctx, uuid.MustParse(authCtx.Payload.UserID))
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateResume] Error getting default resume", err)
		return err
	}

	storageKey, err := s.s3Storage.UploadFile(ctx, req.File, constants.S3ResumeKey, authCtx.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateResume] Error uploading file", err)
		return err
	}

	resumeReq := &db.CreateResumeParams{
		UserID:     uuid.MustParse(authCtx.Payload.UserID),
		StorageKey: storageKey,
		FileName:   req.File.Filename,
		MimeType:   req.File.Header.Get("Content-Type"),
		ByteSize:   int32(req.File.Size),
		IsDefault:  !isDefaultResume,
	}

	if err := s.resumeRepo.CreateResume(ctx, resumeReq); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateResume] Error creating resume", err)
		return err
	}

	return nil
}

// func (s *resumeService) GetListResume(ctx context.Context) (*entities.GetListResumeResponse, error) {
// 	s.log.InfoWithID(ctx, "[Service: GetListResume] Called")

// 	authCtx, err := s.authContext.GetAuthContext(ctx)
// 	if err != nil {
// 		s.log.ErrorWithID(ctx, "[Service: GetListResume] Error getting auth context", err)
// 		return nil, err
// 	}

// 	resumeList, err := s.resumeRepo.GetListResumeByUserID(ctx, uuid.MustParse(authCtx.Payload.UserID))
// 	if err != nil {
// 		s.log.ErrorWithID(ctx, "[Service: GetListResume] Error getting resume list", err)
// 		return nil, err
// 	}

// 	resp := entities.GetListResumeResponse{
// 		DefaultResume: entities.GetListResumeByIdResponse{},
// 		Resumes:       []entities.GetListResumeByIdResponse{},
// 	}

// 	for _, resume := range resumeList {

// 		if resume.IsDefault {
// 			resp.DefaultResume = entities.GetListResumeByIdResponse{
// 				ID:        resume.ID.String(),
// 				FileName:  resume.FileName,
// 				MimeType:  resume.MimeType,
// 				ByteSize:  resume.ByteSize,
// 				CreatedAt: resume.CreatedAt,
// 				UpdatedAt: resume.UpdatedAt,
// 			}
// 		} else {
// 			resp.Resumes = append(resp.Resumes, entities.GetListResumeByIdResponse{
// 				ID:        resume.ID.String(),
// 				FileName:  resume.FileName,
// 				MimeType:  resume.MimeType,
// 				ByteSize:  resume.ByteSize,
// 				CreatedAt: resume.CreatedAt,
// 				UpdatedAt: resume.UpdatedAt,
// 			})
// 		}
// 	}

// 	return &resp, nil
// }
