package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type ReviewCommentHandler struct {
	reviewCommentService services.ReviewCommentService
	log                  *log.Logger
	authContext          middleware.AuthContext
	validator            utils.Validator
}

func NewReviewCommentHandler(
	l *log.Logger,
	reviewCommentService services.ReviewCommentService,
	authContext middleware.AuthContext,
	validator utils.Validator,
) *ReviewCommentHandler {
	return &ReviewCommentHandler{
		log:                  l,
		reviewCommentService: reviewCommentService,
		authContext:          authContext,
		validator:            validator,
	}
}

// CreateReviewComment godoc
// @Summary Create review comment
// @Description Create a review comment for a session
// @Tags Review Comments
// @Accept json
// @Produce json
// @Param request body entities.CreateReviewCommentReq true "Create review comment request"
// @Security BearerAuth
// @Success 204 "Review comment created successfully"
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /review-comments [post]
func (h *ReviewCommentHandler) CreateReviewComment(c *gin.Context) {
	ctx := c.Request.Context()
	h.log.InfoWithID(ctx, "[Handler: CreateReviewComment] Called")

	var req entities.CreateReviewCommentReq
	if err := h.validator.ValidateAndBind(c, &req, "CreateReviewComment"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateReviewComment] Error validating request", err)
		utils.RespondWithError(c, err)
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateReviewComment] Error getting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	if err := h.reviewCommentService.CreateReviewComment(ctx, &req); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: CreateReviewComment] Error creating review comment", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
