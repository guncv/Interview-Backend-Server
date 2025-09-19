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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue/publisher"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type ResumeService interface {
	ListResume(ctx context.Context, req *entities.ListResumeRequest) (*entities.ListResumeResponse, error)
	SwitchDefaultResume(ctx context.Context, req *entities.SwitchDefaultResumeRequest) error
	GetResumeByID(ctx context.Context, req *entities.GetResumeByIDRequest) (*entities.GetResumeByIDResponse, error)
	DownloadResumeBySessionToken(ctx context.Context, req *entities.DownloadResumeBySessionTokenReq) (*entities.DownloadResumeBySessionTokenResp, error)
}

type resumeService struct {
	log                     *log.Logger
	resumeRepo              repositories.ResumeReposity
	authContext             middleware.AuthContext
	s3Storage               aws.S3Storage
	validator               utils.Validator
	queue                   publisher.RedisTaskPublisher
	redisClient             database.RedisClient
	generator               utils.Generator
	interviewSessionService InterviewSessionService
}

func NewResumeService(
	l *log.Logger,
	resumeRepo repositories.ResumeReposity,
	authContext middleware.AuthContext,
	s3Storage aws.S3Storage,
	validator utils.Validator,
	queue publisher.RedisTaskPublisher,
	redisClient database.RedisClient,
	generator utils.Generator,
	interviewSessionService InterviewSessionService,
) ResumeService {
	return &resumeService{
		log:                     l,
		resumeRepo:              resumeRepo,
		authContext:             authContext,
		s3Storage:               s3Storage,
		validator:               validator,
		queue:                   queue,
		redisClient:             redisClient,
		generator:               generator,
		interviewSessionService: interviewSessionService,
	}
}

func (s *resumeService) ListResume(ctx context.Context, req *entities.ListResumeRequest) (*entities.ListResumeResponse, error) {
	s.log.InfoWithID(ctx, "[Service: ListResume] Called")

	authCtx, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListResume] Error getting auth context", err)
		return nil, err
	}
	userID, err := uuid.Parse(authCtx.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: ListResume] Invalid user ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	var (
		defaultResume db.Resumes
		resumeList    []db.Resumes
	)

	defaultResumeKey := fmt.Sprintf("%s%s", constants.RedisPrefixDefaultResume, userID.String())
	cacheValue, err := s.redisClient.Get(ctx, defaultResumeKey)

	if err == nil {
		if err := json.Unmarshal([]byte(cacheValue), &defaultResume); err != nil {
			s.log.WarnWithID(ctx, "[Service: ListResume] Redis hit but failed to unmarshal, falling back", err)
			goto FetchBoth
		}

		s.log.InfoWithID(ctx, "[Service: ListResume] Redis hit for default resume, fetching only resume list")

		if req.UpdatedAt != nil {
			resumeList, err = s.resumeRepo.ListResumeByUserIDPaginated(ctx, &db.ListResumeByUserIDPaginatedParams{
				UserID:    userID,
				UpdatedAt: *req.UpdatedAt,
			})
		} else {
			resumeList, err = s.resumeRepo.ListResumeByUserIDFirstPage(ctx, userID)
		}
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
			var list []db.Resumes
			var err error

			if req.UpdatedAt != nil {
				list, err = s.resumeRepo.ListResumeByUserIDPaginated(ctx, &db.ListResumeByUserIDPaginatedParams{
					UserID:    userID,
					UpdatedAt: *req.UpdatedAt,
				})
			} else {
				list, err = s.resumeRepo.ListResumeByUserIDFirstPage(ctx, userID)
			}
			if err != nil {
				errorChan <- err
				return
			}
			resumeListChan <- list
		}()

		go func() {
			resume, err := s.resumeRepo.GetDefaultResumeByUserID(ctx, userID)
			if err != nil {
				if appErr, ok := err.(*app_error.AppError); ok && appErr.Code == app_error.ErrCodeResumeNotFound {
					s.log.InfoWithID(ctx, "[Service: ListResume] No default resume found for user", err)
					defaultResumeChan <- db.Resumes{}
					return
				}
				errorChan <- err
				return
			}
			if resume != nil {
				defaultResumeChan <- *resume
			} else {
				defaultResumeChan <- db.Resumes{}
			}
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

		if defaultResume.ID != uuid.Nil {
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
	}

Finalize:
	var defaultResumeResp *entities.GetListResumeByIdResponse
	count := len(resumeList)
	if defaultResume.ID != uuid.Nil {
		count++
		defaultResumeResp = &entities.GetListResumeByIdResponse{
			ID:        defaultResume.ID.String(),
			FileName:  defaultResume.FileName,
			MimeType:  defaultResume.MimeType,
			ByteSize:  defaultResume.ByteSize,
			CreatedAt: utils.FormatToBangkokTime(defaultResume.CreatedAt),
			UpdatedAt: utils.FormatToBangkokTime(defaultResume.UpdatedAt),
		}
	} else {
		defaultResumeResp = &entities.GetListResumeByIdResponse{
			ID:        "",
			FileName:  "",
			MimeType:  "",
			ByteSize:  0,
			CreatedAt: "",
			UpdatedAt: "",
		}
	}

	var resumeContent *entities.ResumeContent
	if count > 0 {
		resumeContent = &entities.ResumeContent{
			DefaultResume: *defaultResumeResp,
			Resumes:       []entities.GetListResumeByIdResponse{},
		}
	}

	var lastUpdatedAt *string
	if defaultResume.ID != uuid.Nil {
		formattedTime := utils.FormatToUTCString(defaultResume.UpdatedAt)
		lastUpdatedAt = &formattedTime
	}

	if len(resumeList) > 0 {
		mostRecentResume := resumeList[0]
		for _, resume := range resumeList {
			if resume.UpdatedAt.After(mostRecentResume.UpdatedAt) {
				mostRecentResume = resume
			}
		}

		if defaultResume.ID != uuid.Nil {
			if mostRecentResume.UpdatedAt.After(defaultResume.UpdatedAt) {
				formattedTime := utils.FormatToUTCString(mostRecentResume.UpdatedAt)
				lastUpdatedAt = &formattedTime
			}
		} else {
			formattedTime := utils.FormatToUTCString(mostRecentResume.UpdatedAt)
			lastUpdatedAt = &formattedTime
		}
	}

	if defaultResume.ID == uuid.Nil && len(resumeList) == 0 {
		lastUpdatedAt = nil
	}

	resp := entities.ListResumeResponse{
		Count:         count,
		ResumeContent: resumeContent,
		LastUpdatedAt: lastUpdatedAt,
	}

	if resumeContent != nil {
		for _, resume := range resumeList {
			if !resume.IsDefault {
				resp.ResumeContent.Resumes = append(resp.ResumeContent.Resumes, entities.GetListResumeByIdResponse{
					ID:        resume.ID.String(),
					FileName:  resume.FileName,
					MimeType:  resume.MimeType,
					ByteSize:  resume.ByteSize,
					CreatedAt: utils.FormatToBangkokTime(resume.CreatedAt),
					UpdatedAt: utils.FormatToBangkokTime(resume.UpdatedAt),
				})
			}
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

	userID, err := uuid.Parse(authCtx.Payload.UserID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Invalid user ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	var defaultResume *db.Resumes
	defaultResumeKey := fmt.Sprintf("%s%s", constants.RedisPrefixDefaultResume, authCtx.Payload.UserID)
	defaultResumeFromRedis, err := s.redisClient.Get(ctx, defaultResumeKey)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.log.InfoWithID(ctx, "[Service: SwitchDefaultResume] Default resume not in Redis, creating new default resume")

			defaultResume, err = s.resumeRepo.GetDefaultResumeByUserID(ctx, userID)
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

			defaultResume, err = s.resumeRepo.GetDefaultResumeByUserID(ctx, userID)
			if err != nil {
				s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error getting default resume", err)
				return err
			}
		}
	}

	resumeID, err := uuid.Parse(req.ResumeID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Invalid resume ID", err)
		return app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	if err := s.resumeRepo.SwitchDefaultResume(ctx, defaultResume.ID, resumeID); err != nil {
		s.log.ErrorWithID(ctx, "[Service: SwitchDefaultResume] Error switching default resume", err)
		return err
	}

	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_ = s.redisClient.Delete(cacheCtx, constants.RedisPrefixDefaultResume+authCtx.Payload.UserID)
	}()

	return nil
}

func (s *resumeService) GetResumeByID(ctx context.Context, req *entities.GetResumeByIDRequest) (*entities.GetResumeByIDResponse, error) {
	s.log.InfoWithID(ctx, "[Service: GetResumeByID] Called")

	resumeID, err := uuid.Parse(req.ResumeID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetResumeByID] Invalid UUID format", err)
		return nil, app_error.New(err, app_error.ErrCodeResumeInvalidID)
	}

	resume, err := s.resumeRepo.GetResumeByID(ctx, resumeID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetResumeByID] Error getting resume", err)
		return nil, err
	}

	fileUrl, err := s.s3Storage.GeneratePresignedURL(ctx, resume.StorageKey, constants.S3PresignedURLTTL)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: GetResumeByID] Error getting file URL", err)
		return nil, err
	}

	resp := entities.GetResumeByIDResponse{
		ID:        resume.ID.String(),
		FileName:  resume.FileName,
		MimeType:  resume.MimeType,
		ByteSize:  resume.ByteSize,
		FileUrl:   fileUrl,
		CreatedAt: utils.FormatToBangkokTime(resume.CreatedAt),
		UpdatedAt: utils.FormatToBangkokTime(resume.UpdatedAt),
	}

	return &resp, nil
}

func (s *resumeService) DownloadResumeBySessionToken(ctx context.Context, req *entities.DownloadResumeBySessionTokenReq) (*entities.DownloadResumeBySessionTokenResp, error) {
	s.log.InfoWithID(ctx, "[Service: DownloadResumeBySessionToken] Called")

	authContext, err := s.authContext.GetAuthContext(ctx)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: DownloadResumeBySessionToken] Error getting auth context", err)
		return nil, err
	}

	sessionPayload, err := s.interviewSessionService.IsSessionValid(ctx, &entities.IsSessionValidReq{
		SessionToken: req.SessionToken,
		UserID:       authContext.Payload.UserID,
	})
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: DownloadResumeBySessionToken] Invalid session", err)
		return nil, err
	}

	s.log.InfoWithID(ctx, "[Service: DownloadResumeBySessionToken] Session payload", sessionPayload)

	resumeID, err := uuid.Parse(sessionPayload.ResumeID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: DownloadResumeBySessionToken] Invalid resume ID", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralInvalidUUID)
	}

	resume, err := s.resumeRepo.GetResumeByID(ctx, resumeID)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: DownloadResumeBySessionToken] Error getting resume", err)
		return nil, err
	}

	fileUrl, err := s.s3Storage.GeneratePresignedURL(ctx, resume.StorageKey, constants.S3PresignedURLTTL)
	if err != nil {
		s.log.ErrorWithID(ctx, "[Service: DownloadResumeBySessionToken] Error getting file URL", err)
		return nil, err
	}

	resp := &entities.DownloadResumeBySessionTokenResp{
		FileUrl:  fileUrl,
		FileName: resume.FileName,
	}

	return resp, nil
}
