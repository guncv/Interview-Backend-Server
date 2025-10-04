package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

type EvaluationHandler struct {
	evaluationService services.EvaluationService
	log               *log.Logger
	authContext       middleware.AuthContext
	validator         utils.Validator
}

func NewEvaluationHandler(
	evaluationService services.EvaluationService,
	log *log.Logger,
	authContext middleware.AuthContext,
	validator utils.Validator,
) *EvaluationHandler {
	return &EvaluationHandler{
		evaluationService: evaluationService,
		log:               log,
		authContext:       authContext,
		validator:         validator,
	}
}

// ListAllRubricsAndCriteria godoc
// @Summary List all rubrics and criteria
// @Description List all rubrics and criteria
// @Tags Evaluation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} gitlab_com_interview-simulation_interview-backend-server_internal_entities.ListAllRubricsAndCriteriaResp
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /evaluation/rubrics [get]
func (h *EvaluationHandler) ListAllRubricsAndCriteria(c *gin.Context) {
	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListAllRubricsAndCriteria] Error extracting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.evaluationService.ListAllRubricsAndCriteria(ctx)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: ListAllRubricsAndCriteria] Error listing all rubrics and criteria", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetPhraseEvaluationsWithCriteriaBySessionID godoc
// @Summary Get phrase evaluations with criteria by session ID
// @Description Get phrase evaluations with criteria by session ID
// @Tags Evaluation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param session_id path string true "Session ID"
// @Success 200 {object} gitlab_com_interview-simulation_interview-backend-server_internal_entities.GetPhraseEvaluationsWithCriteriaResp
// @Failure 400 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Validation error or business logic error"
// @Failure 401 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Unauthorized"
// @Failure 500 {object} gitlab_com_interview-simulation_interview-backend-server_internal_infras_app_error.AppError "Internal server error"
// @Router /evaluation/phrase-evaluations/{session_id} [get]
func (h *EvaluationHandler) GetPhraseEvaluationsWithCriteriaBySessionID(c *gin.Context) {
	ctx := c.Request.Context()

	sessionID := c.Param("session_id")
	if sessionID == "" {
		h.log.ErrorWithID(ctx, "[Handler: GetPhraseEvaluationsWithCriteriaBySessionID] Session ID is required")
		utils.RespondWithError(c, app_error.New(errors.New("session ID is required"), app_error.ErrCodeSessionInvalidSessionID))
		return
	}

	if err := h.validator.GetValidate().Var(sessionID, "required,uuid"); err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetPhraseEvaluationsWithCriteriaBySessionID] Invalid session ID", err)
		utils.RespondWithError(c, app_error.New(err, app_error.ErrCodeSessionInvalidSessionID))
		return
	}

	ctx, err := h.authContext.ExtractAuthContext(c)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetPhraseEvaluationsWithCriteriaBySessionID] Error extracting auth context", err)
		utils.RespondWithError(c, err)
		return
	}

	resp, err := h.evaluationService.GetPhraseEvaluationsWithCriteriaBySessionID(ctx, sessionID)
	if err != nil {
		h.log.ErrorWithID(ctx, "[Handler: GetPhraseEvaluationsWithCriteriaBySessionID] Error getting phrase evaluations with criteria by session ID", err)
		utils.RespondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
