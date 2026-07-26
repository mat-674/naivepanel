package handlers

import (
	"encoding/json"
	"log"
	"naivepanel/internal/database"
	"naivepanel/internal/naiveproxy"
	"net/http"
)

// SettingsHandler handles server settings
type SettingsHandler struct {
	DB            *database.DB
	Manager       *naiveproxy.Manager
	PanelUpstream string
}

// Get handles GET /api/settings
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	settings, err := h.DB.GetSettings()
	if err != nil {
		jsonError(w, "failed to get settings", http.StatusInternalServerError)
		return
	}

	jsonSuccess(w, "", settings)
}

// Update handles PUT /api/settings
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Start from the stored settings so keys the UI never sends — the
	// panel-publication trio, managed by --panel-public-enable/-disable — are
	// preserved instead of being cleared by a save from the Settings page.
	settings, err := h.DB.GetSettings()
	if err != nil {
		log.Printf("Failed to read settings before update: %v", err)
		jsonError(w, "failed to get settings", http.StatusInternalServerError)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(settings); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if settings.Port == 0 {
		settings.Port = 443
	}
	if settings.Port < 1 || settings.Port > 65535 {
		jsonError(w, "proxy port must be between 1 and 65535", http.StatusBadRequest)
		return
	}

	domain, err := naiveproxy.NormalizeDomain(settings.Domain)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	tlsEmail, err := naiveproxy.NormalizeTLSEmail(settings.TLSEmail)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	decoySite, err := naiveproxy.NormalizeDecoySite(settings.DecoySite)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	settings.Domain = domain
	settings.TLSEmail = tlsEmail
	settings.DecoySite = decoySite

	subPath, err := naiveproxy.NormalizeSubscriptionPath(settings.SubPath)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	settings.SubPath = subPath

	// GenerateCaddyfile rejects this combination, and RegenerateCaddyfile only
	// logs that failure — the save would appear to succeed while Caddy kept
	// serving the previous config. Reject it here instead.
	if settings.PanelPublic && settings.SubPath == settings.PanelPublicPath {
		jsonError(w, "subscription path conflicts with the published panel path", http.StatusBadRequest)
		return
	}

	if err := h.DB.SaveSettings(settings); err != nil {
		log.Printf("Failed to save settings: %v", err)
		jsonError(w, "failed to save settings", http.StatusInternalServerError)
		return
	}

	// Regenerate Caddyfile
	RegenerateCaddyfile(h.DB, h.Manager, h.PanelUpstream)

	jsonSuccess(w, "settings updated", settings)
}

// ServeHTTP routes settings requests
func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.Get(w, r)
	case http.MethodPut:
		h.Update(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
