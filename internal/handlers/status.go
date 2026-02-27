package handlers

import (
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
}

// GetStatus handles GET /api/status
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userCount, _ := h.DB.UserCount()
	sysOS, sysArch := naiveproxy.GetSystemInfo()

	// Calculate total traffic
	users, _ := h.DB.ListUsers()
	var totalUp, totalDown int64
	for _, u := range users {
		totalUp += u.TrafficUp
		totalDown += u.TrafficDown
	}

	status := models.ServerStatus{
		Running:      h.Manager.IsRunning(),
		PID:          h.Manager.PID(),
		Uptime:       h.Manager.FormatUptime(),
		UserCount:    userCount,
		TotalUp:      totalUp,
		TotalDown:    totalDown,
		Version:      "1.0.0",
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
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonSuccess(w, action+" successful", nil)
}
