package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	config "gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
)

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

			svc := NewEvaluationScoresRepository(lgr, mockStore, cfg)

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
		name           string
		input          *CreateEvaluationOverallSummaryTxReq
		serverResponse func(w http.ResponseWriter, r *http.Request)
		verify         func(t *testing.T, gotResp *CreateEvaluationOverallSummaryTxResp, gotErr error)
	}{
		{
			name:  "Success",
			input: validReq,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/feedback-and-score/overall-summary", r.URL.Path)
				assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var receivedReq CreateEvaluationOverallSummaryTxReq
				err = json.Unmarshal(body, &receivedReq)
				assert.NoError(t, err)

				assert.Equal(t, []string{"test"}, receivedReq.SummaryMd)

				mockResponse := CreateEvaluationOverallSummaryTxResp{
					OverallSummaryMd: "Hello Unit Testing",
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(mockResponse)
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
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/feedback-and-score/overall-summary", r.URL.Path)
				assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var receivedReq CreateEvaluationOverallSummaryTxReq
				err = json.Unmarshal(body, &receivedReq)
				assert.NoError(t, err)

				assert.Equal(t, []string{"test"}, receivedReq.SummaryMd)

				mockResponse := CreateEvaluationOverallSummaryTxResp{
					OverallSummaryMd: "Hello Unit Testing",
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(mockResponse)
			},
			verify: func(t *testing.T, gotResp *CreateEvaluationOverallSummaryTxResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - UnmarshalFailed",
			input: validReq,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/feedback-and-score/overall-summary", r.URL.Path)
				assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var receivedReq CreateEvaluationOverallSummaryTxReq
				err = json.Unmarshal(body, &receivedReq)
				assert.NoError(t, err)

				assert.Equal(t, []string{"test"}, receivedReq.SummaryMd)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode("Invalid JSON")
			},
			verify: func(t *testing.T, gotResp *CreateEvaluationOverallSummaryTxResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tC.serverResponse))
			defer server.Close()

			testConfig := &config.Config{
				InterviewSessionConfig: config.InterviewSessionConfig{
					InterviewAgentURL: server.URL,
				},
			}

			mockStore := new(mockSqlc.MockStore)

			repo := NewEvaluationScoresRepository(lgr, mockStore, testConfig)
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
		name           string
		input          *InterviewFeedbackAndScoreReq
		serverResponse func(w http.ResponseWriter, r *http.Request)
		verify         func(t *testing.T, gotResp *InterviewFeedbackAndScoreResp, gotErr error)
	}{
		{
			name:  "Success",
			input: validReq,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/feedback-and-score", r.URL.Path)
				assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var receivedReq InterviewFeedbackAndScoreReq
				err = json.Unmarshal(body, &receivedReq)
				assert.NoError(t, err)

				assert.Equal(t, userMessage, receivedReq.UserMessage)
				assert.Equal(t, interviewerMessage, receivedReq.InterviewerMessage)
				assert.Equal(t, rubricName, receivedReq.RubricName)
				assert.Equal(t, rubricDescriptionMd, receivedReq.RubricDescriptionMd)
				assert.Equal(t, criteria, receivedReq.Criteria)

				mockResponse := InterviewFeedbackAndScoreResp{
					OverallScore:    5.00,
					OverallFeedback: "Good performance overall",
					CriteriaScores: []CriteriaScore{
						{
							CriterionID:       "test",
							CriterionScore:    5,
							CriterionFeedback: "Good work",
						},
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(mockResponse)
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
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/feedback-and-score", r.URL.Path)
				assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
			},
			verify: func(t *testing.T, gotResp *InterviewFeedbackAndScoreResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
		{
			name:  "Error - WithUnmarshalFailed",
			input: validReq,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/feedback-and-score", r.URL.Path)
				assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var receivedReq InterviewFeedbackAndScoreReq
				err = json.Unmarshal(body, &receivedReq)
				assert.NoError(t, err)

				assert.Equal(t, userMessage, receivedReq.UserMessage)
				assert.Equal(t, interviewerMessage, receivedReq.InterviewerMessage)
				assert.Equal(t, rubricName, receivedReq.RubricName)
				assert.Equal(t, rubricDescriptionMd, receivedReq.RubricDescriptionMd)
				assert.Equal(t, criteria, receivedReq.Criteria)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode("Invalid JSON")
			},
			verify: func(t *testing.T, gotResp *InterviewFeedbackAndScoreResp, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, gotResp)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tC.serverResponse))
			defer server.Close()

			testConfig := &config.Config{
				InterviewSessionConfig: config.InterviewSessionConfig{
					InterviewAgentURL: server.URL,
				},
			}

			mockStore := new(mockSqlc.MockStore)

			repo := NewEvaluationScoresRepository(lgr, mockStore, testConfig)
			gotResp, gotErr := repo.InterviewFeedbackAndScore(ctx, tC.input)

			tC.verify(t, gotResp, gotErr)
		})
	}
}
