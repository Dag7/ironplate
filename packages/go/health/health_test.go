package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivenessHandler(t *testing.T) {
	h := NewHandler("test-service")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.LivenessHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if resp.Service != "test-service" {
		t.Errorf("service = %q, want %q", resp.Service, "test-service")
	}
}

func TestReadinessHandler_Healthy(t *testing.T) {
	h := NewHandler("test-service")
	h.Register("db", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy, Message: "connected"}
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.ReadinessHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if resp.Checks["db"].Status != StatusHealthy {
		t.Errorf("check status = %q, want %q", resp.Checks["db"].Status, StatusHealthy)
	}
}

func TestReadinessHandler_Unhealthy(t *testing.T) {
	h := NewHandler("test-service")
	h.Register("db", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusUnhealthy, Message: "connection refused"}
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.ReadinessHandler()(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
