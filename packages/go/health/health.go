// Package health provides HTTP handlers for Kubernetes liveness and
// readiness probes.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the outcome of a health check.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckResult is returned by a health check function.
type CheckResult struct {
	Status    Status `json:"status"`
	Message   string `json:"message,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// CheckFunc is a function that performs a health check.
type CheckFunc func(ctx context.Context) CheckResult

// Response is the JSON body returned by health endpoints.
type Response struct {
	Status  string                 `json:"status"`
	Service string                 `json:"service"`
	Uptime  int64                  `json:"uptime"`
	Checks  map[string]CheckResult `json:"checks,omitempty"`
}

// Handler manages health check registrations and serves HTTP endpoints.
type Handler struct {
	serviceName string
	startTime   time.Time
	mu          sync.RWMutex
	checks      map[string]CheckFunc
}

// NewHandler creates a new health check handler.
func NewHandler(serviceName string) *Handler {
	return &Handler{
		serviceName: serviceName,
		startTime:   time.Now(),
		checks:      make(map[string]CheckFunc),
	}
}

// Register adds a named health check. Checks are executed on readiness probes.
func (h *Handler) Register(name string, check CheckFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = check
}

// LivenessHandler returns an http.HandlerFunc for the /healthz endpoint.
func (h *Handler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Status:  "ok",
			Service: h.serviceName,
			Uptime:  int64(time.Since(h.startTime).Seconds()),
		})
	}
}

// ReadinessHandler returns an http.HandlerFunc for the /readyz endpoint.
// It runs all registered checks and returns an aggregate status.
func (h *Handler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		checks := make(map[string]CheckFunc, len(h.checks))
		for k, v := range h.checks {
			checks[k] = v
		}
		h.mu.RUnlock()

		results := make(map[string]CheckResult, len(checks))
		overallStatus := "ok"

		for name, check := range checks {
			start := time.Now()
			result := check(r.Context())
			result.LatencyMs = time.Since(start).Milliseconds()
			results[name] = result

			switch result.Status {
			case StatusUnhealthy:
				overallStatus = "unhealthy"
			case StatusDegraded:
				if overallStatus == "ok" {
					overallStatus = "degraded"
				}
			}
		}

		resp := Response{
			Status:  overallStatus,
			Service: h.serviceName,
			Uptime:  int64(time.Since(h.startTime).Seconds()),
		}
		if len(results) > 0 {
			resp.Checks = results
		}

		w.Header().Set("Content-Type", "application/json")
		if overallStatus == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(resp)
	}
}

// RegisterRoutes adds liveness and readiness routes to an http.ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.LivenessHandler())
	mux.HandleFunc("GET /readyz", h.ReadinessHandler())
}
