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
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	mockSqlc "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/db/sqlc"
)

func TestUserRepository_CheckIsUserExistsByID(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	successResp := db.Users{
		ID:                 uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Email:              "test@test.com",
		PasswordHash:       "test",
		FullName:           "test",
		Country:            "test",
		Gender:             "test",
		DateOfBirth:        time.Now(),
		IsAdmin:            sql.NullBool{Bool: false, Valid: true},
		IsEmailVerified:    sql.NullBool{Bool: false, Valid: true},
		LastLoginAt:        sql.NullTime{Time: time.Now(), Valid: true},
		LoginAttemptCount:  sql.NullInt32{Int32: 0, Valid: true},
		IsSuspended:        sql.NullBool{Bool: false, Valid: true},
		LastLoginIp:        sql.NullString{String: "test", Valid: true},
		LastLoginUserAgent: sql.NullString{String: "test", Valid: true},
		CreatedAt:          sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt:          sql.NullTime{Time: time.Now(), Valid: true},
	}

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, got *db.Users, gotErr error)
	}{
		{
			name:  "Success - Check is user exists by ID",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					CheckIsUserExistsByID(ctx, successResp.ID).
					Return(successResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Users, gotErr error) {
				assert.Equal(t, got, &successResp)
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - Check is user exists by ID not found",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					CheckIsUserExistsByID(ctx, successResp.ID).
					Return(db.Users{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Users, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[ONX0206]")
				assert.Contains(t, gotErr.Error(), "We couldn't find your account.")
			},
		},
		{
			name:  "Error - Check is user exists by ID error",
			input: successResp.ID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					CheckIsUserExistsByID(ctx, successResp.ID).
					Return(db.Users{}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, got *db.Users, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[ONX0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
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

			svc := NewUserRepository(lgr, mockStore)
			got, gotErr := svc.CheckIsUserExistsByID(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserRepository_CheckIsEmailExists(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	successResp := db.Users{
		ID:                 uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Email:              "test@test.com",
		PasswordHash:       "test",
		FullName:           "test",
		Country:            "test",
		Gender:             "test",
		DateOfBirth:        time.Now(),
		IsAdmin:            sql.NullBool{Bool: false, Valid: true},
		IsEmailVerified:    sql.NullBool{Bool: false, Valid: true},
		LastLoginAt:        sql.NullTime{Time: time.Now(), Valid: true},
		LoginAttemptCount:  sql.NullInt32{Int32: 0, Valid: true},
		IsSuspended:        sql.NullBool{Bool: false, Valid: true},
		LastLoginIp:        sql.NullString{String: "test", Valid: true},
		LastLoginUserAgent: sql.NullString{String: "test", Valid: true},
		CreatedAt:          sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt:          sql.NullTime{Time: time.Now(), Valid: true},
	}

	testCases := []struct {
		name   string
		input  string
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, got *db.Users, gotErr error)
	}{
		{
			name:  "Success - Check is email exists",
			input: successResp.Email,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					CheckIsEmailExists(ctx, successResp.Email).
					Return(successResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Users, gotErr error) {
				assert.Equal(t, got, &successResp)
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - Check is email exists not found",
			input: successResp.Email,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					CheckIsEmailExists(ctx, successResp.Email).
					Return(db.Users{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Users, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[ONX0206]")
				assert.Contains(t, gotErr.Error(), "We couldn't find your account.")
			},
		},
		{
			name:  "Error - Check is email exists error",
			input: successResp.Email,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					CheckIsEmailExists(ctx, successResp.Email).
					Return(db.Users{}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, got *db.Users, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[ONX0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
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

			svc := NewUserRepository(lgr, mockStore)
			got, gotErr := svc.CheckIsEmailExists(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserRepository_UpdateUser(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	updateReq := &db.UpdateUserParams{
		ID:           uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Email:        "test@test.com",
		PasswordHash: "test",
		FullName:     "test",
		Country:      "test",
		Gender:       "test",
		DateOfBirth:  time.Now(),
	}

	successResp := db.Users{
		ID:                 uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Email:              "test@test.com",
		PasswordHash:       "test",
		FullName:           "test",
		Country:            "test",
		Gender:             "test",
		DateOfBirth:        time.Now(),
		IsAdmin:            sql.NullBool{Bool: false, Valid: true},
		IsEmailVerified:    sql.NullBool{Bool: false, Valid: true},
		LastLoginAt:        sql.NullTime{Time: time.Now(), Valid: true},
		LoginAttemptCount:  sql.NullInt32{Int32: 0, Valid: true},
		IsSuspended:        sql.NullBool{Bool: false, Valid: true},
		LastLoginIp:        sql.NullString{String: "test", Valid: true},
		LastLoginUserAgent: sql.NullString{String: "test", Valid: true},
		CreatedAt:          sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt:          sql.NullTime{Time: time.Now(), Valid: true},
	}

	testCases := []struct {
		name   string
		input  *db.UpdateUserParams
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, got *db.Users, gotErr error)
	}{
		{
			name:  "Success - Update user",
			input: updateReq,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateUser(ctx, *updateReq).
					Return(successResp, nil)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Users, gotErr error) {
				assert.Equal(t, got, &successResp)
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - Update user not found",
			input: updateReq,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateUser(ctx, *updateReq).
					Return(db.Users{}, sql.ErrNoRows)

				return mockStore
			},
			verify: func(t *testing.T, got *db.Users, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[ONX0206]")
				assert.Contains(t, gotErr.Error(), "We couldn't find your account.")
			},
		},
		{
			name:  "Error - Update user error",
			input: updateReq,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					UpdateUser(ctx, *updateReq).
					Return(db.Users{}, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, got *db.Users, gotErr error) {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
				assert.Contains(t, gotErr.Error(), "[ONX0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
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

			svc := NewUserRepository(lgr, mockStore)
			got, gotErr := svc.UpdateUser(ctx, tC.input)

			tC.verify(t, got, gotErr)
		})
	}
}

func TestUserRepository_VerifyEmail(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	userID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	testCases := []struct {
		name   string
		input  uuid.UUID
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success - Verify email",
			input: userID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					VerifyEmail(ctx, userID).
					Return(1, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - Verify email not found",
			input: userID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					VerifyEmail(ctx, userID).
					Return(0, nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[ONX0206]")
				assert.Contains(t, gotErr.Error(), "We couldn't find your account.")
			},
		},
		{
			name:  "Error - Verify email error",
			input: userID,
			setup: func() *mockSqlc.MockStore {
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().
					VerifyEmail(ctx, userID).
					Return(0, errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[ONX0101]")
				assert.Contains(t, gotErr.Error(), "We're having trouble connecting to the server")
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

			svc := NewUserRepository(lgr, mockStore)
			gotErr := svc.VerifyEmail(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

func TestUserRepository_SignInUserByEmailAndPasswordTx(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	userReq := &SignInUserByEmailAndPasswordTxModel{
		Email:            "test@test.com",
		LastLoginAt:      time.Now(),
		SessionID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		UpdatedAt:        time.Now(),
		UserID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		UserAgent:        "test",
		IpAddress:        "test",
		RefreshTokenHash: "test",
		LastActive:       time.Now(),
		ExpiresAt:        time.Now(),
	}

	testCases := []struct {
		name   string
		input  *SignInUserByEmailAndPasswordTxModel
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: userReq,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)
					mockResult := dummyResult{affected: 1}

					// Mock SignInUserByEmailAndPassword
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.Email,
							sql.NullTime{Time: userReq.LastLoginAt, Valid: true},
							sql.NullString{String: userReq.IpAddress, Valid: true},
							sql.NullString{String: userReq.UserAgent, Valid: true},
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, nil).Once()

					// Mock CreateSession
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.SessionID,
							userReq.UserID,
							userReq.RefreshTokenHash,
							userReq.UserAgent,
							userReq.IpAddress,
							sql.NullTime{Time: userReq.LastActive, Valid: true},
							sql.NullTime{Time: userReq.ExpiresAt, Valid: true},
						).
						Return(mockResult, nil).Once()

					fn(queries)
				}).Return(nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - user or password is incorrect",
			input: userReq,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)
					mockResult := dummyResult{affected: 0}

					// Mock SignInUserByEmailAndPassword
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.Email,
							sql.NullTime{Time: userReq.LastLoginAt, Valid: true},
							sql.NullString{String: userReq.IpAddress, Valid: true},
							sql.NullString{String: userReq.UserAgent, Valid: true},
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, nil).Once()

					fn(queries)
				}).Return(app_error.New(errors.New("this email or password is incorrect"), app_error.ErrCodeAuthUserNotFound))

				return mockStore
			},

			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[ONX0206]")
				assert.Contains(t, gotErr.Error(), "We couldn't find your account.")
			},
		},
		{
			name:  "Error - SignInUserByEmailAndPassword error",
			input: userReq,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)
					mockResult := dummyResult{affected: 0}

					// Mock SignInUserByEmailAndPassword
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.Email,
							sql.NullTime{Time: userReq.LastLoginAt, Valid: true},
							sql.NullString{String: userReq.IpAddress, Valid: true},
							sql.NullString{String: userReq.UserAgent, Valid: true},
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, errors.New("error")).Once()

					fn(queries)
				}).Return(errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name:  "Error - CreateSession error",
			input: userReq,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)
					mockResult := dummyResult{affected: 1}

					// Mock SignInUserByEmailAndPassword
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.Email,
							sql.NullTime{Time: userReq.LastLoginAt, Valid: true},
							sql.NullString{String: userReq.IpAddress, Valid: true},
							sql.NullString{String: userReq.UserAgent, Valid: true},
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, nil).Once()

					// Mock CreateSession
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.SessionID,
							userReq.UserID,
							userReq.RefreshTokenHash,
							userReq.UserAgent,
							userReq.IpAddress,
							sql.NullTime{Time: userReq.LastActive, Valid: true},
							sql.NullTime{Time: userReq.ExpiresAt, Valid: true},
						).
						Return(mockResult, errors.New("error")).Once()

					fn(queries)
				}).Return(errors.New("error"))

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

			svc := NewUserRepository(lgr, mockStore)

			gotErr := svc.SignInUserByEmailAndPasswordTx(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}

type dummyResult struct {
	affected int64
}

func (r dummyResult) LastInsertId() (int64, error) { return 0, nil }
func (r dummyResult) RowsAffected() (int64, error) { return r.affected, nil }

func TestUserRepository_ResetUserPasswordAndUpdateResetTokenTx(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	userReq := &ResetUserPasswordTxModel{
		UserID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		PasswordHash: "test",
		ResetToken:   "test",
		UpdatedAt:    time.Now(),
	}

	testCases := []struct {
		name   string
		input  *ResetUserPasswordTxModel
		setup  func() *mockSqlc.MockStore
		verify func(t *testing.T, gotErr error)
	}{
		{
			name:  "Success",
			input: userReq,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)
					mockResult := dummyResult{affected: 1}

					// Mock ResetUserPassword
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.UserID,
							userReq.PasswordHash,
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, nil).Once()

					// Mock UpdateResetTokenUsed
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.ResetToken,
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, nil).Once()

					fn(queries)
				}).Return(nil)

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.NoError(t, gotErr)
			},
		},
		{
			name:  "Error - user not found",
			input: userReq,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)
					mockResult := dummyResult{affected: 0}

					// Mock ResetUserPassword
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.UserID,
							userReq.PasswordHash,
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, nil).Once()

					fn(queries)
				}).Return(app_error.New(errors.New("user not found"), app_error.ErrCodeAuthUserNotFound))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[ONX0206]")
				assert.Contains(t, gotErr.Error(), "We couldn't find your account.")
			},
		},
		{
			name:  "Error - ResetUserPassword error",
			input: userReq,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)
					mockResult := dummyResult{affected: 1}

					// Mock ResetUserPassword
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.UserID,
							userReq.PasswordHash,
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, errors.New("error")).Once()

					fn(queries)
				}).Return(errors.New("error"))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
			},
		},
		{
			name:  "Error - UpdateResetTokenUsed not found",
			input: userReq,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)
					mockResult := dummyResult{affected: 1}
					mockResult2 := dummyResult{affected: 0}

					// Mock ResetUserPassword
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.UserID,
							userReq.PasswordHash,
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, nil).Once()

					// Mock UpdateResetTokenUsed
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.ResetToken,
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult2, nil).Once()

					fn(queries)
				}).Return(app_error.New(errors.New("reset token not found"), app_error.ErrCodeAuthResetTokenNotFound))

				return mockStore
			},
			verify: func(t *testing.T, gotErr error) {
				assert.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), "[ONX0208]")
				assert.Contains(t, gotErr.Error(), "That reset link is invalid. Please request a new one.")
			},
		},
		{
			name:  "Error - UpdateResetTokenUsed error",
			input: userReq,
			setup: func() *mockSqlc.MockStore {
				mockDBTX := new(mockSqlc.MockDBTX)
				mockStore := new(mockSqlc.MockStore)

				mockStore.EXPECT().ExecTx(ctx, mock.MatchedBy(func(fn func(*db.Queries) error) bool {
					return true
				})).Run(func(ctx context.Context, fn func(*db.Queries) error) {
					queries := db.New(mockDBTX)
					mockResult := dummyResult{affected: 1}

					// Mock ResetUserPassword
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.UserID,
							userReq.PasswordHash,
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, nil).Once()

					// Mock UpdateResetTokenUsed
					mockDBTX.EXPECT().
						ExecContext(mock.AnythingOfType(
							"context.backgroundCtx"),
							mock.AnythingOfType("string"),
							userReq.ResetToken,
							sql.NullTime{Time: userReq.UpdatedAt, Valid: true},
						).
						Return(mockResult, errors.New("error")).Once()

					fn(queries)
				}).Return(errors.New("error"))

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

			svc := NewUserRepository(lgr, mockStore)

			gotErr := svc.ResetUserPasswordAndUpdateResetTokenTx(ctx, tC.input)

			tC.verify(t, gotErr)
		})
	}
}
