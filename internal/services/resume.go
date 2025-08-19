package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	ListResume(ctx context.Context, req *entities.ListResumeRequest) (*entities.ListResumeResponse, error)
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
	generator   utils.Generator
}

func NewResumeService(
	l *log.Logger,
	resumeRepo repositories.ResumeReposity,
	authContext middleware.AuthContext,
	s3Storage aws.S3Storage,
	validator utils.Validator,
	queue queue.RedisTaskPublisher,
	redisClient database.RedisClient,
	generator utils.Generator,
) ResumeService {
	return &resumeService{
		log:         l,
		resumeRepo:  resumeRepo,
		authContext: authContext,
		s3Storage:   s3Storage,
		validator:   validator,
		queue:       queue,
		redisClient: redisClient,
		generator:   generator,
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
		ID:         s.generator.GenerateUUID(ctx),
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

	return nil
}

func (s *resumeService) ListResume(ctx context.Context, req *entities.ListResumeRequest) (*entities.ListResumeResponse, error) {
	s.log.InfoWithID(ctx, "[Service: tListResume] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListResume] Error getting auth context", err)
		return nil, err
	}
	userID := uuid.MustParse(authCtx.Payload.UserID)

	var (
		defaultResume db.Resumes
		resumeList    []db.Resumes
	)

	var updatedAt time.Time
	if req.UpdatedAt != nil {
		updatedAt = *req.UpdatedAt
	} else {
		updatedAt = time.Time{}
	}

	defaultResumeKey := fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, userID.String())
	cacheValue, err := s.redisClient.Get(ctx, defaultResumeKey)

	if err == nil {
		if err := json.Unmarshal([]byte(cacheValue), &defaultResume); err != nil {
			s.log.WarnWithID(ctx, "[Service: ListResume] Redis hit but failed to unmarshal, falling back", err)
			goto FetchBoth
		}

		s.log.InfoWithID(ctx, "[Service: ListResume] Redis hit for default resume, fetching only resume list")
		resumeList, err = s.resumeRepo.ListResumeByUserID(ctx, &db.ListResumeByUserIDParams{
			UserID:    userID,
			UpdatedAt: updatedAt,
		})
		if err != nil {
			s.log.ErrorWithID(ctx, "[Service: ListResume] Error getting resume list", err)
			return nil, err
		}
	} else if errors.Is(err, redis.Nil) {
		s.log.InfoWithID(ctx, "[Service: ListResume] Default resume not in Redis, fetching both concurrently")
		goto FetchBoth
	} else {
		s.log.ErrorWithID(ctx, "[Service: ListResume] Redis error", err)
		return nil, err
	}

	goto Finalize

FetchBoth:
	{
		resumeListChan := make(chan []db.Resumes, 1)
		defaultResumeChan := make(chan db.Resumes, 1)
		errorChan := make(chan error, 2)

		go func() {
			list, err := s.resumeRepo.ListResumeByUserID(ctx, &db.ListResumeByUserIDParams{
				UserID:    userID,
				UpdatedAt: updatedAt,
			})
			if err != nil {
				errorChan <- err
				return
			}
			resumeListChan <- list
		}()

		go func() {
			resume, err := s.resumeRepo.GetDefaultResumeByUserID(ctx, userID)
			if err != nil {
				errorChan <- err
				return
			}
			defaultResumeChan <- resume
		}()

		var completed int
		for completed < 2 {
			select {
			case err := <-errorChan:
				s.log.ErrorWithID(ctx, "[Service: ListResume] Error in concurrent fetch", err)
				return nil, err
			case list := <-resumeListChan:
				resumeList = list
				completed++
			case resume := <-defaultResumeChan:
				defaultResume = resume
				completed++
			case <-ctx.Done():
				s.log.ErrorWithID(ctx, "[Service: ListResume] Context canceled", ctx.Err())
				return nil, ctx.Err()
			}
		}

		raw, err := json.Marshal(defaultResume)
		if err != nil {
			s.log.WarnWithID(ctx, "[Service: ListResume] Error marshalling default resume", err)
		} else {
			if err := s.redisClient.Set(ctx, database.RedisPayload{
				Key:   defaultResumeKey,
				Value: string(raw),
				TTL:   24 * time.Hour,
			}); err != nil {
				s.log.ErrorWithID(ctx, "[Service: ListResume] Error caching default resume in Redis", err)
			}
		}
	}

Finalize:
	resp := entities.ListResumeResponse{
		DefaultResume: entities.GetListResumeByIdResponse{
			ID:        defaultResume.ID.String(),
			FileName:  defaultResume.FileName,
			MimeType:  defaultResume.MimeType,
			ByteSize:  defaultResume.ByteSize,
			CreatedAt: defaultResume.CreatedAt,
			UpdatedAt: defaultResume.UpdatedAt,
		},
		Resumes: []entities.GetListResumeByIdResponse{},
	}

	for _, resume := range resumeList {
		if !resume.IsDefault {
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

	return &resp, nil
}

func (s *resumeService) SwitchDefaultResume(ctx context.Context, req *entities.SwitchDefaultResumeRequest) error {
	s.log.InfoWithID(ctx, "[Service: SwitchDefaultResume] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error getting auth context", err)
		return err
	}

	var defaultResume db.Resumes
	defaultResumeKey := fmt.Sprintf("%s:%s", constants.RedisPrefixDefaultResume, authCtx.Payload.UserID)
	defaultResumeFromRedis, err := s.redisClient.Get(ctx, defaultResumeKey)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.log.InfoWithID(ctx, "[Service: SwitchDefaultResume] Default resume not in Redis, creating new default resume")

			defaultResume, err = s.resumeRepo.GetDefaultResumeByUserID(ctx, uuid.MustParse(authCtx.Payload.UserID))
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error getting default resume", err)
				return err
			}
		} else {
			s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error getting default resume from Redis", err)
			return err
		}
	} else {
		if err := json.Unmarshal([]byte(defaultResumeFromRedis), &defaultResume); err != nil {
			s.log.WarnWithID(ctx, "[Service: SwitchDefaultResume] Error unmarshalling default resume from Redis", err)

			defaultResume, err = s.resumeRepo.GetDefaultResumeByUserID(ctx, uuid.MustParse(authCtx.Payload.UserID))
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error getting default resume", err)
				return err
			}
		}
	}

	if err := s.resumeRepo.SwitchDefaultResume(ctx, defaultResume.ID, uuid.MustParse(req.ResumeID)); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error switching default resume", err)
		return err
	}

	if err := s.queue.PublishTaskDeleteRedis(ctx, &database.RedisDeletePayload{
		Keys: []string{constants.RedisPrefixDefaultResume + ":" + authCtx.Payload.UserID},
	}); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error publishing delete redis task", err)
	}

	return nil
}
