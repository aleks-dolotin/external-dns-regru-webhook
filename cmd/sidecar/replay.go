package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/audit"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
	"go.uber.org/zap"
)

// Story 8.2: Replay failed operations

// failedOpJSON is the JSON representation of a failed operation for the list endpoint.
type failedOpJSON struct {
	OpID     string `json:"op_id"`
	Zone     string `json:"zone"`
	Action   string `json:"action"`
	Error    string `json:"error"`
	Attempts int    `json:"attempts"`
	Time     string `json:"time"`
}

// replayResponse is the JSON response for a successful replay.
type replayResponse struct {
	Status string `json:"status"`
	OpID   string `json:"op_id"`
}

// replayAllResponse is the JSON response for replay-all.
type replayAllResponse struct {
	Replayed int `json:"replayed"`
}

// handleListFailed handles GET /adapter/v1/failed (AC #1).
// Returns a JSON list of failed operations from the dead-letter list.
func (a *app) handleListFailed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeReplayError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	failed := a.pool.FailedOps()
	result := make([]failedOpJSON, len(failed))
	for i, fo := range failed {
		errMsg := ""
		if fo.Err != nil {
			errMsg = fo.Err.Error()
		}
		result[i] = failedOpJSON{
			OpID:     fo.Op.OpID,
			Zone:     fo.Op.Zone,
			Action:   fo.Op.Action,
			Error:    errMsg,
			Attempts: fo.Attempts,
			Time:     fo.Time.Format(time.RFC3339),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// handleReplay handles POST /adapter/v1/replay/{id} (AC #2, #3).
// Replays a single failed operation by ID.
func (a *app) handleReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeReplayError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Go 1.22+ path parameter support.
	opID := r.PathValue("id")
	if opID == "" {
		writeReplayError(w, http.StatusBadRequest, "operation ID is required")
		return
	}

	op, err := a.pool.ReplayOp(opID)
	if err != nil {
		// AC #3: 404 for unknown operation.
		writeReplayError(w, http.StatusNotFound, "operation "+opID+" not found in failed list")
		return
	}

	// Re-enqueue with fresh state.
	a.queue.Enqueue(queue.Operation{ID: op.OpID, Body: op})

	a.logger.Info("operation replayed",
		zap.String("correlating_id", op.OpID),
		zap.String("zone", op.Zone),
		zap.String("action", op.Action),
	)

	// AC #4: audit trail for replay.
	if a.auditor != nil {
		a.auditor.Record(audit.AuditEvent{
			Timestamp:     time.Now().UTC(),
			Operation:     "replay",
			Actor:         "operator",
			Zone:          op.Zone,
			FQDN:          op.Name,
			RecordType:    op.Type,
			CorrelatingID: op.OpID,
			Result:        "replayed",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(replayResponse{Status: "replayed", OpID: op.OpID})
}

// handleReplayAll handles POST /adapter/v1/replay-all (AC #5).
// Replays all failed operations.
func (a *app) handleReplayAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeReplayError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ops := a.pool.ReplayAll()
	for _, op := range ops {
		a.queue.Enqueue(queue.Operation{ID: op.OpID, Body: op})
	}

	a.logger.Info("all failed operations replayed", zap.Int("count", len(ops)))

	// Audit each replayed operation.
	if a.auditor != nil {
		for _, op := range ops {
			a.auditor.Record(audit.AuditEvent{
				Timestamp:     time.Now().UTC(),
				Operation:     "replay-all",
				Actor:         "operator",
				Zone:          op.Zone,
				FQDN:          op.Name,
				RecordType:    op.Type,
				CorrelatingID: op.OpID,
				Result:        "replayed",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(replayAllResponse{Replayed: len(ops)})
}

func writeReplayError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(resyncErrorResponse{Error: msg})
}
