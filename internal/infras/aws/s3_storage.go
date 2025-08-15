package aws

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	cfg "gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type S3Storage interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader, key string) (string, error)
	GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

type s3Storage struct {
	s3Client *s3.Client
	log      *log.Logger
	cfp      *cfg.Config
}

func NewS3Storage(cfp *cfg.Config, logger *log.Logger) (S3Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constants.TimeoutContext)
	defer cancel()

	customCreds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		cfp.AWSConfig.AccessKey,
		cfp.AWSConfig.SecretKey,
		"",
	))

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfp.AWSConfig.Region),
		config.WithCredentialsProvider(customCreds),
	)

	if err != nil {
		logger.ErrorWithID(ctx, "[S3: Init] Failed to load AWS config", err)
		return nil, err
	}

	s3Client := s3.NewFromConfig(awsCfg)

	return &s3Storage{
		s3Client: s3Client,
		log:      logger,
		cfp:      cfp,
	}, nil
}

func (s *s3Storage) UploadFile(ctx context.Context, file *multipart.FileHeader, key string) (string, error) {
	s.log.InfoWithID(ctx, "[S3: UploadFile] Uploading file Called: ", file.Filename)

	src, err := file.Open()
	if err != nil {
		s.log.ErrorWithID(ctx, "[S3: UploadFile] Failed to open file", err)
		return "", err
	}
	defer src.Close()

	fileExt := filepath.Ext(file.Filename)
	objectKey := fmt.Sprintf("%s/%d%s", key, time.Now().UnixNano(), fileExt)

	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfp.AWSConfig.S3Bucket),
		Key:         aws.String(objectKey),
		Body:        src,
		ContentType: aws.String(file.Header.Get("Content-Type")),
		ACL:         types.ObjectCannedACL("private"),
	})
	if err != nil {
		s.log.ErrorWithID(ctx, "[S3: UploadFile] Failed to upload", err)
		return "", err
	}

	return objectKey, nil
}

func (s *s3Storage) GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	s.log.InfoWithID(ctx, "[S3: GeneratePresignedURL] Generating presigned URL Called: ", key)
	presignClient := s3.NewPresignClient(s.s3Client)

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfp.AWSConfig.S3Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		s.log.ErrorWithID(ctx, "[S3: GeneratePresignedURL] Failed", err)
		return "", err
	}

	return req.URL, nil
}
