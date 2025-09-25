package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	log "gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type HTTPClientConfig struct {
	BaseURL   string
	Timeout   time.Duration
	Method    string
	Headers   map[string]string
	LogPrefix string
}

type HTTPClientResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

type HTTPClient struct {
	log *log.Logger
}

func NewHTTPClient(logger *log.Logger) *HTTPClient {
	return &HTTPClient{
		log: logger,
	}
}

func (c *HTTPClient) MakeJSONRequest(ctx context.Context, config HTTPClientConfig, requestBody interface{}, responseBody interface{}) (*HTTPClientResponse, error) {
	c.log.InfoWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Making JSON request to %s", config.LogPrefix, config.BaseURL))

	var jsonBody []byte
	var err error
	if requestBody != nil {
		jsonBody, err = json.Marshal(requestBody)
		if err != nil {
			c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to marshal request body", config.LogPrefix), err)
			return nil, err
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, config.Method, config.BaseURL, bytes.NewReader(jsonBody))
	if err != nil {
		c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to create HTTP request", config.LogPrefix), err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	for key, value := range config.Headers {
		httpReq.Header.Set(key, value)
	}

	response, err := c.makeRequest(ctx, config, httpReq)
	if err != nil {
		return nil, err
	}

	if responseBody != nil && len(response.Body) > 0 {
		if err := json.Unmarshal(response.Body, responseBody); err != nil {
			c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to unmarshal response body", config.LogPrefix), err)
			return nil, err
		}
	}

	return response, nil
}

func (c *HTTPClient) MakeMultipartRequest(ctx context.Context, config HTTPClientConfig, body io.Reader, contentType string, responseBody interface{}) (*HTTPClientResponse, error) {
	c.log.InfoWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Making multipart request to %s", config.LogPrefix, config.BaseURL))

	httpReq, err := http.NewRequestWithContext(ctx, config.Method, config.BaseURL, body)
	if err != nil {
		c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to create HTTP request", config.LogPrefix), err)
		return nil, err
	}

	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	for key, value := range config.Headers {
		httpReq.Header.Set(key, value)
	}

	response, err := c.makeRequest(ctx, config, httpReq)
	if err != nil {
		return nil, err
	}

	if responseBody != nil && len(response.Body) > 0 {
		if err := json.Unmarshal(response.Body, responseBody); err != nil {
			c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to unmarshal response body", config.LogPrefix), err)
			return nil, err
		}
	}

	return response, nil
}

func (c *HTTPClient) makeRequest(ctx context.Context, config HTTPClientConfig, httpReq *http.Request) (*HTTPClientResponse, error) {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = constants.TimeoutHTTP
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] HTTP request failed", config.LogPrefix), err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to read response body", config.LogPrefix), err)
		return nil, err
	}

	// Log response
	c.log.InfoWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Response: %s | Body: %s", config.LogPrefix, resp.Status, string(bodyBytes)))

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("third-party service returned status: %s | body: %s", resp.Status, string(bodyBytes))
		c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] HTTP error response", config.LogPrefix), err)
		return nil, err
	}

	return &HTTPClientResponse{
		StatusCode: resp.StatusCode,
		Body:       bodyBytes,
		Headers:    resp.Header,
	}, nil
}

func (c *HTTPClient) MakeJSONRequestWithCustomStatusCheck(ctx context.Context, config HTTPClientConfig, requestBody interface{}, responseBody interface{}, expectedStatusCodes []int) (*HTTPClientResponse, error) {
	c.log.InfoWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Making JSON request to %s", config.LogPrefix, config.BaseURL))

	var jsonBody []byte
	var err error
	if requestBody != nil {
		jsonBody, err = json.Marshal(requestBody)
		if err != nil {
			c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to marshal request body", config.LogPrefix), err)
			return nil, err
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, config.Method, config.BaseURL, bytes.NewReader(jsonBody))
	if err != nil {
		c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to create HTTP request", config.LogPrefix), err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	for key, value := range config.Headers {
		httpReq.Header.Set(key, value)
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = constants.TimeoutHTTP
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] HTTP request failed", config.LogPrefix), err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to read response body", config.LogPrefix), err)
		return nil, err
	}

	c.log.InfoWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Response: %s | Body: %s", config.LogPrefix, resp.Status, string(bodyBytes)))

	isValidStatus := false
	for _, expectedStatus := range expectedStatusCodes {
		if resp.StatusCode == expectedStatus {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		err := fmt.Errorf("third-party service returned unexpected status: %s | body: %s", resp.Status, string(bodyBytes))
		c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] HTTP error response", config.LogPrefix), err)
		return nil, err
	}

	if responseBody != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, responseBody); err != nil {
			c.log.ErrorWithID(ctx, fmt.Sprintf("[HTTPClient: %s] Failed to unmarshal response body", config.LogPrefix), err)
			return nil, err
		}
	}

	return &HTTPClientResponse{
		StatusCode: resp.StatusCode,
		Body:       bodyBytes,
		Headers:    resp.Header,
	}, nil
}
