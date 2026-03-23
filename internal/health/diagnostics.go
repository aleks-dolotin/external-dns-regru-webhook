package health

import (
	"net/http"
	"time"
)

// ZoneStatus holds per-zone diagnostics information.
type ZoneStatus struct {
	LastError     string     `json:"last_error,omitempty"`
	LastErrorTime *time.Time `json:"last_error_time,omitempty"`
}

// DiagnosticsResponse is the JSON body returned by /adapter/v1/diagnostics.
type DiagnosticsResponse struct {
	QueueDepth    int                   `json:"queue_depth"`
	WorkerCount   int                   `json:"worker_count"`
	LastHeartbeat *time.Time            `json:"last_heartbeat,omitempty"`
	Zones         map[string]ZoneStatus `json:"zones,omitempty"`
	Timestamp     time.Time             `json:"timestamp"`
}

// DiagnosticsSource provides data for the diagnostics endpoint.
// Implemented by the application layer to decouple health from queue/worker packages.
type DiagnosticsSource interface {
	QueueDepth() int
	WorkerCount() int
	LastHeartbeat() time.Time
	ZoneErrors() map[string]ZoneErrorInfo
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
			QueueDepth:  src.QueueDepth(),
			WorkerCount: src.WorkerCount(),
			Timestamp:   time.Now().UTC(),
		}

		hb := src.LastHeartbeat()
		if !hb.IsZero() {
			resp.LastHeartbeat = &hb
		}

		errs := src.ZoneErrors()
		if len(errs) > 0 {
			resp.Zones = make(map[string]ZoneStatus, len(errs))
			for zone, ze := range errs {
				t := ze.Time
				resp.Zones[zone] = ZoneStatus{
					LastError:     ze.Message,
					LastErrorTime: &t,
				}
			}
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
