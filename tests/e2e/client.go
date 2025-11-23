package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:8080"
	timeout = 30 * time.Second
)

// TestClient HTTP клиент для E2E тестов
type TestClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewTestClient создает новый тестовый клиент
func NewTestClient() *TestClient {
	return &TestClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Get выполняет GET запрос
func (c *TestClient) Get(t *testing.T, path string) *http.Response {
	resp, err := c.HTTPClient.Get(c.BaseURL + path)
	require.NoError(t, err, "GET request failed")
	return resp
}

// Post выполняет POST запрос
func (c *TestClient) Post(t *testing.T, path string, body interface{}) *http.Response {
	jsonData, err := json.Marshal(body)
	require.NoError(t, err, "Failed to marshal request body")

	resp, err := c.HTTPClient.Post(
		c.BaseURL+path,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err, "POST request failed")
	return resp
}

// ReadBody читает тело ответа
func (c *TestClient) ReadBody(t *testing.T, resp *http.Response) []byte {
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	return body
}

// DecodeJSON декодирует JSON ответ
func (c *TestClient) DecodeJSON(t *testing.T, resp *http.Response, v interface{}) {
	body := c.ReadBody(t, resp)
	err := json.Unmarshal(body, v)
	require.NoError(t, err, "Failed to decode JSON: %s", string(body))
}

// WaitForService ожидает доступности сервиса
func (c *TestClient) WaitForService(t *testing.T, maxAttempts int) {
	for i := 0; i < maxAttempts; i++ {
		resp, err := c.HTTPClient.Get(c.BaseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			t.Log("Service is ready")
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("Service did not become ready in time")
}

// AssertStatusCode проверяет код ответа
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	assert.Equal(t, expected, resp.StatusCode,
		fmt.Sprintf("Expected status %d, got %d", expected, resp.StatusCode))
}
