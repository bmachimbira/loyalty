package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewTestHTTPServer creates a test HTTP server with Gin router
func NewTestHTTPServer(t *testing.T, setupRouter func(*gin.Engine)) *httptest.Server {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	setupRouter(router)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server
}

// NewTestRouter creates a test Gin router
func NewTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	return gin.New()
}

// NewTestContext creates a test Gin context
func NewTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	return ctx, w
}

// MakeRequest makes an HTTP request to a test server
func MakeRequest(t *testing.T, method, url string, body interface{}, headers map[string]string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
		bodyReader = bytes.NewReader(bodyJSON)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err, "Failed to create request")

	// Set default headers
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to execute request")

	return resp
}

// MakeAuthenticatedRequest makes an authenticated HTTP request with JWT token
func MakeAuthenticatedRequest(t *testing.T, method, url, token string, body interface{}) *http.Response {
	t.Helper()

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	return MakeRequest(t, method, url, body, headers)
}

// MakeHMACRequest makes an HMAC-authenticated request
func MakeHMACRequest(t *testing.T, method, url, signature, timestamp string, body interface{}) *http.Response {
	t.Helper()

	headers := map[string]string{
		"X-Signature":  signature,
		"X-Timestamp":  timestamp,
		"X-Identifier": "test-key",
	}

	return MakeRequest(t, method, url, body, headers)
}

// ParseJSONResponse parses JSON response into a struct
func ParseJSONResponse(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	err = json.Unmarshal(body, v)
	require.NoError(t, err, "Failed to unmarshal response: %s", string(body))
}

// AssertJSONResponse asserts the response has correct status and parses JSON
func AssertJSONResponse(t *testing.T, resp *http.Response, expectedStatus int, v interface{}) {
	t.Helper()

	assert.Equal(t, expectedStatus, resp.StatusCode, "Unexpected status code")
	ParseJSONResponse(t, resp, v)
}

// AssertErrorResponse asserts the response is an error with expected message
func AssertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedErrorContains string) {
	t.Helper()

	assert.Equal(t, expectedStatus, resp.StatusCode, "Unexpected status code")

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	var errorResp map[string]interface{}
	err = json.Unmarshal(body, &errorResp)
	require.NoError(t, err, "Failed to unmarshal error response")

	errorMsg, ok := errorResp["error"].(string)
	require.True(t, ok, "Error response missing 'error' field")

	if expectedErrorContains != "" {
		assert.Contains(t, errorMsg, expectedErrorContains, "Error message doesn't contain expected text")
	}
}

// AssertStatusCode asserts the response status code
func AssertStatusCode(t *testing.T, resp *http.Response, expectedStatus int) {
	t.Helper()

	if resp.StatusCode != expectedStatus {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Response body: %s", string(body))
	}

	assert.Equal(t, expectedStatus, resp.StatusCode, "Unexpected status code")
}

// ReadResponseBody reads and returns the response body
func ReadResponseBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	return string(body)
}

// MakeGinRequest creates a Gin test request
func MakeGinRequest(t *testing.T, router *gin.Engine, method, url string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
		bodyReader = bytes.NewReader(bodyJSON)
	}

	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err, "Failed to create request")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	router.ServeHTTP(w, req)
	return w
}

// MakeGinAuthenticatedRequest creates a Gin test request with auth header
func MakeGinAuthenticatedRequest(t *testing.T, router *gin.Engine, method, url, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
		bodyReader = bytes.NewReader(bodyJSON)
	}

	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err, "Failed to create request")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)
	return w
}

// ParseGinResponse parses a Gin response into a struct
func ParseGinResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()

	err := json.Unmarshal(w.Body.Bytes(), v)
	require.NoError(t, err, "Failed to unmarshal response: %s", w.Body.String())
}

// AssertGinJSONResponse asserts Gin response status and parses JSON
func AssertGinJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, v interface{}) {
	t.Helper()

	assert.Equal(t, expectedStatus, w.Code, "Unexpected status code")
	ParseGinResponse(t, w, v)
}

// AssertGinErrorResponse asserts Gin error response
func AssertGinErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	t.Helper()

	assert.Equal(t, expectedStatus, w.Code, "Unexpected status code")

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err, "Failed to unmarshal error response")

	errorMsg, ok := errorResp["error"].(string)
	require.True(t, ok, "Error response missing 'error' field")

	if expectedErrorContains != "" {
		assert.Contains(t, errorMsg, expectedErrorContains, "Error message doesn't contain expected text")
	}
}

// CreateTestJWT creates a test JWT token for testing
func CreateTestJWT(t *testing.T, userID, tenantID, role string) string {
	t.Helper()

	// This is a simplified JWT for testing
	// In real implementation, use the actual JWT generation from auth package
	return "test-jwt-token"
}
