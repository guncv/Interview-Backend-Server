package utils

import (
	"errors"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Validator interface {
	ValidateAndBind(c *gin.Context, req interface{}, handlerName string) error
	GetValidate() *validator.Validate
}

type validatorImpl struct {
	validate *validator.Validate
	log      *log.Logger
}

func NewValidator(log *log.Logger) Validator {
	v := validator.New()

	// Register custom validation for file uploads
	v.RegisterValidation("file_required", validateFileRequired)
	v.RegisterValidation("file_optional", validateFileOptional)

	return &validatorImpl{
		validate: v,
		log:      log,
	}
}

// Custom validation for required file uploads
func validateFileRequired(fl validator.FieldLevel) bool {
	fieldName := fl.FieldName()
	fieldValue := fl.Field().Interface()

	if file, ok := fieldValue.(*multipart.FileHeader); ok {
		if file == nil {
			fmt.Printf("DEBUG: %s file is nil\n", fieldName)
			return false
		}
		if file.Size == 0 {
			fmt.Printf("DEBUG: %s file size is 0\n", fieldName)
			return false
		}
		fmt.Printf("DEBUG: %s file is valid, size: %d, filename: %s\n", fieldName, file.Size, file.Filename)
		return true
	}

	if file, ok := fieldValue.(multipart.FileHeader); ok {
		if file.Size == 0 {
			fmt.Printf("DEBUG: %s file size is 0\n", fieldName)
			return false
		}
		fmt.Printf("DEBUG: %s file is valid, size: %d, filename: %s\n", fieldName, file.Size, file.Filename)
		return true
	}

	if str, ok := fieldValue.(string); ok {
		fmt.Printf("DEBUG: %s field is string: %s\n", fieldName, str)
		return str != ""
	}

	if bytes, ok := fieldValue.([]byte); ok {
		fmt.Printf("DEBUG: %s field is []byte with length: %d\n", fieldName, len(bytes))
		return len(bytes) > 0
	}

	if bytes, ok := fieldValue.([]uint8); ok {
		fmt.Printf("DEBUG: %s field is []uint8 with length: %d\n", fieldName, len(bytes))
		return len(bytes) > 0
	}

	fmt.Printf("DEBUG: %s field is unknown type: %T\n", fieldName, fieldValue)
	return false
}

func validateFileOptional(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().Interface()

	if fieldValue == nil {
		return true
	}

	if file, ok := fieldValue.(*multipart.FileHeader); ok {
		if file == nil {
			return true
		}
		if file.Size == 0 {
			return true
		}
		if file.Filename == "" {
			return true
		}
		if file.Filename != "" && file.Size > 0 {
			contentType := file.Header.Get("Content-Type")
			if contentType != "" {
				validTypes := []string{
					"image/jpeg", "image/jpg", "image/png", "image/gif",
					"image/webp", "image/bmp", "image/svg+xml",
				}
				for _, validType := range validTypes {
					if contentType == validType {
						return true
					}
				}
				return false
			}
			return true
		}
	}
	return true
}

func (v *validatorImpl) ValidateAndBind(c *gin.Context, req interface{}, handlerName string) error {
	ctx := c.Request.Context()

	if err := c.ShouldBind(req); err != nil {
		v.log.ErrorWithID(ctx, fmt.Sprintf("[%s] Form data binding failed", handlerName), err)
		if err := c.ShouldBindJSON(req); err != nil {
			v.log.ErrorWithID(ctx, fmt.Sprintf("[%s] JSON binding also failed", handlerName), err)
			return app_error.New(err, app_error.ErrCodeAuthInvalidRequest)
		}
		v.log.InfoWithID(ctx, fmt.Sprintf("[%s] Using JSON binding as fallback", handlerName))
	} else {
		v.log.InfoWithID(ctx, fmt.Sprintf("[%s] Form data binding successful", handlerName))
	}

	if err := v.validate.Struct(req); err != nil {
		if validatorErrors, ok := err.(validator.ValidationErrors); ok {
			firstError := validatorErrors[0]
			errorMsg := getSimpleErrorMessage(firstError)
			v.log.ErrorWithID(ctx, fmt.Sprintf("[%s] Validation:", handlerName), errorMsg)
			return app_error.NewWithCustomMessage(errors.New(errorMsg), app_error.ErrCodeAuthInvalidRequest, errorMsg)
		}

		v.log.ErrorWithID(ctx, fmt.Sprintf("[%s] Validation failed", handlerName), err)
		return app_error.New(err, app_error.ErrCodeAuthInvalidRequest)
	}

	return nil
}

func (v *validatorImpl) GetValidate() *validator.Validate {
	return v.validate
}

func getSimpleErrorMessage(err validator.FieldError) string {
	field := err.Field()

	switch err.Tag() {
	case "required":
		fieldName := strings.ToLower(field)
		if field == "CategoryID" {
			fieldName = "category_id"
		}
		return fmt.Sprintf("%s is required", fieldName)
	case "file_required":
		fieldName := strings.ToLower(field)
		if field == "ThumbnailURL" {
			fieldName = "thumbnail_url"
		}
		return fmt.Sprintf("%s file is required", fieldName)
	case "file_optional":
		fieldName := strings.ToLower(field)
		if field == "ThumbnailURL" {
			fieldName = "thumbnail_url"
		}
		return fmt.Sprintf("%s must be a valid image file (JPEG, PNG, GIF, WebP, BMP, SVG)", fieldName)
	case "email":
		return "invalid email format"
	case "min":
		fieldName := strings.ToLower(field)
		if field == "CategoryID" {
			fieldName = "category_id"
		}
		return fmt.Sprintf("%s must be at least %s characters", fieldName, err.Param())
	case "max":
		fieldName := strings.ToLower(field)
		if field == "CategoryID" {
			fieldName = "category_id"
		}
		return fmt.Sprintf("%s is too long (max %s characters)", fieldName, err.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", strings.ToLower(field), err.Param())
	case "uuid":
		fieldName := strings.ToLower(field)
		if field == "CategoryID" {
			fieldName = "category_id"
		}
		return fmt.Sprintf("invalid %s format", fieldName)
	case "oneof":
		return fmt.Sprintf("invalid %s value", strings.ToLower(field))
	default:
		return fmt.Sprintf("invalid %s", strings.ToLower(field))
	}
}
