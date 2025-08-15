package utils

import (
	"net/http"
	"strings"

	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func RespondWithError(ctx *gin.Context, err error) {
	// Check if it's already an AppError
	appErr, ok := err.(*app_error.AppError)
	if ok {
		ctx.JSON(int(app_error.ErrorCodeToHttpCode[appErr.Code]), appErr)
		return
	}

	// Check if it's a validation error
	if validationErr, ok := err.(validator.ValidationErrors); ok {
		// Convert validation errors to a proper error response
		validationError := app_error.NewWithCustomMessage(
			err,
			app_error.ErrCodeAuthInvalidRequest,
			"Validation failed: "+formatValidationErrors(validationErr),
		)
		ctx.JSON(http.StatusBadRequest, validationError)
		return
	}

	// Check if it's a single validation error
	if strings.Contains(err.Error(), "Field validation for") {
		validationError := app_error.NewWithCustomMessage(
			err,
			app_error.ErrCodeAuthInvalidRequest,
			"Validation failed: "+err.Error(),
		)
		ctx.JSON(http.StatusBadRequest, validationError)
		return
	}

	// Default case - internal server error
	ctx.JSON(http.StatusInternalServerError, err)
}

func formatValidationErrors(errors validator.ValidationErrors) string {
	var messages []string
	for _, err := range errors {
		messages = append(messages, err.Field()+" "+err.Tag())
	}
	return strings.Join(messages, ", ")
}
