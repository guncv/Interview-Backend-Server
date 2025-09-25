package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	app_error "gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/services"
	"gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
)

func TestEvaluationHandler_ListAllRubricsAndCriteria(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()

	rubricID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	criteriaID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	resp := &entities.ListAllRubricsAndCriteriaResp{
		Rubrics: []entities.RubricAndCriteriaRow{
			{
				ID:            rubricID.String(),
				Name:          "test name",
				DescriptionMd: "test description md",
				Criteria: []entities.CriterionRow{
					{
						ID:            criteriaID.String(),
						Name:          "test name",
						DescriptionMd: "test description md",
						Percentage:    "test percentage",
						Color:         "test color",
					},
				},
			},
		},
	}

	tests := []struct {
		name   string
		setup  func() (*services.MockEvaluationService, *utils.MockValidator, *middleware.MockAuthContext)
		verify func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Success",
			setup: func() (*services.MockEvaluationService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockEvaluationService := new(services.MockEvaluationService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockEvaluationService.EXPECT().
					ListAllRubricsAndCriteria(ctx).
					Return(resp, nil)

				return mockEvaluationService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"rubrics":[{"id":"`+resp.Rubrics[0].ID+`","name":"`+resp.Rubrics[0].Name+`","description_md":"`+resp.Rubrics[0].DescriptionMd+`","criteria":[{"id":"`+resp.Rubrics[0].Criteria[0].ID+`","name":"`+resp.Rubrics[0].Criteria[0].Name+`","description_md":"`+resp.Rubrics[0].Criteria[0].DescriptionMd+`","percentage":"`+resp.Rubrics[0].Criteria[0].Percentage+`","color":"`+resp.Rubrics[0].Criteria[0].Color+`"}]}]}`, w.Body.String())
			},
		},
		{
			name: "Error WithExtractAuthContext",
			setup: func() (*services.MockEvaluationService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockEvaluationService := new(services.MockEvaluationService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				authErr := app_error.New(errors.New("extract auth context error"), app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockEvaluationService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
		},
		{
			name: "Error WithListAllRubricsAndCriteria",
			setup: func() (*services.MockEvaluationService, *utils.MockValidator, *middleware.MockAuthContext) {
				mockEvaluationService := new(services.MockEvaluationService)
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(errors.New("service error"), app_error.ErrCodeAuthInvalidRequest)
				mockEvaluationService.EXPECT().
					ListAllRubricsAndCriteria(ctx).
					Return(nil, serviceErr)

				return mockEvaluationService, mockValidator, mockAuthContext
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "Something went wrong with the request")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/evaluation/rubrics", nil)
			c.Request.Header.Set("Content-Type", "application/json")

			mockEvaluationService, mockValidator, mockAuthContext := tt.setup()
			defer func() {
				if mockEvaluationService != nil {
					mockEvaluationService.AssertExpectations(t)
				}
				if mockValidator != nil {
					mockValidator.AssertExpectations(t)
				}
				if mockAuthContext != nil {
					mockAuthContext.AssertExpectations(t)
				}
			}()

			handler := NewEvaluationHandler(mockEvaluationService, log, mockAuthContext, mockValidator)
			handler.ListAllRubricsAndCriteria(c)

			tt.verify(t, w)
		})
	}
}

func TestEvaluationHandler_GetPhraseEvaluationsWithCriteriaBySessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := log.Initialize("test")
	ctx := context.Background()
	sessionID := "123e4567-e89b-12d3-a456-426614174000"

	validResp := &entities.GetPhraseEvaluationsWithCriteriaResp{
		PhraseEvaluations: []entities.PhraseEvaluations{
			{
				StateID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440000").String(),
				StateName:    "test",
				OverallScore: 100,
				Criteria: []entities.PhraseEvaluationCriteria{
					{
						CriteriaID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440000").String(),
						CriteriaName:    "test",
						CriteriaScore:   100,
						CriteriaComment: "test",
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		sessionID      string
		setup          func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockEvaluationService)
		verify         func(t *testing.T, w *httptest.ResponseRecorder)
		expectedStatus int
	}{
		{
			name:      "Success",
			sessionID: sessionID,
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockEvaluationService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockEvaluationService := new(services.MockEvaluationService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				mockEvaluationService.EXPECT().
					GetPhraseEvaluationsWithCriteriaBySessionID(ctx, sessionID).
					Return(validResp, nil)

				return mockValidator, mockAuthContext, mockEvaluationService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)

				expectedBody, err := json.Marshal(validResp)
				assert.NoError(t, err)
				assert.JSONEq(t, string(expectedBody), w.Body.String())
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Error WithNoSessionID",
			sessionID: "",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockEvaluationService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockEvaluationService := new(services.MockEvaluationService)

				return mockValidator, mockAuthContext, mockEvaluationService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Contains(t, w.Body.String(), "The session ID is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Error WithValidateError",
			sessionID: "invalid-session-id",
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockEvaluationService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockEvaluationService := new(services.MockEvaluationService)

				mockValidate := validator.New()
				mockValidator.EXPECT().
					GetValidate().
					Return(mockValidate)

				return mockValidator, mockAuthContext, mockEvaluationService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)

				assert.Contains(t, w.Body.String(), "The session ID is invalid. Please try again.")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Error WithExtractAuthContext",
			sessionID: sessionID,
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockEvaluationService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockEvaluationService := new(services.MockEvaluationService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				authErr := app_error.New(errors.New("extract auth context error"), app_error.ErrCodeAuthInvalidHeader)
				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, authErr)

				return mockValidator, mockAuthContext, mockEvaluationService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, w.Code)

				assert.Contains(t, w.Body.String(), "Please log in to continue")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "Error WithGetPhraseEvaluationsWithCriteriaBySessionID",
			sessionID: sessionID,
			setup: func() (*utils.MockValidator, *middleware.MockAuthContext, *services.MockEvaluationService) {
				mockValidator := new(utils.MockValidator)
				mockAuthContext := new(middleware.MockAuthContext)
				mockEvaluationService := new(services.MockEvaluationService)

				realValidator := validator.New()

				mockValidator.EXPECT().
					GetValidate().
					Return(realValidator)

				mockAuthContext.EXPECT().
					ExtractAuthContext(mock.Anything).
					Return(ctx, nil)

				serviceErr := app_error.New(errors.New("service error"), app_error.ErrCodeAuthInvalidRequest)
				mockEvaluationService.EXPECT().
					GetPhraseEvaluationsWithCriteriaBySessionID(ctx, sessionID).
					Return(nil, serviceErr)

				return mockValidator, mockAuthContext, mockEvaluationService
			},
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, w.Code)

				assert.Contains(t, w.Body.String(), "Something went wrong with the request")
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			url := fmt.Sprintf("/api/v1/sessions/%s", tt.sessionID)
			c.Request = httptest.NewRequest(http.MethodGet, url, nil)
			c.Params = gin.Params{{Key: "session_id", Value: tt.sessionID}}

			mockValidator, mockAuthContext, mockEvaluationService := tt.setup()
			defer mockValidator.AssertExpectations(t)
			defer mockAuthContext.AssertExpectations(t)
			defer mockEvaluationService.AssertExpectations(t)

			handler := NewEvaluationHandler(mockEvaluationService, log, mockAuthContext, mockValidator)
			handler.GetPhraseEvaluationsWithCriteriaBySessionID(c)

			tt.verify(t, w)
		})
	}
}
