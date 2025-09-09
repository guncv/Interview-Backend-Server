package websocket_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gitlab.com/interview-simulation/interview-backend-server/internal/entities"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/app_error"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/websocket"
	mockDatabase "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/database"
	mockServices "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/services"
	mockUtils "gitlab.com/interview-simulation/interview-backend-server/internal/mocks/utils"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
)

func TestWebSocketServerLogic_SendMessageTypeSegmentStart(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()
	segmentMapping := uuid.New()
	sessionID := "test-session-123"
	segmentID := "test-segment-456"

	testCases := []struct {
		name    string
		payload []byte
		client  *websocket.Client
		setup   func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockServices.MockInterviewSessionService, func(context.Context, *websocket.Client), func(context.Context, *websocket.Client, any))
		verify  func(t *testing.T, disconnectCalled bool, errorSent bool, sentError interface{})
	}{
		{
			name: "Success - Valid segment start message",
			payload: func() []byte {
				payload, _ := json.Marshal(websocket.MsgSegmentStart{
					Type:      constants.WebSocketMessageTypeSegmentStart,
					SessionID: sessionID,
					SegmentID: segmentID,
				})
				return payload
			}(),
			client: &websocket.Client{
				SessionID:        sessionID,
				CurrentSegmentID: segmentID,
				StartSessionTime: time.Now(),
			},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockServices.MockInterviewSessionService, func(context.Context, *websocket.Client), func(context.Context, *websocket.Client, any)) {
				mockGen := mockUtils.NewMockGenerator(t)
				mockRedis := mockDatabase.NewMockRedisClient(t)
				mockInterviewSvc := mockServices.NewMockInterviewSessionService(t)

				mockGen.EXPECT().GenerateUUID(ctx).Return(segmentMapping)

				redisPayload := database.RedisPayload{
					Key:   fmt.Sprintf("%s%s", constants.RedisPrefixInterviewSegmentMapping, segmentID),
					Value: segmentMapping.String(),
					TTL:   constants.RedisTTLInterviewSegmentMapping,
				}

				mockRedis.EXPECT().Set(ctx, redisPayload).Return(nil)

				mockInterviewSvc.EXPECT().SetSessionStartTime(ctx, &entities.SetSessionStartTimeReq{
					SessionID: sessionID,
					StartedAt: utils.FormatSecondsToMMSS(time.Since(time.Now()).Seconds()),
				}).Return(nil)

				return mockGen, mockRedis, mockInterviewSvc, nil, nil
			},
			verify: func(t *testing.T, disconnectCalled bool, errorSent bool, sentError interface{}) {
				assert.False(t, disconnectCalled, "Should not disconnect on valid message")
				assert.False(t, errorSent, "Should not send error on valid message")
			},
		},
		{
			name:    "Error - Invalid JSON payload",
			payload: []byte("invalid json"),
			client:  &websocket.Client{},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockServices.MockInterviewSessionService, func(context.Context, *websocket.Client), func(context.Context, *websocket.Client, any)) {
				mockGen := mockUtils.NewMockGenerator(t)
				mockRedis := mockDatabase.NewMockRedisClient(t)
				mockInterviewSvc := mockServices.NewMockInterviewSessionService(t)

				disconnectFunc := func(ctx context.Context, client *websocket.Client) {}
				writeJSONFunc := func(ctx context.Context, client *websocket.Client, data any) {}

				return mockGen, mockRedis, mockInterviewSvc, disconnectFunc, writeJSONFunc
			},
			verify: func(t *testing.T, disconnectCalled bool, errorSent bool, sentError interface{}) {
				if errMsg, ok := sentError.(map[string]interface{}); ok {
					assert.Equal(t, constants.WebSocketMessageTypeError, errMsg["type"])
					assert.Equal(t, string(app_error.ErrCodeWebSocketInvalidSegmentStart), errMsg["code"])
				}
			},
		},
		{
			name: "Error - Empty segment ID",
			payload: func() []byte {
				payload, _ := json.Marshal(websocket.MsgSegmentStart{
					Type:      constants.WebSocketMessageTypeSegmentStart,
					SessionID: sessionID,
					SegmentID: "", // Empty segment ID
				})
				return payload
			}(),
			client: &websocket.Client{},
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockServices.MockInterviewSessionService, func(context.Context, *websocket.Client), func(context.Context, *websocket.Client, any)) {
				mockGen := mockUtils.NewMockGenerator(t)
				mockRedis := mockDatabase.NewMockRedisClient(t)
				mockInterviewSvc := mockServices.NewMockInterviewSessionService(t)

				disconnectFunc := func(ctx context.Context, client *websocket.Client) {}
				writeJSONFunc := func(ctx context.Context, client *websocket.Client, data any) {}

				return mockGen, mockRedis, mockInterviewSvc, disconnectFunc, writeJSONFunc
			},
			verify: func(t *testing.T, disconnectCalled bool, errorSent bool, sentError interface{}) {
				assert.True(t, disconnectCalled, "Should disconnect on empty segment ID")
				assert.True(t, errorSent, "Should send error on empty segment ID")
			},
		},
		{
			name: "Error - Session ID mismatch",
			payload: func() []byte {
				payload, _ := json.Marshal(websocket.MsgSegmentStart{
					Type:      constants.WebSocketMessageTypeSegmentStart,
					SessionID: "different-session-id",
					SegmentID: segmentID,
				})
				return payload
			}(),
			client: &websocket.Client{}, // sessionID will be empty, causing mismatch
			setup: func() (*mockUtils.MockGenerator, *mockDatabase.MockRedisClient, *mockServices.MockInterviewSessionService, func(context.Context, *websocket.Client), func(context.Context, *websocket.Client, any)) {
				mockGen := mockUtils.NewMockGenerator(t)
				mockRedis := mockDatabase.NewMockRedisClient(t)
				mockInterviewSvc := mockServices.NewMockInterviewSessionService(t)

				disconnectFunc := func(ctx context.Context, client *websocket.Client) {
				}

				writeJSONFunc := func(ctx context.Context, client *websocket.Client, data any) {
				}

				return mockGen, mockRedis, mockInterviewSvc, disconnectFunc, writeJSONFunc
			},
			verify: func(t *testing.T, disconnectCalled bool, errorSent bool, sentError interface{}) {
				assert.True(t, disconnectCalled, "Should disconnect on session ID mismatch")
				assert.True(t, errorSent, "Should send error on session ID mismatch")
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			mockGen, mockRedis, mockInterviewSvc, disconnectFunc, writeJSONFunc := tC.setup()
			clientManager := websocket.NewClientManager(lgr)

			var disconnectCalled bool
			var errorSent bool
			var sentError interface{}

			wrappedDisconnectFunc := func(ctx context.Context, client *websocket.Client) {
				disconnectCalled = true
				disconnectFunc(ctx, client)
			}

			wrappedWriteJSONFunc := func(ctx context.Context, client *websocket.Client, data any) {
				errorSent = true
				sentError = data
				writeJSONFunc(ctx, client, data)
			}

			defer func() {
				if mockGen != nil {
					mockGen.AssertExpectations(t)
				}
				if mockRedis != nil {
					mockRedis.AssertExpectations(t)
				}
				if mockInterviewSvc != nil {
					mockInterviewSvc.AssertExpectations(t)
				}
			}()

			logic := websocket.NewWebSocketServerLogic(
				lgr,
				wrappedDisconnectFunc,
				wrappedWriteJSONFunc,
				clientManager,
				mockInterviewSvc,
				mockRedis,
				mockGen,
			)

			logic.SendMessageTypeSegmentStart(ctx, tC.client, tC.payload)

			tC.verify(t, disconnectCalled, errorSent, sentError)
		})
	}
}
