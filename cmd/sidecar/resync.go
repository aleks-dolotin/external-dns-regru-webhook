package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/audit"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/reconciler"
	"go.uber.org/zap"
)

// resyncState tracks the current force-resync operation and its last result.
// Story 8.1: used by the /adapter/v1/resync endpoint and diagnostics.
type resyncState struct {
	mu              sync.Mutex
	running         bool
	lastResyncTime  time.Time
	lastResyncOps   int
	lastResyncError string
}

// resyncResponse is the JSON response for a successful resync.
type resyncResponse struct {
	Status          string `json:"status"`
	ActionsEnqueued int    `json:"actions_enqueued"`
	Scope           string `json:"scope"`
}

// resyncErrorResponse is the JSON response for resync errors.
type resyncErrorResponse struct {
	Error string `json:"error"`
}

// handleResync handles POST /adapter/v1/resync with query params zone and/or namespace.
// It runs reconciliation for the specified scope and returns the number of corrective actions.
// Concurrent resync requests are rejected with 409 Conflict (AC #4).
func (a *app) handleResync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeResyncError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	zone := r.URL.Query().Get("zone")
	namespace := r.URL.Query().Get("namespace")

	// AC: at least one param required.
	if zone == "" && namespace == "" {
		writeResyncError(w, http.StatusBadRequest, "zone or namespace query parameter required")
		return
	}

	// AC #4: mutex — only one resync at a time.
	if !a.resync.mu.TryLock() {
		writeResyncError(w, http.StatusConflict, "resync already in progress")
		return
	}
	defer a.resync.mu.Unlock()

	a.resync.running = true
	defer func() { a.resync.running = false }()

	// Resolve scope: determine which zones to reconcile.
	var zones []string
	var scope string

	if zone != "" {
		zones = []string{zone}
		scope = "zone=" + zone
	} else {
		// AC #2: namespace → look up mapped zones.
		if a.configStore == nil {
			writeResyncError(w, http.StatusServiceUnavailable, "config store not available, cannot resolve namespace")
			return
		}
		zones = a.configStore.ZonesForNamespace(namespace)
		if len(zones) == 0 {
			writeResyncError(w, http.StatusNotFound, "no zones mapped to namespace "+namespace)
			return
		}
		scope = "namespace=" + namespace
	}

	// Check reconciler availability.
	if a.reconciler == nil {
		writeResyncError(w, http.StatusServiceUnavailable, "reconciler not available")
		return
	}

	a.logger.Info("force-resync started",
		zap.String("scope", scope),
		zap.Strings("zones", zones),
	)

	// Run reconciliation for each zone. We need desired records per zone.
	// The reconciler's Reconcile method takes desired records and compares with actual.
	// For force-resync, we pass the desired records and enqueue corrections.
	var totalActions int
	var resyncErr error

	for _, z := range zones {
		desired := a.desiredRecordsForZone(z)
		actions, err := a.reconciler.Reconcile(r.Context(), desired, true)
		if err != nil {
			resyncErr = err
			a.logger.Error("resync reconciliation failed",
				zap.String("zone", z),
				zap.Error(err),
			)
			break
		}
		totalActions += len(actions)
	}

	// Update resync state for diagnostics (AC #3).
	a.resync.lastResyncTime = time.Now().UTC()
	a.resync.lastResyncOps = totalActions
	if resyncErr != nil {
		a.resync.lastResyncError = resyncErr.Error()
	} else {
		a.resync.lastResyncError = ""
	}

	// AC #5: audit event.
	if a.auditor != nil {
		result := "success"
		errDetail := ""
		if resyncErr != nil {
			result = "failure"
			errDetail = resyncErr.Error()
		}
		a.auditor.Record(audit.AuditEvent{
			Timestamp:   time.Now().UTC(),
			Operation:   "force-resync",
			Actor:       "operator",
			Zone:        scope,
			Result:      result,
			ErrorDetail: errDetail,
		})
	}

	if resyncErr != nil {
		writeResyncError(w, http.StatusInternalServerError, "resync failed: "+resyncErr.Error())
		return
	}

	a.logger.Info("force-resync completed",
		zap.String("scope", scope),
		zap.Int("actions_enqueued", totalActions),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resyncResponse{
		Status:          "completed",
		ActionsEnqueued: totalActions,
		Scope:           scope,
	})
}

// desiredRecordsForZone returns the desired DNS records for a zone from
// the in-memory cache. The cache is populated by incoming events on
// /adapter/v1/events — create/update add records, delete removes them.
func (a *app) desiredRecordsForZone(zone string) []reconciler.DesiredRecord {
	if a.desired == nil {
		return nil
	}
	return a.desired.ForZone(zone)
}

func writeResyncError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(resyncErrorResponse{Error: msg})
}
