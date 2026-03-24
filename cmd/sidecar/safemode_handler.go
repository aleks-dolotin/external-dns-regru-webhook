package main

import (
	"encoding/json"
	"net/http"
)

// Story 8.3: Safe-mode HTTP endpoints

// handleSafeModeToggle handles POST /adapter/v1/safe-mode?enabled=true|false (AC #1, #3).
func (a *app) handleSafeModeToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeSafeModeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	enabled := r.URL.Query().Get("enabled")
	switch enabled {
	case "true":
		a.safeMode.Enable()
		a.logger.Info("safe-mode ENABLED by operator")
	case "false":
		a.safeMode.Disable()
		a.logger.Info("safe-mode DISABLED by operator")
	default:
		writeSafeModeError(w, http.StatusBadRequest, "enabled query parameter must be 'true' or 'false'")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(a.safeMode.Status())
}

// handleSafeModeStatus handles GET /adapter/v1/safe-mode (AC #4).
func (a *app) handleSafeModeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeSafeModeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(a.safeMode.Status())
}

func writeSafeModeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(resyncErrorResponse{Error: msg})
}
