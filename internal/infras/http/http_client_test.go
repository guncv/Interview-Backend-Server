package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

func TestHTTPClient_MakeJSONRequest(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	testCases := []struct {
		name           string
		config         HTTPClientConfig
		requestBody    interface{}
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
		verifyResponse func(t *testing.T, response *HTTPClientResponse, responseBody interface{})
	}{
		{
			name: "Success - with request body",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestRequest",
			},
			requestBody: map[string]string{"key": "value"},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var receivedBody map[string]string
				err = json.Unmarshal(body, &receivedBody)
				assert.NoError(t, err)
				assert.Equal(t, "value", receivedBody["key"])

				response := map[string]string{"result": "success"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			},
			verifyResponse: func(t *testing.T, response *HTTPClientResponse, responseBody interface{}) {
				assert.Equal(t, 200, response.StatusCode)
				assert.Contains(t, string(response.Body), "success")

				if responseBody != nil {
					respMap, ok := responseBody.(*map[string]string)
					require.True(t, ok)
					assert.Equal(t, "success", (*respMap)["result"])
				}
			},
		},
		{
			name: "Success - without request body",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "GET",
				LogPrefix: "TestRequest",
			},
			requestBody: nil,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

				response := map[string]string{"result": "success"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			},
			verifyResponse: func(t *testing.T, response *HTTPClientResponse, responseBody interface{}) {
				assert.Equal(t, 200, response.StatusCode)
				assert.Contains(t, string(response.Body), "success")
			},
		},
		{
			name: "Success - with custom headers",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestRequest",
				Headers: map[string]string{
					"Authorization":   "Bearer token123",
					"X-Custom-Header": "custom-value",
				},
			},
			requestBody: map[string]string{"key": "value"},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
				assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

				response := map[string]string{"result": "success"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			},
			verifyResponse: func(t *testing.T, response *HTTPClientResponse, responseBody interface{}) {
				assert.Equal(t, 200, response.StatusCode)
			},
		},
		{
			name: "Error - HTTP 500",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestRequest",
			},
			requestBody: map[string]string{"key": "value"},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			expectedError: "third-party service returned status: 500 Internal Server Error",
		},
		{
			name: "Error - HTTP 404",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestRequest",
			},
			requestBody: map[string]string{"key": "value"},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Not Found"))
			},
			expectedError: "third-party service returned status: 404 Not Found",
		},
		{
			name: "Error - invalid JSON response",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestRequest",
			},
			requestBody: map[string]string{"key": "value"},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			expectedError: "invalid character 'i' looking for beginning of value",
		},
		{
			name: "Error - marshal request body failed",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestRequest",
			},
			requestBody:   make(chan int), // This will cause marshaling to fail
			expectedError: "json: unsupported type: chan int",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.serverResponse))
			defer server.Close()

			// Update the config with the test server URL
			tc.config.BaseURL = server.URL

			client := NewHTTPClient(lgr)

			var responseBody map[string]string
			response, err := client.MakeJSONRequest(ctx, tc.config, tc.requestBody, &responseBody)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, response)

			if tc.verifyResponse != nil {
				tc.verifyResponse(t, response, &responseBody)
			}
		})
	}
}

func TestHTTPClient_MakeMultipartRequest(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	testCases := []struct {
		name           string
		config         HTTPClientConfig
		body           io.Reader
		contentType    string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
		verifyResponse func(t *testing.T, response *HTTPClientResponse)
	}{
		{
			name: "Success - multipart request",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestMultipartRequest",
			},
			body:        strings.NewReader("test content"),
			contentType: "multipart/form-data",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "multipart/form-data", r.Header.Get("Content-Type"))

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(body))

				response := map[string]string{"result": "success"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			},
			verifyResponse: func(t *testing.T, response *HTTPClientResponse) {
				assert.Equal(t, 200, response.StatusCode)
				assert.Contains(t, string(response.Body), "success")
			},
		},
		{
			name: "Error - HTTP 500",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestMultipartRequest",
			},
			body:        strings.NewReader("test content"),
			contentType: "multipart/form-data",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			expectedError: "third-party service returned status: 500 Internal Server Error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.serverResponse))
			defer server.Close()

			// Update the config with the test server URL
			tc.config.BaseURL = server.URL

			client := NewHTTPClient(lgr)

			var responseBody map[string]string
			response, err := client.MakeMultipartRequest(ctx, tc.config, tc.body, tc.contentType, &responseBody)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, response)

			if tc.verifyResponse != nil {
				tc.verifyResponse(t, response)
			}
		})
	}
}

func TestHTTPClient_MakeJSONRequestWithCustomStatusCheck(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	testCases := []struct {
		name                string
		config              HTTPClientConfig
		requestBody         interface{}
		expectedStatusCodes []int
		serverResponse      func(w http.ResponseWriter, r *http.Request)
		expectedError       string
		verifyResponse      func(t *testing.T, response *HTTPClientResponse)
	}{
		{
			name: "Success - 201 status code",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestCustomStatusRequest",
			},
			requestBody:         map[string]string{"key": "value"},
			expectedStatusCodes: []int{200, 201},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				response := map[string]string{"result": "created"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(response)
			},
			verifyResponse: func(t *testing.T, response *HTTPClientResponse) {
				assert.Equal(t, 201, response.StatusCode)
				assert.Contains(t, string(response.Body), "created")
			},
		},
		{
			name: "Success - 202 status code",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestCustomStatusRequest",
			},
			requestBody:         map[string]string{"key": "value"},
			expectedStatusCodes: []int{200, 202},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				response := map[string]string{"result": "accepted"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(response)
			},
			verifyResponse: func(t *testing.T, response *HTTPClientResponse) {
				assert.Equal(t, 202, response.StatusCode)
				assert.Contains(t, string(response.Body), "accepted")
			},
		},
		{
			name: "Error - unexpected status code",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestCustomStatusRequest",
			},
			requestBody:         map[string]string{"key": "value"},
			expectedStatusCodes: []int{200, 201},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad Request"))
			},
			expectedError: "third-party service returned unexpected status: 400 Bad Request",
		},
		{
			name: "Error - marshal request body failed",
			config: HTTPClientConfig{
				BaseURL:   "http://test-server",
				Method:    "POST",
				LogPrefix: "TestCustomStatusRequest",
			},
			requestBody:         make(chan int),
			expectedStatusCodes: []int{200, 201},
			expectedError:       "json: unsupported type: chan int",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.serverResponse))
			defer server.Close()

			tc.config.BaseURL = server.URL

			client := NewHTTPClient(lgr)

			var responseBody map[string]string
			response, err := client.MakeJSONRequestWithCustomStatusCheck(ctx, tc.config, tc.requestBody, &responseBody, tc.expectedStatusCodes)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, response)

			if tc.verifyResponse != nil {
				tc.verifyResponse(t, response)
			}
		})
	}
}

func TestHTTPClient_Timeout(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("delayed response"))
	}))
	defer server.Close()

	config := HTTPClientConfig{
		BaseURL:   server.URL,
		Method:    "GET",
		LogPrefix: "TestTimeout",
		Timeout:   100 * time.Millisecond, // Very short timeout
	}

	client := NewHTTPClient(lgr)

	_, err := client.MakeJSONRequest(ctx, config, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestHTTPClient_DefaultTimeout(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	config := HTTPClientConfig{
		BaseURL:   server.URL,
		Method:    "GET",
		LogPrefix: "TestDefaultTimeout",
		// No timeout specified, should use default
	}

	client := NewHTTPClient(lgr)

	response, err := client.MakeJSONRequest(ctx, config, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 200, response.StatusCode)
}

func TestHTTPClient_ContextCancellation(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)

	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("delayed response"))
	}))
	defer server.Close()

	config := HTTPClientConfig{
		BaseURL:   server.URL,
		Method:    "GET",
		LogPrefix: "TestContextCancellation",
		Timeout:   5 * time.Second,
	}

	client := NewHTTPClient(lgr)

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := client.MakeJSONRequest(ctx, config, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestHTTPClient_EmptyResponseBody(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		// No body
	}))
	defer server.Close()

	config := HTTPClientConfig{
		BaseURL:   server.URL,
		Method:    "GET",
		LogPrefix: "TestEmptyResponseBody",
	}

	client := NewHTTPClient(lgr)

	var responseBody map[string]string
	response, err := client.MakeJSONRequest(ctx, config, nil, &responseBody)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 204, response.StatusCode)
	assert.Empty(t, response.Body)
}

func TestHTTPClient_InvalidURL(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	config := HTTPClientConfig{
		BaseURL:   "invalid-url",
		Method:    "GET",
		LogPrefix: "TestInvalidURL",
	}

	client := NewHTTPClient(lgr)

	_, err := client.MakeJSONRequest(ctx, config, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
}

func TestHTTPClient_ResponseHeaders(t *testing.T) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	config := HTTPClientConfig{
		BaseURL:   server.URL,
		Method:    "GET",
		LogPrefix: "TestResponseHeaders",
	}

	client := NewHTTPClient(lgr)

	response, err := client.MakeJSONRequest(ctx, config, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "custom-value", response.Headers.Get("X-Custom-Header"))
	assert.Equal(t, "application/json", response.Headers.Get("Content-Type"))
}

// Benchmark tests
func BenchmarkHTTPClient_MakeJSONRequest(b *testing.B) {
	lgr := log.Initialize(constants.TestAppEnv)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"result": "success"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := HTTPClientConfig{
		BaseURL:   server.URL,
		Method:    "POST",
		LogPrefix: "BenchmarkTest",
	}

	client := NewHTTPClient(lgr)
	requestBody := map[string]string{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var responseBody map[string]string
		_, err := client.MakeJSONRequest(ctx, config, requestBody, &responseBody)
		if err != nil {
			b.Fatal(err)
		}
	}
}
