package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClientConfig() *ClientConfig {
	return &ClientConfig{
		ServiceName:    "apigateway",
		ServiceVersion: "test",
	}
}

func TestCreateLivenessHandlerReturnsHealthyContract(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	createLivenessHandler(testClientConfig())(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	if body := rec.Body.String(); !containsJSONField(body, `"status":"healthy"`) {
		t.Fatalf("expected healthy status in response: %s", body)
	}
}

func TestCreateReadinessHandlerFailsWhenDependenciesAreUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	createReadinessHandler(testClientConfig(), nil, nil)(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected HTTP 503, got %d", rec.Code)
	}
	if body := rec.Body.String(); !containsJSONField(body, `"status":"not_ready"`) {
		t.Fatalf("expected not_ready status in response: %s", body)
	}
}

func containsJSONField(body, field string) bool {
	for i := 0; i+len(field) <= len(body); i++ {
		if body[i:i+len(field)] == field {
			return true
		}
	}
	return false
}
