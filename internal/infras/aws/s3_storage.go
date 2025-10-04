package aws

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type CustomFileHeader struct {
	Filename    string
	Size        int64
	Header      map[string][]string
	FileContent []byte
}

func NewCustomFileHeader(filename string, size int64, header map[string][]string, fileContent []byte) *CustomFileHeader {
	return &CustomFileHeader{
		Filename:    filename,
		Size:        size,
		Header:      header,
		FileContent: fileContent,
	}
}

func (f *CustomFileHeader) Open() (multipart.File, error) {
	return &CustomFile{
		content: bytes.NewReader(f.FileContent),
		header:  f,
	}, nil
}

type CustomFile struct {
	content *bytes.Reader
	header  *CustomFileHeader
}

func (f *CustomFile) Read(p []byte) (n int, err error) {
	return f.content.Read(p)
}

func (f *CustomFile) Seek(offset int64, whence int) (int64, error) {
	return f.content.Seek(offset, whence)
}

func (f *CustomFile) Close() error {
	return nil
}

func (f *CustomFile) ReadAt(p []byte, off int64) (n int, err error) {
	return f.content.ReadAt(p, off)
}

func (f *CustomFile) WriteTo(w io.Writer) (n int64, err error) {
	return f.content.WriteTo(w)
}

type S3Storage interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader, key string, userID string) (string, error)
	GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	DeleteFile(ctx context.Context, key string) error
	DownloadFile(ctx context.Context, key string) (*CustomFileHeader, error)
}

type DeleteFilePayload struct {
	Key string
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
		cfp.AWSConfig.S3AccessKey,
		cfp.AWSConfig.S3SecretAccessKey,
		"",
	))

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfp.AWSConfig.Region),
		config.WithCredentialsProvider(customCreds),
	)

	if err != nil {
		logger.ErrorWithID(ctx, "[S3: Init] Failed to load AWS config", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	s3Client := s3.NewFromConfig(awsCfg)

	return &s3Storage{
		s3Client: s3Client,
		log:      logger,
		cfp:      cfp,
	}, nil
}

func (s *s3Storage) UploadFile(ctx context.Context, file *multipart.FileHeader, key string, userID string) (string, error) {

	src, err := file.Open()
	if err != nil {
		s.log.ErrorWithID(ctx, "[S3: UploadFile] Failed to open file", err)
		return "", app_error.New(err, app_error.ErrCodeResumeUploadFailed)
	}
	defer src.Close()

	fileExt := filepath.Ext(file.Filename)
	objectKey := fmt.Sprintf("%s/%s%d%s", key, userID, time.Now().UnixNano(), fileExt)

	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfp.AWSConfig.S3Bucket),
		Key:         aws.String(objectKey),
		Body:        src,
		ContentType: aws.String(file.Header.Get("Content-Type")),
		ACL:         types.ObjectCannedACL("private"),
	})
	if err != nil {
		s.log.ErrorWithID(ctx, "[S3: UploadFile] Failed to upload", err)
		return "", app_error.New(err, app_error.ErrCodeResumeUploadFailed)
	}

	return objectKey, nil
}

func (s *s3Storage) GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.s3Client)

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfp.AWSConfig.S3Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		s.log.ErrorWithID(ctx, "[S3: GeneratePresignedURL] Failed", err)
		return "", app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return req.URL, nil
}

func (s *s3Storage) DeleteFile(ctx context.Context, key string) error {

	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfp.AWSConfig.S3Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		s.log.ErrorWithID(ctx, "[S3: DeleteFile] Failed", err)
		return app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	return nil
}

func (s *s3Storage) DownloadFile(ctx context.Context, key string) (*CustomFileHeader, error) {

	result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfp.AWSConfig.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		s.log.ErrorWithID(ctx, "[S3: DownloadFile] Failed to get object", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}
	defer result.Body.Close()

	fileContent, err := io.ReadAll(result.Body)
	if err != nil {
		s.log.ErrorWithID(ctx, "[S3: DownloadFile] Failed to read file content", err)
		return nil, app_error.New(err, app_error.ErrCodeGeneralServerUnavailable)
	}

	filename := filepath.Base(key)
	if filename == "" || filename == "." {
		filename = "downloaded_file"
	}

	fileHeader := NewCustomFileHeader(
		filename,
		int64(len(fileContent)),
		make(map[string][]string),
		fileContent,
	)

	if result.ContentType != nil {
		fileHeader.Header["Content-Type"] = []string{*result.ContentType}
	}

	return fileHeader, nil
}
