package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	http "gitlab.com/interview-simulation/interview-backend-server/internal/infras/http"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	config "gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
	mockHttp "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/http"
)

type mockResult struct {
	affected int64
}

func (r mockResult) LastInsertId() (int64, error) { return 0, nil }
func (r mockResult) RowsAffected() (int64, error) { return r.affected, nil }

func TestEvaluationScoresRepository_CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	globalID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	improvementSentenceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	improvementSentence := "test"
	llmModel := "test"
	evaluationID := globalID
	sessionID := globalID
	userID := globalID
	turnID := globalID
	rubricID := globalID
	currentState := "test"
	overallScore := "5"
	summaryMd := "test"
	createdAt := time.Now()
	updatedAt := time.Now()
	criteria := []CreateScoreTxReq{
		{
			ID:            globalID,
			CriterionID:   globalID,
			CriterionName: "test",
			Score:         5,
			CommentMd:     "test",
		},
		{
			ID:            globalID,
			CriterionID:   globalID,
			CriterionName: "test",
			Score:         5,
			CommentMd:     "test",
		},
	}

	input := &CreateEvaluationAndScoreTxReq{
		EvaluationID:      evaluationID,
		SessionID:         sessionID,
		UserID:            userID,
		TurnID:            turnID,
		RubricID:          rubricID,
		CurrentState:      currentState,
		OverallScore:      overallScore,
		SummaryMd:         summaryMd,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		Criteria:          criteria,
		ImproveSentenceID: improvementSentenceID,
		ImproveSentence:   improvementSentence,
		LLmModel:          llmModel,
	}

	testCases := []struct {
		name   string
		input  *CreateEvaluationAndScoreTxReq
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreateEvaluation
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							evaluationID,
							sessionID,
							turnID,
							rubricID,
							userID,
							currentState,
							overallScore,
							summaryMd,
							createdAt,
							sql.NullTime{Time: updatedAt, Valid: true},
						).
						Return(nil, nil).Once()

					// Mock CreateImproveSentence
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							improvementSentenceID,
							turnID,
							improvementSentence,
							llmModel,
							createdAt,
						).
						Return(nil, nil).Once()

					// Mock CreateEvaluationCriteriaScore
					mockDBTX.EXPECT().
						ExecContext(
							mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							criteria[0].ID,
							evaluationID,
							criteria[0].CriterionID,
							criteria[0].CriterionName,
							int32(criteria[0].Score),
							criteria[0].CommentMd,
							createdAt,
							sql.NullTime{Time: updatedAt, Valid: true},
						).
						Return(nil, nil).Twice()

					fn(queries)
				}).Return(nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - CreateEvaluationError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							evaluationID,
							sessionID,
							turnID,
							rubricID,
							userID,
							currentState,
							overallScore,
							summaryMd,
							createdAt,
							sql.NullTime{Time: updatedAt, Valid: true},
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name:  "Error - CreateImproveSentenceError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreateEvaluation
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							evaluationID,
							sessionID,
							turnID,
							rubricID,
							userID,
							currentState,
							overallScore,
							summaryMd,
							createdAt,
							sql.NullTime{Time: updatedAt, Valid: true},
						).
						Return(nil, nil).Once()

					// Mock CreateImproveSentence
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							improvementSentenceID,
							turnID,
							improvementSentence,
							llmModel,
							createdAt,
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name:  "Error - CreateEvaluationCriteriaScoreError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							evaluationID,
							sessionID,
							turnID,
							rubricID,
							userID,
							currentState,
							overallScore,
							summaryMd,
							createdAt,
							sql.NullTime{Time: updatedAt, Valid: true},
						).
						Return(nil, nil).Once()

					// Mock CreateImproveSentence
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							improvementSentenceID,
							turnID,
							improvementSentence,
							llmModel,
							createdAt,
						).
						Return(nil, nil).Once()

					mockDBTX.EXPECT().
						ExecContext(
							mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							criteria[0].ID,
							evaluationID,
							criteria[0].CriterionID,
							criteria[0].CriterionName,
							int32(criteria[0].Score),
							criteria[0].CommentMd,
							createdAt,
							sql.NullTime{Time: updatedAt, Valid: true},
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockStore := tC.setup()

			defer func() {
				if mockStore != nil {
					mockStore.AssertExpectations(t)
				}
			}()

			cfg := &config.Config{}

			svc := NewEvaluationScoresRepository(lgr, mockStore, cfg, nil)

			gotErr := svc.CreateEvaluationWithCriteriaScoreAndImproveSentenceTx(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestEvaluationScoresRepository_GetEvaluationOverallSummary(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	validReq := &CreateEvaluationOverallSummaryTxReq{
		SummaryMd: []string{"test"},
	}

	testCases := []struct {
		name      string
		input     *CreateEvaluationOverallSummaryTxReq
		mockSetup func(mockHTTPClient *mockHttp.MockHTTPClient)
		verify    func(t *testing.T, gotResp *CreateEvaluationOverallSummaryTxResp, gotErr error)
	}{
		{
			name:  "Success",
			input: validReq,
			mockSetup: func(mockHTTPClient *mockHttp.MockHTTPClient) {
				expectedConfig := http.HTTPClientConfig{
					BaseURL:   "http://test-url/api/v1/feedback-and-score/overall-summary",
					Method:    "POST",
					LogPrefix: "GetEvaluationOverallSummary",
				}

				mockResponse := &http.HTTPClientResponse{
					StatusCode: 200,
					Body:       []byte(`{"overall_summary_md":"Hello Unit Testing"}`),
					Headers:    make(map[string][]string),
				}

				mockHTTPClient.EXPECT().
					MakeJSONRequest(ctx, expectedConfig, validReq, mock.AnythingOfType("*repositories.CreateEvaluationOverallSummaryTxResp")).
					Run(func(ctx context.Context, config http.HTTPClientConfig, requestBody interface{}, responseBody interface{}) {
						// Simulate unmarshaling the response
						resp := responseBody.(*CreateEvaluationOverallSummaryTxResp)
						resp.OverallSummaryMd = "Hello Unit Testing"
					}).
					Return(mockResponse, nil)
			},
			verify: func(t *testing.T, gotResp *CreateEvaluationOverallSummaryTxResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, "Hello Unit Testing", gotResp.OverallSummaryMd)
			},
		},
		{
			name:  "Error - HTTPError",
			input: validReq,
			mockSetup: func(mockHTTPClient *mockHttp.MockHTTPClient) {
				expectedConfig := http.HTTPClientConfig{
					BaseURL:   "http://test-url/api/v1/feedback-and-score/overall-summary",
					Method:    "POST",
					LogPrefix: "GetEvaluationOverallSummary",
				}

				mockHTTPClient.EXPECT().
					MakeJSONRequest(ctx, expectedConfig, validReq, mock.AnythingOfType("*repositories.CreateEvaluationOverallSummaryTxResp")).
					Return(nil, errors.New("third-party service returned status: 500 Internal Server Error"))
			},
			verify: func(t *testing.T, gotResp *CreateEvaluationOverallSummaryTxResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "third-party service returned status: 500 Internal Server Error")
			},
		},
		{
			name:  "Error - UnmarshalFailed",
			input: validReq,
			mockSetup: func(mockHTTPClient *mockHttp.MockHTTPClient) {
				expectedConfig := http.HTTPClientConfig{
					BaseURL:   "http://test-url/api/v1/feedback-and-score/overall-summary",
					Method:    "POST",
					LogPrefix: "GetEvaluationOverallSummary",
				}

				mockResponse := &http.HTTPClientResponse{
					StatusCode: 200,
					Body:       []byte(`"Invalid JSON"`),
					Headers:    make(map[string][]string),
				}

				mockHTTPClient.EXPECT().
					MakeJSONRequest(ctx, expectedConfig, validReq, mock.AnythingOfType("*repositories.CreateEvaluationOverallSummaryTxResp")).
					Return(mockResponse, errors.New("failed to unmarshal response body"))
			},
			verify: func(t *testing.T, gotResp *CreateEvaluationOverallSummaryTxResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "failed to unmarshal response body")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockHTTPClient := mockHttp.NewMockHTTPClient(t)
			tC.mockSetup(mockHTTPClient)

			testConfig := &config.Config{
				InterviewSessionConfig: config.InterviewSessionConfig{
					InterviewAgentURL: "http://test-url",
				},
			}

			mockStore := new(mockSqlc.MockStore)

			repo := NewEvaluationScoresRepository(lgr, mockStore, testConfig, mockHTTPClient)
			gotResp, gotErr := repo.GetEvaluationOverallSummary(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestEvaluationScoresRepository_InterviewFeedbackAndScore(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	userMessage := "test"
	interviewerMessage := "test"
	rubricName := "test"
	rubricDescriptionMd := "test"
	criteria := []entities.CritetiaRow{
		{
			CriterionID:            "test",
			CriterionCode:          "test",
			CriterionName:          "test",
			CriterionDescriptionMd: "test",
			CriterionWeight:        "test",
			CriterionMaxScore:      "test",
		},
	}
	validReq := &InterviewFeedbackAndScoreReq{
		UserMessage:         userMessage,
		InterviewerMessage:  interviewerMessage,
		RubricName:          rubricName,
		RubricDescriptionMd: rubricDescriptionMd,
		Criteria:            criteria,
	}

	testCases := []struct {
		name      string
		input     *InterviewFeedbackAndScoreReq
		mockSetup func(mockHTTPClient *mockHttp.MockHTTPClient)
		verify    func(t *testing.T, gotResp *InterviewFeedbackAndScoreResp, gotErr error)
	}{
		{
			name:  "Success",
			input: validReq,
			mockSetup: func(mockHTTPClient *mockHttp.MockHTTPClient) {
				expectedConfig := http.HTTPClientConfig{
					BaseURL:   "http://test-url/api/v1/feedback-and-score",
					Method:    "POST",
					LogPrefix: "InterviewFeedbackAndScore",
				}

				mockResponse := &http.HTTPClientResponse{
					StatusCode: 200,
					Body:       []byte(`{"overall_score":5.00,"overall_feedback":"Good performance overall","criteria_scores":[{"criterion_id":"test","criterion_score":5,"criterion_feedback":"Good work"}]}`),
					Headers:    make(map[string][]string),
				}

				mockHTTPClient.EXPECT().
					MakeJSONRequest(ctx, expectedConfig, validReq, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreResp")).
					Run(func(ctx context.Context, config http.HTTPClientConfig, requestBody interface{}, responseBody interface{}) {
						resp := responseBody.(*InterviewFeedbackAndScoreResp)
						resp.OverallScore = 5.00
						resp.OverallFeedback = "Good performance overall"
						resp.CriteriaScores = []CriteriaScore{
							{
								CriterionID:       "test",
								CriterionScore:    5,
								CriterionFeedback: "Good work",
							},
						}
					}).
					Return(mockResponse, nil)
			},
			verify: func(t *testing.T, gotResp *InterviewFeedbackAndScoreResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, 5.00, gotResp.OverallScore)
				assert.Equal(t, "Good performance overall", gotResp.OverallFeedback)
				assert.Len(t, gotResp.CriteriaScores, 1)
				assert.Equal(t, "test", gotResp.CriteriaScores[0].CriterionID)
				assert.Equal(t, 5, gotResp.CriteriaScores[0].CriterionScore)
				assert.Equal(t, "Good work", gotResp.CriteriaScores[0].CriterionFeedback)
			},
		},
		{
			name:  "Error - WithPostFeedbackAndScoreFailed",
			input: validReq,
			mockSetup: func(mockHTTPClient *mockHttp.MockHTTPClient) {
				expectedConfig := http.HTTPClientConfig{
					BaseURL:   "http://test-url/api/v1/feedback-and-score",
					Method:    "POST",
					LogPrefix: "InterviewFeedbackAndScore",
				}

				mockHTTPClient.EXPECT().
					MakeJSONRequest(ctx, expectedConfig, validReq, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreResp")).
					Return(nil, errors.New("third-party service returned status: 500 Internal Server Error"))
			},
			verify: func(t *testing.T, gotResp *InterviewFeedbackAndScoreResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "third-party service returned status: 500 Internal Server Error")
			},
		},
		{
			name:  "Error - WithUnmarshalFailed",
			input: validReq,
			mockSetup: func(mockHTTPClient *mockHttp.MockHTTPClient) {
				expectedConfig := http.HTTPClientConfig{
					BaseURL:   "http://test-url/api/v1/feedback-and-score",
					Method:    "POST",
					LogPrefix: "InterviewFeedbackAndScore",
				}

				mockResponse := &http.HTTPClientResponse{
					StatusCode: 200,
					Body:       []byte(`"Invalid JSON"`),
					Headers:    make(map[string][]string),
				}

				mockHTTPClient.EXPECT().
					MakeJSONRequest(ctx, expectedConfig, validReq, mock.AnythingOfType("*repositories.InterviewFeedbackAndScoreResp")).
					Return(mockResponse, errors.New("failed to unmarshal response body"))
			},
			verify: func(t *testing.T, gotResp *InterviewFeedbackAndScoreResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "failed to unmarshal response body")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockHTTPClient := mockHttp.NewMockHTTPClient(t)
			tC.mockSetup(mockHTTPClient)

			testConfig := &config.Config{
				InterviewSessionConfig: config.InterviewSessionConfig{
					InterviewAgentURL: "http://test-url",
				},
			}

			mockStore := new(mockSqlc.MockStore)

			repo := NewEvaluationScoresRepository(lgr, mockStore, testConfig, mockHTTPClient)
			gotResp, gotErr := repo.InterviewFeedbackAndScore(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestEvaluationScoresRepository_IsLastUserStateTurnScored(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionId := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	currentState := "Greeting"

	req := db.IsLastUserStateTurnScoredParams{
		SessionID:    sessionId,
		CurrentState: currentState,
	}

	testCases := []struct {
		name   string
		input  *db.IsLastUserStateTurnScoredParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotResp bool, gotErr error)
	}{
		{
			name:  "Success",
			input: &req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					IsLastUserStateTurnScored(ctx, req).
					Return(sql.NullBool{Bool: true, Valid: true}, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotResp bool, gotErr error) {
				assert.NoError(t, gotErr)
				assert.True(t, gotResp)
			},
		},
		{
			name:  "Error - WithIsLastUserStateTurnScoredNotFound",
			input: &req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					IsLastUserStateTurnScored(ctx, req).
					Return(sql.NullBool{Bool: false, Valid: true}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, gotResp bool, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0602]")
				assert.Contains(t, gotErr.Error(), "The interview turns were not found. Please try again.")
				assert.False(t, gotResp)
			},
		},
		{
			name:  "Error - WithIsLastUserStateTurnScoredError",
			input: &req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					IsLastUserStateTurnScored(ctx, req).
					Return(sql.NullBool{Bool: false, Valid: true}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotResp bool, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
				assert.False(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockStore := tC.setup()

			defer func() {
				if mockStore != nil {
					mockStore.AssertExpectations(t)
				}
			}()

			cfg := &config.Config{}

			svc := NewEvaluationScoresRepository(lgr, mockStore, cfg, nil)

			gotResp, gotErr := svc.IsLastUserStateTurnScored(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestEvaluationScoresRepository_GetEvaluationSummaryJsonBySessionAndState(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	sessionId := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	currentState := "Greeting"

	req := db.GetEvaluationSummaryJsonBySessionAndStateParams{
		SessionID:    sessionId,
		CurrentState: currentState,
	}

	testCases := []struct {
		name   string
		input  *db.GetEvaluationSummaryJsonBySessionAndStateParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotResp json.RawMessage, gotErr error)
	}{
		{
			name:  "Success",
			input: &req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, req).
					Return(json.RawMessage([]byte("test")), nil)

				return mockStore
			},
			verify: func(t *testing.T, gotResp json.RawMessage, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, json.RawMessage([]byte("test")), gotResp)
			},
		},
		{
			name:  "Error - WithGetEvaluationSummaryJsonBySessionAndStateNotFound",
			input: &req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, req).
					Return(nil, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, gotResp json.RawMessage, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS1000]")
				assert.Contains(t, gotErr.Error(), "The evaluation summary was not found. Please try again.")
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - WithGetEvaluationSummaryJsonBySessionAndStateError",
			input: &req,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					GetEvaluationSummaryJsonBySessionAndState(ctx, req).
					Return(json.RawMessage([]byte("test")), errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotResp json.RawMessage, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[INS0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockStore := tC.setup()

			defer func() {
				if mockStore != nil {
					mockStore.AssertExpectations(t)
				}
			}()

			cfg := &config.Config{}

			svc := NewEvaluationScoresRepository(lgr, mockStore, cfg, nil)

			gotResp, gotErr := svc.GetEvaluationSummaryJsonBySessionAndState(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestEvaluationScoresRepository_CalculateEachCriteriaCommentBySessionAndState(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	validReq := &PreProcessedCriteriaReq{
		Criteria: []PreProcessedCriteria{
			{
				CriteriaID:       "test",
				CriteriaName:     "test",
				CriteriaAvgScore: 0,
				CriteriaComment:  []string{"test"},
			},
		},
	}

	mockResp := []byte(`{"criteria":[{"criteria_id":"test","criteria_name":"test","criteria_avg_score":0,"criteria_comment":"test"}]}`)

	validResp := &PostProcessedCriteriaResp{
		Criteria: []PostProcessedCriteria{
			{
				CriteriaID:       "test",
				CriteriaName:     "test",
				CriteriaAvgScore: 0,
				CriteriaComment:  "test",
			},
		},
	}

	testCases := []struct {
		name      string
		input     *PreProcessedCriteriaReq
		mockSetup func(mockHTTPClient *mockHttp.MockHTTPClient)
		verify    func(t *testing.T, gotResp *PostProcessedCriteriaResp, gotErr error)
	}{
		{
			name:  "Success",
			input: validReq,
			mockSetup: func(mockHTTPClient *mockHttp.MockHTTPClient) {
				expectedConfig := http.HTTPClientConfig{
					BaseURL:   "http://test-url/api/v1/feedback-and-score/criteria-comment",
					Method:    "POST",
					LogPrefix: "CalculateEachCriteriaCommentBySessionAndState",
				}

				mockResponse := &http.HTTPClientResponse{
					StatusCode: 200,
					Body:       mockResp,
					Headers:    make(map[string][]string),
				}

				mockHTTPClient.EXPECT().
					MakeJSONRequest(ctx, expectedConfig, validReq, mock.AnythingOfType("*repositories.PostProcessedCriteriaResp")).
					Run(func(ctx context.Context, config http.HTTPClientConfig, requestBody interface{}, responseBody interface{}) {
						resp := responseBody.(*PostProcessedCriteriaResp)
						resp.Criteria = []PostProcessedCriteria{
							{
								CriteriaID:       "test",
								CriteriaName:     "test",
								CriteriaAvgScore: 0,
								CriteriaComment:  "test",
							},
						}
					}).
					Return(mockResponse, nil)
			},
			verify: func(t *testing.T, gotResp *PostProcessedCriteriaResp, gotErr error) {
				assert.NoError(t, gotErr)
				assert.NotNil(t, gotResp)
				assert.Equal(t, validResp, gotResp)
			},
		},
		{
			name:  "Error - HTTPError",
			input: validReq,
			mockSetup: func(mockHTTPClient *mockHttp.MockHTTPClient) {
				expectedConfig := http.HTTPClientConfig{
					BaseURL:   "http://test-url/api/v1/feedback-and-score/criteria-comment",
					Method:    "POST",
					LogPrefix: "CalculateEachCriteriaCommentBySessionAndState",
				}

				mockHTTPClient.EXPECT().
					MakeJSONRequest(ctx, expectedConfig, validReq, mock.AnythingOfType("*repositories.PostProcessedCriteriaResp")).
					Return(nil, errors.New("third-party service returned status: 500 Internal Server Error"))
			},
			verify: func(t *testing.T, gotResp *PostProcessedCriteriaResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "third-party service returned status: 500 Internal Server Error")
			},
		},
		{
			name:  "Error - UnmarshalFailed",
			input: validReq,
			mockSetup: func(mockHTTPClient *mockHttp.MockHTTPClient) {
				expectedConfig := http.HTTPClientConfig{
					BaseURL:   "http://test-url/api/v1/feedback-and-score/criteria-comment",
					Method:    "POST",
					LogPrefix: "CalculateEachCriteriaCommentBySessionAndState",
				}

				mockResponse := &http.HTTPClientResponse{
					StatusCode: 200,
					Body:       []byte(`"Invalid JSON"`),
					Headers:    make(map[string][]string),
				}

				mockHTTPClient.EXPECT().
					MakeJSONRequest(ctx, expectedConfig, validReq, mock.AnythingOfType("*repositories.PostProcessedCriteriaResp")).
					Return(mockResponse, errors.New("failed to unmarshal response body"))
			},
			verify: func(t *testing.T, gotResp *PostProcessedCriteriaResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
				assert.Contains(t, gotErr.Error(), "failed to unmarshal response body")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockHTTPClient := mockHttp.NewMockHTTPClient(t)
			tC.mockSetup(mockHTTPClient)

			testConfig := &config.Config{
				InterviewSessionConfig: config.InterviewSessionConfig{
					InterviewAgentURL: "http://test-url",
				},
			}

			mockStore := new(mockSqlc.MockStore)

			repo := NewEvaluationScoresRepository(lgr, mockStore, testConfig, mockHTTPClient)
			gotResp, gotErr := repo.CalculateEachCriteriaCommentBySessionAndState(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}

func TestEvaluationScoresRepository_CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	globalID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	evaluationID := globalID
	sessionID := globalID
	stateID := globalID
	stateName := "test"
	overallScore := 5.0
	criteria := CreateCriteriaReqTx{
		ID:               []uuid.UUID{globalID},
		EvaluationID:     []uuid.UUID{evaluationID},
		CriteriaID:       []uuid.UUID{globalID},
		CriteriaName:     []string{"test"},
		CriteriaAvgScore: []float64{5.0},
		CriteriaComment:  []string{"test"},
	}

	input := &CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx{
		EvaluationID: evaluationID,
		SessionID:    sessionID,
		StateID:      stateID,
		StateName:    stateName,
		OverallScore: overallScore,
		Criteria:     criteria,
	}

	testCases := []struct {
		name   string
		input  *CreatePhraseEvaluationAndCriteriaScoreWithIsScoredStateReqTx
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreatePhraseEvaluation
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							evaluationID,
							sessionID,
							stateID,
							stateName,
							overallScore,
						).
						Return(nil, nil).Once()

					// Mock CreateAllPhraseRubricScores
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("*pq.StringArray"),
							mock.AnythingOfType("*pq.Float64Array"),
							mock.AnythingOfType("*pq.StringArray"),
						).
						Return(nil, nil).Once()

					// Mock UpdateIsEvaluatedInterviewStateByID
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							stateID,
							sql.NullBool{Bool: true, Valid: true},
						).
						Return(mockResult{affected: 1}, nil).Once()

					fn(queries)
				}).Return(nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error WithCreatePhraseEvaluationError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreatePhraseEvaluation
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							evaluationID,
							sessionID,
							stateID,
							stateName,
							overallScore,
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, gotErr, errors.New("database error"))
			},
		},
		{
			name:  "Error WithCreateAllPhraseRubricScoresError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreatePhraseEvaluation
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							evaluationID,
							sessionID,
							stateID,
							stateName,
							overallScore,
						).
						Return(nil, nil).Once()

					// Mock CreateAllPhraseRubricScores
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("*pq.StringArray"),
							mock.AnythingOfType("*pq.Float64Array"),
							mock.AnythingOfType("*pq.StringArray"),
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, gotErr, errors.New("database error"))
			},
		},
		{
			name:  "Error WithUpdateIsEvaluatedInterviewStateByIDError",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreatePhraseEvaluation
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							evaluationID,
							sessionID,
							stateID,
							stateName,
							overallScore,
						).
						Return(nil, nil).Once()

					// Mock CreateAllPhraseRubricScores
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("*pq.StringArray"),
							mock.AnythingOfType("*pq.Float64Array"),
							mock.AnythingOfType("*pq.StringArray"),
						).
						Return(nil, nil).Once()

					// Mock UpdateIsEvaluatedInterviewStateByID
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							stateID,
							sql.NullBool{Bool: true, Valid: true},
						).
						Return(nil, errors.New("database error")).Once()

					fn(queries)
				}).Return(errors.New("database error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, gotErr, errors.New("database error"))
			},
		},
		{
			name:  "Error WithCreatePhraseEvaluationNotFound",
			input: input,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)

					// Mock CreatePhraseEvaluation
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							evaluationID,
							sessionID,
							stateID,
							stateName,
							overallScore,
						).
						Return(nil, nil).Once()

					// Mock CreateAllPhraseRubricScores
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("pq.GenericArray"),
							mock.AnythingOfType("*pq.StringArray"),
							mock.AnythingOfType("*pq.Float64Array"),
							mock.AnythingOfType("*pq.StringArray"),
						).
						Return(nil, nil).Once()

					// Mock UpdateIsEvaluatedInterviewStateByID
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType("context.backgroundCtx"),
							mock.AnythingOfType("string"),
							stateID,
							sql.NullBool{Bool: true, Valid: true},
						).
						Return(mockResult{affected: 0}, app_error.New(errors.New("interview state not found"), app_error.ErrCodeInterviewStateNotFound)).Once()

					fn(queries)
				}).Return(app_error.New(errors.New("interview state not found"), app_error.ErrCodeInterviewStateNotFound))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Equal(t, gotErr, app_error.New(errors.New("interview state not found"), app_error.ErrCodeInterviewStateNotFound))
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockStore := tC.setup()

			defer func() {
				if mockStore != nil {
					mockStore.AssertExpectations(t)
				}
			}()

			cfg := &config.Config{}

			svc := NewEvaluationScoresRepository(lgr, mockStore, cfg, nil)

			gotErr := svc.CreatePhraseEvaluationAndCriteriaScoreWithIsScoredState(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
