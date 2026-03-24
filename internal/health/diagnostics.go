package health

import (
	"net/http"
	"time"
)

// ZoneStatus holds per-zone diagnostics information.
type ZoneStatus struct {
	LastError     string     `json:"last_error,omitempty"`
	LastErrorTime *time.Time `json:"last_error_time,omitempty"`
	CircuitState  string     `json:"circuit_state,omitempty"` // Story 5.4
	RateLimited   bool       `json:"rate_limited,omitempty"`  // Story 5.5
}

// DiagnosticsResponse is the JSON body returned by /adapter/v1/diagnostics.
type DiagnosticsResponse struct {
	QueueDepth     int                   `json:"queue_depth"`
	WorkerCount    int                   `json:"worker_count"`
	LastHeartbeat  *time.Time            `json:"last_heartbeat,omitempty"`
	Backpressure   bool                  `json:"backpressure"`              // Story 5.5
	ThrottledZones []string              `json:"throttled_zones,omitempty"` // Story 5.5
	Zones          map[string]ZoneStatus `json:"zones,omitempty"`
	Resync         *ResyncStatus         `json:"resync,omitempty"` // Story 8.1
	Timestamp      time.Time             `json:"timestamp"`
}

// ResyncStatus reports the state of the last force-resync operation (Story 8.1).
type ResyncStatus struct {
	Running     bool       `json:"running"`
	LastTime    *time.Time `json:"last_time,omitempty"`
	LastActions int        `json:"last_actions"`
	LastError   string     `json:"last_error,omitempty"`
}

// DiagnosticsSource provides data for the diagnostics endpoint.
// Implemented by the application layer to decouple health from queue/worker packages.
type DiagnosticsSource interface {
	QueueDepth() int
	WorkerCount() int
	LastHeartbeat() time.Time
	ZoneErrors() map[string]ZoneErrorInfo
	// Story 5.5: backpressure signals.
	IsBackpressured() bool
	ThrottledZones() []string
	// Story 5.4: circuit breaker states.
	CircuitStates() map[string]string
	// Story 8.1: force-resync status.
	ResyncRunning() bool
	LastResyncTime() time.Time
	LastResyncActions() int
	LastResyncError() string
}

// ZoneErrorInfo is a simplified zone error for the diagnostics source interface.
type ZoneErrorInfo struct {
	Message string
	Time    time.Time
}

// DiagnosticsHandler returns an http.HandlerFunc that serves diagnostics JSON.
func DiagnosticsHandler(src DiagnosticsSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := DiagnosticsResponse{
			QueueDepth:     src.QueueDepth(),
			WorkerCount:    src.WorkerCount(),
			Backpressure:   src.IsBackpressured(),
			ThrottledZones: src.ThrottledZones(),
			Timestamp:      time.Now().UTC(),
		}

		hb := src.LastHeartbeat()
		if !hb.IsZero() {
			resp.LastHeartbeat = &hb
		}

		errs := src.ZoneErrors()
		circuitStates := src.CircuitStates()

		// Merge zone errors and circuit states.
		if len(errs) > 0 || len(circuitStates) > 0 {
			resp.Zones = make(map[string]ZoneStatus)
			for zone, ze := range errs {
				t := ze.Time
				zs := resp.Zones[zone]
				zs.LastError = ze.Message
				zs.LastErrorTime = &t
				resp.Zones[zone] = zs
			}
			for zone, state := range circuitStates {
				zs := resp.Zones[zone]
				zs.CircuitState = state
				resp.Zones[zone] = zs
			}
		}

		// Story 8.1: include resync status if any resync has occurred.
		lrt := src.LastResyncTime()
		if src.ResyncRunning() || !lrt.IsZero() {
			rs := &ResyncStatus{
				Running:     src.ResyncRunning(),
				LastActions: src.LastResyncActions(),
				LastError:   src.LastResyncError(),
			}
			if !lrt.IsZero() {
				rs.LastTime = &lrt
			}
			resp.Resync = rs
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
