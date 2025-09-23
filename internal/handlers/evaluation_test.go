package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
				VersionLabel:  "test version label",
				Criteria: []entities.CriterionRow{
					{
						ID:            criteriaID.String(),
						Name:          "test name",
						DescriptionMd: "test description md",
						Weight:        "test weight",
						MaxScore:      "test max score",
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
				assert.JSONEq(t, `{"rubrics":[{"id":"`+resp.Rubrics[0].ID+`","name":"`+resp.Rubrics[0].Name+`","description_md":"`+resp.Rubrics[0].DescriptionMd+`","version_label":"`+resp.Rubrics[0].VersionLabel+`","criteria":[{"id":"`+resp.Rubrics[0].Criteria[0].ID+`","name":"`+resp.Rubrics[0].Criteria[0].Name+`","description_md":"`+resp.Rubrics[0].Criteria[0].DescriptionMd+`","weight":"`+resp.Rubrics[0].Criteria[0].Weight+`","max_score":"`+resp.Rubrics[0].Criteria[0].MaxScore+`"}]}]}`, w.Body.String())
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
