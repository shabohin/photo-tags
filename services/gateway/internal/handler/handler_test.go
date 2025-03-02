package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
)

func TestHealthCheck(t *testing.T) {
	// Setup
	logger := logging.NewLogger("test")
	cfg := &config.Config{ServerPort: 8080}
	h := NewHandler(logger, cfg)

	// Create request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.HealthCheck)

	// Execute
	handler.ServeHTTP(rr, req)

	// Verify status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Check status field
	if status, ok := response["status"]; !ok || status != "ok" {
		t.Errorf("Expected status to be 'ok', got '%v'", status)
	}

	// Check service field
	if service, ok := response["service"]; !ok || service != "gateway" {
		t.Errorf("Expected service to be 'gateway', got '%v'", service)
	}

	// Timestamp should exist
	if _, ok := response["timestamp"]; !ok {
		t.Errorf("Expected timestamp to exist in response")
	}
}

func TestSetupRoutes(t *testing.T) {
	// Setup
	logger := logging.NewLogger("test")
	cfg := &config.Config{ServerPort: 8080}
	h := NewHandler(logger, cfg)

	// Create router
	router := h.SetupRoutes()

	// Test health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify not found for non-existent endpoint
	req, err = http.NewRequest("GET", "/non-existent", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify status code for not found
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code for non-existent path: got %v want %v", status, http.StatusNotFound)
	}
}
