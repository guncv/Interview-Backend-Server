package utils

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Validator interface {
	ValidateAndBind(c *gin.Context, req interface{}, handlerName string) error
	GetValidate() *validator.Validate
	IsAllowedResumeContentType(ctx context.Context, fileHeader *multipart.FileHeader) bool
}

type validatorImpl struct {
	validate *validator.Validate
	log      *log.Logger
}

func NewValidator(log *log.Logger) Validator {
	v := validator.New()

	v.RegisterValidation("file_required", validateFileRequired)
	v.RegisterValidation("file_optional", validateFileOptional)
	v.RegisterValidation("valid_date_time", validateDateTime)
	v.RegisterValidation("valid_email", validateEmail)

	return &validatorImpl{
		validate: v,
		log:      log,
	}
}

// Custom email validation function
func validateEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()

	// Check if email is empty (handled by required tag)
	if email == "" {
		return true
	}

	// Basic email format validation using regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return false
	}

	// Split email into local and domain parts
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	localPart := parts[0]
	domain := parts[1]

	// Check local part doesn't start or end with dot
	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return false
	}

	// Check for consecutive dots in local part
	if strings.Contains(localPart, "..") {
		return false
	}

	// Check domain doesn't start or end with dot
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	// Check domain doesn't start or end with hyphen
	if strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") {
		return false
	}

	// Check for consecutive dots in domain
	if strings.Contains(domain, "..") {
		return false
	}

	// Check for consecutive hyphens in domain
	if strings.Contains(domain, "--") {
		return false
	}

	// Check domain has valid TLD
	domainParts := strings.Split(domain, ".")
	if len(domainParts) < 2 {
		return false
	}

	// TLD should be at least 2 characters
	tld := domainParts[len(domainParts)-1]
	if len(tld) < 2 {
		return false
	}

	// Check that no domain part is empty
	for _, part := range domainParts {
		if part == "" {
			return false
		}
	}

	return true
}

func validateDateTime(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().Interface()

	if date, ok := fieldValue.(time.Time); ok {
		if date.IsZero() {
			return false
		}

		if date.After(time.Now()) {
			return false
		}

		minDate := time.Now().AddDate(-150, 0, 0)
		return !date.Before(minDate)
	}

	return false
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

// getSpecificBindingErrorMessage provides specific error messages for common binding failures
func (v *validatorImpl) getSpecificBindingErrorMessage(err error) string {
	errStr := err.Error()

	// Date/time parsing errors
	if strings.Contains(errStr, "time") || strings.Contains(errStr, "date") {
		return "invalid date format. Please use ISO 8601 format (YYYY-MM-DD)"
	}

	// Type conversion errors - check this before JSON errors to avoid conflicts
	if strings.Contains(errStr, "cannot unmarshal") {
		if strings.Contains(errStr, "string") && strings.Contains(errStr, "bool") {
			return "invalid data type. Expected boolean value but received string"
		}
		if strings.Contains(errStr, "string") && strings.Contains(errStr, "int") {
			return "invalid data type. Expected number but received string"
		}
		if strings.Contains(errStr, "number") && strings.Contains(errStr, "string") {
			return "invalid data type. Expected string but received number"
		}
		return "invalid data type. One or more fields have incorrect data types"
	}

	// JSON syntax errors
	if strings.Contains(errStr, "unexpected end") {
		return "invalid JSON format. Request body is incomplete or malformed"
	}
	if strings.Contains(errStr, "invalid character") {
		return "invalid JSON format. Check for syntax errors in your request body"
	}
	if strings.Contains(errStr, "json") && strings.Contains(errStr, "unmarshal") {
		return "invalid JSON format. One or more fields have incorrect data types"
	}

	// Field validation errors
	if strings.Contains(errStr, "field") {
		return "invalid field value. Please check the data types and formats of your input fields"
	}

	// Default generic message
	return "invalid request format. Please check your input data and try again"
}

func (v *validatorImpl) ValidateAndBind(c *gin.Context, req interface{}, handlerName string) error {
	ctx := c.Request.Context()

	// First try form data binding
	if err := c.ShouldBind(req); err != nil {
		v.log.ErrorWithID(ctx, fmt.Sprintf("[%s] Form data binding failed: %v", handlerName, err))

		// Try JSON binding as fallback
		if err := c.ShouldBindJSON(req); err != nil {
			v.log.ErrorWithID(ctx, fmt.Sprintf("[%s] JSON binding also failed: %v", handlerName, err))

			// Provide specific error messages for common binding failures
			errMsg := v.getSpecificBindingErrorMessage(err)
			return app_error.NewWithCustomMessage(
				err,
				app_error.ErrCodeAuthInvalidRequest,
				errMsg,
			)
		}
		v.log.InfoWithID(ctx, fmt.Sprintf("[%s] Using JSON binding as fallback", handlerName))
	} else {
		v.log.InfoWithID(ctx, fmt.Sprintf("[%s] Form data binding successful", handlerName))
	}

	// Log the request data for debugging
	v.log.InfoWithID(ctx, fmt.Sprintf("[%s] Request data: %+v", handlerName, req))

	// Validate the struct
	if err := v.validate.Struct(req); err != nil {
		if validatorErrors, ok := err.(validator.ValidationErrors); ok {
			// Log all validation errors for debugging
			for i, validationErr := range validatorErrors {
				v.log.ErrorWithID(ctx, fmt.Sprintf("[%s] Validation error %d: Field=%s, Tag=%s, Value=%v, Param=%s",
					handlerName, i+1, validationErr.Field(), validationErr.Tag(), validationErr.Value(), validationErr.Param()))
			}

			// Return the first validation error with a clear message
			firstError := validatorErrors[0]
			errorMsg := getSimpleErrorMessage(firstError)
			v.log.ErrorWithID(ctx, fmt.Sprintf("[%s] Validation failed: %v", handlerName, errorMsg))
			return app_error.NewWithCustomMessage(errors.New(errorMsg), app_error.ErrCodeAuthInvalidRequest, errorMsg)
		}

		v.log.ErrorWithID(ctx, fmt.Sprintf("[%s] Validation failed with non-validation error: %v", handlerName, err))
		return app_error.NewWithCustomMessage(
			err,
			app_error.ErrCodeAuthInvalidRequest,
			"validation failed. Please check your input data.",
		)
	}

	v.log.InfoWithID(ctx, fmt.Sprintf("[%s] Validation successful", handlerName))
	return nil
}

func (v *validatorImpl) GetValidate() *validator.Validate {
	return v.validate
}

func getSimpleErrorMessage(err validator.FieldError) string {
	field := err.Field()
	fieldName := strings.ToLower(field)

	// Handle special field name mappings
	switch field {
	case "CategoryID":
		fieldName = "category_id"
	case "DateOfBirth":
		fieldName = "date of birth"
	case "FullName":
		fieldName = "full name"
	case "ThumbnailURL":
		fieldName = "thumbnail_url"
	}

	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fieldName)
	case "file_required":
		return fmt.Sprintf("%s file is required", fieldName)
	case "file_optional":
		return fmt.Sprintf("%s must be a valid image file (JPEG, PNG, GIF, WebP, BMP, SVG)", fieldName)
	case "email":
		return "invalid email format"
	case "valid_email":
		return "invalid email format. Please provide a valid email address (e.g., user@example.com)"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fieldName, err.Param())
	case "max":
		return fmt.Sprintf("%s is too long (max %s characters)", fieldName, err.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", fieldName, err.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", fieldName, err.Param())
	case "uuid":
		return fmt.Sprintf("invalid %s format", fieldName)
	case "oneof":
		// Handle gender field specifically
		if field == "Gender" {
			return "gender must be one of: male, female, other"
		}
		return fmt.Sprintf("invalid %s value", fieldName)
	case "valid_date_time":
		return fmt.Sprintf("invalid %s. Date must be in the past and not more than 150 years ago. Please use YYYY-MM-DD format (e.g., 1990-01-15)", fieldName)
	case "datetime":
		return fmt.Sprintf("invalid %s format. Please use ISO 8601 format (YYYY-MM-DD)", fieldName)
	case "time":
		return fmt.Sprintf("invalid %s format. Please use ISO 8601 format (YYYY-MM-DD)", fieldName)
	case "date":
		return fmt.Sprintf("invalid %s format. Please use ISO 8601 format (YYYY-MM-DD)", fieldName)
	default:
		// For unknown validation tags, provide a more helpful message
		if field == "DateOfBirth" {
			return "invalid date of birth format. Please use YYYY-MM-DD format (e.g., 1990-01-15)"
		}
		return fmt.Sprintf("invalid %s", fieldName)
	}
}

func (v *validatorImpl) IsAllowedResumeContentType(ctx context.Context, fileHeader *multipart.FileHeader) bool {
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		v.log.ErrorWithID(ctx, "[Validator: isAllowedResumeContentType] Content type is empty")
		return false
	}
	for _, allowed := range constants.ResumeAllowContentTypes {
		if contentType == allowed {
			v.log.InfoWithID(ctx, fmt.Sprintf("[Validator: isAllowedResumeContentType] Content type %s is allowed", contentType))
			return true
		}
	}

	v.log.ErrorWithID(ctx, fmt.Sprintf("[Validator: isAllowedResumeContentType] Content type %s is not allowed", contentType))
	return false
}
