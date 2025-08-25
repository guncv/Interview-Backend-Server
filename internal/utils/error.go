package utils

import (
	"net/http"
	"strings"

	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func RespondWithError(ctx *gin.Context, err error) {
	appErr, ok := err.(*app_error.AppError)
	if ok {
		ctx.JSON(int(app_error.ErrorCodeToHttpCode[appErr.Code]), appErr)
		return
	}

	if validationErr, ok := err.(validator.ValidationErrors); ok {
		validationError := app_error.NewWithCustomMessage(
			err,
			app_error.ErrCodeAuthInvalidRequest,
			"Validation failed: "+formatValidationErrors(validationErr),
		)
		ctx.JSON(http.StatusBadRequest, validationError)
		return
	}

	if strings.Contains(err.Error(), "Field validation for") {
		validationError := app_error.NewWithCustomMessage(
			err,
			app_error.ErrCodeAuthInvalidRequest,
			"Validation failed: "+err.Error(),
		)
		ctx.JSON(http.StatusBadRequest, validationError)
		return
	}

	type errorResponseKey string
	const errorKey errorResponseKey = "error"

	errorResponse := map[errorResponseKey]string{
		errorKey: err.Error(),
	}
	ctx.JSON(http.StatusInternalServerError, errorResponse)
}

func formatValidationErrors(errors validator.ValidationErrors) string {
	var messages []string
	for _, err := range errors {
		messages = append(messages, err.Field()+" "+err.Tag())
	}
	return strings.Join(messages, ", ")
}
