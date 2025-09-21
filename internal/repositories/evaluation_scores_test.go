package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	config "gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
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
			ID:          globalID,
			CriterionID: globalID,
			Score:       5,
			CommentMd:   "test",
		},
		{
			ID:          globalID,
			CriterionID: globalID,
			Score:       5,
			CommentMd:   "test",
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
