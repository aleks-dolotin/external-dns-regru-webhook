package health

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status string.
type Status string

const (
	StatusOK   Status = "ok"
	StatusFail Status = "fail"
)

// CheckResult holds the result of a single health sub-check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthResponse is the JSON body returned by /healthz.
type HealthResponse struct {
	Status    Status    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ReadyResponse is the JSON body returned by /ready.
type ReadyResponse struct {
	Status    Status        `json:"status"`
	Timestamp time.Time     `json:"timestamp"`
	Checks    []CheckResult `json:"checks"`
}

// ReadyCheck is a named function that returns whether a subsystem is ready.
type ReadyCheck struct {
	Name  string
	Check func() (bool, string) // (ok, message)
}

// Checker aggregates readiness checks and serves /healthz and /ready.
type Checker struct {
	mu     sync.RWMutex
	checks []ReadyCheck
}

// NewChecker creates a Checker with the given readiness checks.
func NewChecker(checks ...ReadyCheck) *Checker {
	return &Checker{checks: checks}
}

// AddCheck appends a readiness check (thread-safe).
func (c *Checker) AddCheck(check ReadyCheck) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks = append(c.checks, check)
}

// HealthzHandler returns HTTP 200 with JSON body (liveness — process alive).
func (c *Checker) HealthzHandler(w http.ResponseWriter, _ *http.Request) {
	resp := HealthResponse{
		Status:    StatusOK,
		Timestamp: time.Now().UTC(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// ReadyHandler returns HTTP 200 if all checks pass, 503 otherwise.
func (c *Checker) ReadyHandler(w http.ResponseWriter, _ *http.Request) {
	c.mu.RLock()
	checks := make([]ReadyCheck, len(c.checks))
	copy(checks, c.checks)
	c.mu.RUnlock()

	results := make([]CheckResult, 0, len(checks))
	allOK := true

	for _, ch := range checks {
		ok, msg := ch.Check()
		st := StatusOK
		if !ok {
			st = StatusFail
			allOK = false
		}
		results = append(results, CheckResult{
			Name:    ch.Name,
			Status:  st,
			Message: msg,
		})
	}

	overall := StatusOK
	if !allOK {
		overall = StatusFail
	}

	resp := ReadyResponse{
		Status:    overall,
		Timestamp: time.Now().UTC(),
		Checks:    results,
	}

	code := http.StatusOK
	if !allOK {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, resp)
}

// writeJSON marshals v to JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
