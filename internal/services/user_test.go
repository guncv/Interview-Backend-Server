package services

// func TestUserService_HealthCheck(t *testing.T) {
// 	lgr := log.Initialize(constants.TestAppEnv)
// 	ctx := context.Background()
// 	mockErr := errors.New("error")

// 	okResponse := entities.HealthCheckResponse{
// 		Status: "ok",
// 	}
// 	testCases := []struct {
// 		name   string
// 		setup  func() *repositories.MockUserRepository
// 		verify func(t *testing.T, got entities.HealthCheckResponse, gotErr error)
// 	}{
// 		{
// 			name: "OK",
// 			setup: func() *repositories.MockUserRepository {
// 				mockUserRepo := new(repositories.MockUserRepository)
// 				mockUserRepo.EXPECT().
// 					HealthCheck(ctx).
// 					Return("ok", nil)

// 				return mockUserRepo
// 			},
// 			verify: func(t *testing.T, got entities.HealthCheckResponse, gotErr error) {
// 				assert.Equal(t, okResponse, got)
// 				assert.NoError(t, gotErr)
// 			},
// 		},
// 		{
// 			name: "Error",
// 			setup: func() *repositories.MockUserRepository {
// 				mockUserRepo := new(repositories.MockUserRepository)
// 				mockUserRepo.EXPECT().
// 					HealthCheck(ctx).
// 					Return("", mockErr)

// 				return mockUserRepo
// 			},
// 			verify: func(t *testing.T, got entities.HealthCheckResponse, gotErr error) {
// 				assert.Equal(t, entities.HealthCheckResponse{}, got)
// 				assert.Error(t, gotErr)
// 				assert.ErrorIs(t, gotErr, mockErr)
// 			},
// 		},
// 	}

// 	for _, tC := range testCases {
// 		t.Run(tC.name, func(t *testing.T) {
// 			mockUserRepo := tC.setup()
// 			defer mockUserRepo.AssertExpectations(t)

// 			svc := NewUserService(lgr, mockUserRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
// 			got, gotErr := svc.HealthCheck(ctx)

// 			tC.verify(t, got, gotErr)
// 		})
// 	}
// }
