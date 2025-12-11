package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// HealthResponse represents the overall health check response
type HealthResponse struct {
	Status     Status                      `json:"status"`
	Version    string                      `json:"version"`
	Timestamp  time.Time                   `json:"timestamp"`
	Components map[string]ComponentHealth  `json:"components"`
}

// Checker interface for health checks
type Checker interface {
	Check(ctx context.Context) ComponentHealth
	Name() string
}

// Server manages health check endpoints
type Server struct {
	port     int
	checkers []Checker
	mu       sync.RWMutex
	server   *http.Server
}

// NewServer creates a new health check server
func NewServer(port int) *Server {
	return &Server{
		port:     port,
		checkers: make([]Checker, 0),
	}
}

// RegisterChecker adds a new health checker
func (s *Server) RegisterChecker(checker Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers = append(s.checkers, checker)
}

// Start starts the health check HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/health/live", s.livenessHandler)
	mux.HandleFunc("/health/ready", s.readinessHandler)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Health check server error: %v\n", err)
		}
	}()

	fmt.Printf("Health check server started on port %d\n", s.port)
	return nil
}

// Shutdown gracefully shuts down the health check server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// healthHandler returns detailed health status
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	s.mu.RLock()
	checkers := s.checkers
	s.mu.RUnlock()

	components := make(map[string]ComponentHealth)
	overallStatus := StatusHealthy

	for _, checker := range checkers {
		health := checker.Check(ctx)
		components[checker.Name()] = health

		// Determine overall status
		if health.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if health.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	response := HealthResponse{
		Status:     overallStatus,
		Version:    "1.0.0",
		Timestamp:  time.Now(),
		Components: components,
	}

	statusCode := http.StatusOK
	if overallStatus == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if overallStatus == StatusDegraded {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// livenessHandler returns basic liveness status (for Kubernetes)
func (s *Server) livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

// readinessHandler checks if the service is ready to handle requests
func (s *Server) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	s.mu.RLock()
	checkers := s.checkers
	s.mu.RUnlock()

	ready := true
	for _, checker := range checkers {
		health := checker.Check(ctx)
		if health.Status == StatusUnhealthy {
			ready = false
			break
		}
	}

	statusCode := http.StatusOK
	status := "ready"
	if !ready {
		statusCode = http.StatusServiceUnavailable
		status = "not_ready"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"status": status,
	})
}

// TemporalChecker checks Temporal server connectivity
type TemporalChecker struct {
	client client.Client
}

// NewTemporalChecker creates a new Temporal health checker
func NewTemporalChecker(c client.Client) *TemporalChecker {
	return &TemporalChecker{client: c}
}

// Name returns the checker name
func (t *TemporalChecker) Name() string {
	return "temporal"
}

// Check performs the health check
func (t *TemporalChecker) Check(ctx context.Context) ComponentHealth {
	start := time.Now()

	// Try to check the system info
	_, err := t.client.CheckHealth(ctx, &client.CheckHealthRequest{})
	latency := time.Since(start)

	if err != nil {
		return ComponentHealth{
			Status:  StatusUnhealthy,
			Message: fmt.Sprintf("Temporal connection failed: %v", err),
			Latency: latency.String(),
		}
	}

	return ComponentHealth{
		Status:  StatusHealthy,
		Message: "Connected to Temporal server",
		Latency: latency.String(),
	}
}

// HTTPChecker checks HTTP endpoint availability
type HTTPChecker struct {
	name   string
	url    string
	client *http.Client
}

// NewHTTPChecker creates a new HTTP health checker
func NewHTTPChecker(name, url string) *HTTPChecker {
	return &HTTPChecker{
		name: name,
		url:  url,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Name returns the checker name
func (h *HTTPChecker) Name() string {
	return h.name
}

// Check performs the health check
func (h *HTTPChecker) Check(ctx context.Context) ComponentHealth {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", h.url, nil)
	if err != nil {
		return ComponentHealth{
			Status:  StatusUnhealthy,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	resp, err := h.client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return ComponentHealth{
			Status:  StatusUnhealthy,
			Message: fmt.Sprintf("Request failed: %v", err),
			Latency: latency.String(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return ComponentHealth{
			Status:  StatusHealthy,
			Message: fmt.Sprintf("HTTP %d", resp.StatusCode),
			Latency: latency.String(),
		}
	}

	return ComponentHealth{
		Status:  StatusDegraded,
		Message: fmt.Sprintf("HTTP %d", resp.StatusCode),
		Latency: latency.String(),
	}
}
