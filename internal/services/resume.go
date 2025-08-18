package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type ResumeService interface {
	CreateResumeWithRequirements(ctx context.Context, req *entities.CreateResumeWithRequirementsRequest) error
	ListResume(ctx context.Context) (*entities.GetListResumeResponse, error)
	SwitchDefaultResume(ctx context.Context, req *entities.SwitchDefaultResumeRequest) error
}

type resumeService struct {
	log         *log.Logger
	resumeRepo  repositories.ResumeReposity
	authContext middleware.AuthContext
	s3Storage   aws.S3Storage
	validator   utils.Validator
	queue       queue.RedisTaskPublisher
	redisClient database.RedisClient
}

func NewResumeService(
	l *log.Logger,
	resumeRepo repositories.ResumeReposity,
	authContext middleware.AuthContext,
	s3Storage aws.S3Storage,
	validator utils.Validator,
	queue queue.RedisTaskPublisher,
	redisClient database.RedisClient,
) ResumeService {
	return &resumeService{
		log:         l,
		resumeRepo:  resumeRepo,
		authContext: authContext,
		s3Storage:   s3Storage,
		validator:   validator,
		queue:       queue,
		redisClient: redisClient,
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
		if err := s.queue.PublishTaskDeleteFile(ctx, &aws.DeleteFilePayload{Key: storageKey}); err != nil {
			s.log.ErrorWithID(ctx, "[Service: CreateResume] Error publishing delete file task", err)
		}

		s.log.ErrorWithID(ctx, "[Service: CreateResume] Error creating resume", err)
		return err
	}

	if err := s.queue.PublishTaskDeleteRedis(ctx, &database.RedisDeletePayload{
		Keys: []string{constants.RedisPrefixResumeList + ":" + authCtx.Payload.UserID},
	}); err != nil {
		s.log.ErrorWithID(ctx, "[Service: CreateResume] Error publishing delete redis task", err)
	}

	return nil
}

func (s *resumeService) ListResume(ctx context.Context) (*entities.GetListResumeResponse, error) {
	s.log.InfoWithID(ctx, "[Service: GetListResume] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetListResume] Error getting auth context", err)
		return nil, err
	}

	resp := entities.GetListResumeResponse{
		DefaultResume: entities.GetListResumeByIdResponse{},
		Resumes:       []entities.GetListResumeByIdResponse{},
	}

	redisKey := fmt.Sprintf("%s:%s", constants.RedisPrefixResumeList, authCtx.Payload.UserID)
	redisValue, err := s.redisClient.Get(ctx, redisKey)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.log.InfoWithID(ctx, "[Service: GetListResume] Redis key not found, getting resume list from database")

			resumeList, err := s.resumeRepo.ListResumeByUserID(ctx, uuid.MustParse(authCtx.Payload.UserID))
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: GetListResume] Error getting resume list", err)
				return nil, err
			}

			for _, resume := range resumeList {
				if resume.IsDefault {
					resp.DefaultResume = entities.GetListResumeByIdResponse{
						ID:        resume.ID.String(),
						FileName:  resume.FileName,
						MimeType:  resume.MimeType,
						ByteSize:  resume.ByteSize,
						CreatedAt: resume.CreatedAt,
						UpdatedAt: resume.UpdatedAt,
					}
				} else {
					resp.Resumes = append(resp.Resumes, entities.GetListResumeByIdResponse{
						ID:        resume.ID.String(),
						FileName:  resume.FileName,
						MimeType:  resume.MimeType,
						ByteSize:  resume.ByteSize,
						CreatedAt: resume.CreatedAt,
						UpdatedAt: resume.UpdatedAt,
					})
				}
			}

			redisKey := fmt.Sprintf("%s:%s", constants.RedisPrefixResumeList, authCtx.Payload.UserID)
			if err := s.queue.PublishTaskSetRedis(ctx, &database.RedisPayload{
				Key:   redisKey,
				Value: resp,
				TTL:   constants.RedisTTLDefault,
			}); err != nil {
				s.log.ErrorWithID(ctx, "[Service: GetListResume] Error publishing set redis task", err)
			}

			return &resp, nil
		}

		s.log.ErrorWithID(ctx, "[Service: GetListResume] Error getting resume list", err)
		return nil, err
	}

	if err := json.Unmarshal([]byte(redisValue), &resp); err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetListResume] Error unmarshalling resume list", err)
		return nil, err
	}

	return &resp, nil
}

func (s *resumeService) SwitchDefaultResume(ctx context.Context, req *entities.SwitchDefaultResumeRequest) error {
	s.log.InfoWithID(ctx, "[Service: SwitchDefaultResume] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error getting auth context", err)
		return err
	}

	defaultResume, err := s.resumeRepo.GetDefaultResumeByUserID(ctx, uuid.MustParse(authCtx.Payload.UserID))
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error getting default resume", err)
		return err
	}

	if defaultResume.ID == uuid.MustParse(req.ResumeID) {
		s.log.InfoWithID(ctx, "[Service: SwitchDefaultResume] Default resume is already the selected resume")
		return nil
	}

	if err := s.resumeRepo.SwitchDefaultResume(ctx, defaultResume.ID, uuid.MustParse(req.ResumeID)); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error switching default resume", err)
		return err
	}

	if err := s.queue.PublishTaskDeleteRedis(ctx, &database.RedisDeletePayload{
		Keys: []string{constants.RedisPrefixResumeList + ":" + authCtx.Payload.UserID},
	}); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error publishing delete redis task", err)
	}

	return nil
}
