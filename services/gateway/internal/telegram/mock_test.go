package telegram

import (
	"net/http"
	"net/http/httptest"
)

// MockHTTPClient is a mock HTTP client for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do mocks the Do method of http.Client
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// Get mocks the Get method of http.Client
func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return m.Do(req)
}

// MockResponse creates a mock HTTP response
func MockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       NewMockReadCloser([]byte(body)),
		Header:     make(http.Header),
	}
}

// MockServer creates a test server that returns a predefined response
func MockServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}
