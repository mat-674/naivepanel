package handlers

import (
	"errors"
	"log"
	"naivepanel/internal/database"
	"naivepanel/internal/models"
	"naivepanel/internal/naiveproxy"
	"net/http"
	"strings"
)

// StatusHandler handles server status and service control
type StatusHandler struct {
	DB      *database.DB
	Manager *naiveproxy.Manager
	// Version is the panel build identifier, injected at link time via
	// -X main.version and threaded in from main. Empty means "unknown".
	Version string
}

// GetStatus handles GET /api/status
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// A database read failure fails the whole response rather than degrading it:
	// reporting 0 users / 0 traffic for a broken database is indistinguishable
	// from a genuinely empty panel, and the dashboard would present the wrong
	// numbers as fact.
	userCount, err := h.DB.UserCount()
	if err != nil {
		log.Printf("Failed to count users for status: %v", err)
		jsonError(w, "failed to read status", http.StatusInternalServerError)
		return
	}

	users, err := h.DB.ListUsers()
	if err != nil {
		log.Printf("Failed to list users for status: %v", err)
		jsonError(w, "failed to read status", http.StatusInternalServerError)
		return
	}

	sysOS, sysArch := naiveproxy.GetSystemInfo()

	// Calculate total traffic
	var totalUp, totalDown int64
	for _, u := range users {
		totalUp += u.TrafficUp
		totalDown += u.TrafficDown
	}

	version := h.Version
	if version == "" {
		version = "unknown"
	}

	status := models.ServerStatus{
		Running:      h.Manager.IsRunning(),
		PID:          h.Manager.PID(),
		Uptime:       h.Manager.FormatUptime(),
		UserCount:    userCount,
		TotalUp:      totalUp,
		TotalDown:    totalDown,
		Version:      version,
		SystemOS:     sysOS,
		SystemArch:   sysArch,
		SystemUptime: naiveproxy.GetSystemUptime(),
	}

	jsonSuccess(w, "", status)
}

// ServiceAction handles POST /api/service/{action}
func (h *StatusHandler) ServiceAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	action := strings.TrimPrefix(r.URL.Path, "/api/service/")

	var err error
	switch action {
	case "start":
		err = h.Manager.Start()
	case "stop":
		err = h.Manager.Stop()
	case "restart":
		err = h.Manager.Restart()
	case "update":
		err = h.Manager.UpdatePanel()
	default:
		jsonError(w, "invalid action, use: start, stop, restart, update", http.StatusBadRequest)
		return
	}

	if err != nil {
		// Always keep the full error server-side; the client sees either a
		// sentinel (operator-actionable deployment guidance, safe verbatim) or a
		// generic message, because every other failure wraps raw systemctl /
		// systemd-run output.
		log.Printf("Service action %q failed: %v", action, err)

		message := serviceActionFailureMessage(action)
		if errors.Is(err, naiveproxy.ErrUpdateUnsupported) || errors.Is(err, naiveproxy.ErrUpdateScriptMissing) {
			message = err.Error()
		}
		jsonError(w, message, http.StatusInternalServerError)
		return
	}

	jsonSuccess(w, action+" successful", nil)
}

// serviceActionFailureMessage returns the generic, internals-free message shown
// to the client when a lifecycle call fails. The detail lives in the panel log.
func serviceActionFailureMessage(action string) string {
	switch action {
	case "update":
		return "failed to launch the panel update, check the panel log"
	default:
		return "failed to " + action + " naiveproxy, check the panel log"
	}
}
