package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"naivepanel/internal/config"
	"naivepanel/internal/database"
	"naivepanel/internal/models"
	"naivepanel/internal/naiveproxy"
	"net/http"
	"strconv"
	"strings"
)

// UserHandler handles proxy user CRUD operations
type UserHandler struct {
	DB      *database.DB
	Manager *naiveproxy.Manager
}

// ServeHTTP routes user requests
func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/users or /api/users/{id} or /api/users/{id}/link
	path := strings.TrimPrefix(r.URL.Path, "/api/users")
	path = strings.TrimPrefix(path, "/")

	switch {
	case path == "" && r.Method == http.MethodGet:
		h.List(w, r)
	case path == "" && r.Method == http.MethodPost:
		h.Create(w, r)
	case r.Method == http.MethodPut:
		id, err := parseID(path)
		if err != nil {
			jsonError(w, "invalid user id", http.StatusBadRequest)
			return
		}
		h.Update(w, r, id)
	case r.Method == http.MethodDelete:
		id, err := parseID(path)
		if err != nil {
			jsonError(w, "invalid user id", http.StatusBadRequest)
			return
		}
		h.Delete(w, r, id)
	case strings.HasSuffix(path, "/link"):
		idStr := strings.TrimSuffix(path, "/link")
		id, err := parseID(idStr)
		if err != nil {
			jsonError(w, "invalid user id", http.StatusBadRequest)
			return
		}
		h.GetLink(w, r, id)
	case strings.HasSuffix(path, "/hwid/reset") && r.Method == http.MethodPost:
		idStr := strings.TrimSuffix(path, "/hwid/reset")
		id, err := parseID(idStr)
		if err != nil {
			jsonError(w, "invalid user id", http.StatusBadRequest)
			return
		}
		h.ResetHWID(w, r, id)
	default:
		jsonError(w, "not found", http.StatusNotFound)
	}
}

// List handles GET /api/users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.DB.ListUsers()
	if err != nil {
		jsonError(w, "failed to list users", http.StatusInternalServerError)
		return
	}

	if users == nil {
		users = []models.ProxyUser{}
	}

	jsonSuccess(w, "", users)
}

// Create handles POST /api/users
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		req.Username = config.GenerateRandomUsername(8)
	}
	if req.Password == "" {
		req.Password = config.GenerateRandomPassword(16)
	}

	user, err := h.DB.CreateUser(req)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	// Regenerate Caddyfile
	h.regenerateCaddyfile()

	jsonResponse(w, models.APIResponse{
		Success: true,
		Message: "user created",
		Data:    user,
	}, http.StatusCreated)
}

// Update handles PUT /api/users/{id}
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request, id int64) {
	// Check if user exists
	if _, err := h.DB.GetUser(id); err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.DB.UpdateUser(id, req); err != nil {
		jsonError(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	// Regenerate Caddyfile
	h.regenerateCaddyfile()

	user, _ := h.DB.GetUser(id)
	jsonSuccess(w, "user updated", user)
}

// Delete handles DELETE /api/users/{id}
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request, id int64) {
	if _, err := h.DB.GetUser(id); err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	if err := h.DB.DeleteUser(id); err != nil {
		jsonError(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	// Regenerate Caddyfile
	h.regenerateCaddyfile()

	jsonSuccess(w, "user deleted", nil)
}

// GetLink handles GET /api/users/{id}/link
func (h *UserHandler) GetLink(w http.ResponseWriter, r *http.Request, id int64) {
	user, err := h.DB.GetUser(id)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	settings, err := h.DB.GetSettings()
	if err != nil {
		jsonError(w, "failed to get settings", http.StatusInternalServerError)
		return
	}

	domain := settings.Domain
	if domain == "" {
		domain = "your-server-ip"
	}
	port := settings.Port
	if port == 0 {
		port = 443
	}

	uri := fmt.Sprintf("naive+https://%s:%s@%s:%d?padding=true", user.Username, user.Password, domain, port)

	// Generate QR code as base64
	qrBase64, err := GenerateQRBase64(uri)
	if err != nil {
		log.Printf("Failed to generate QR code: %v", err)
		qrBase64 = ""
	}

	jsonSuccess(w, "", models.UserLink{
		URI:    uri,
		QRCode: qrBase64,
	})
}

func (h *UserHandler) regenerateCaddyfile() {
	settings, err := h.DB.GetSettings()
	if err != nil {
		log.Printf("Failed to get settings for caddyfile regen: %v", err)
		return
	}

	users, err := h.DB.GetEnabledUsers()
	if err != nil {
		log.Printf("Failed to get users for caddyfile regen: %v", err)
		return
	}

	content, err := naiveproxy.GenerateCaddyfile(settings, users)
	if err != nil {
		log.Printf("Failed to generate caddyfile: %v", err)
		return
	}

	if err := h.Manager.WriteCaddyfile(content); err != nil {
		log.Printf("Failed to write caddyfile: %v", err)
		return
	}

	// Reload if running
	if h.Manager.IsRunning() {
		if err := h.Manager.Reload(); err != nil {
			log.Printf("Failed to reload naiveproxy: %v", err)
		}
	}
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ResetHWID handles POST /api/users/{id}/hwid/reset
func (h *UserHandler) ResetHWID(w http.ResponseWriter, r *http.Request, id int64) {
	if _, err := h.DB.GetUser(id); err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	if err := h.DB.ResetHWIDs(id); err != nil {
		jsonError(w, "failed to reset HWIDs", http.StatusInternalServerError)
		return
	}
	
	h.DB.UpdateHWIDResetTime(id)

	jsonSuccess(w, "HWIDs reset successfully", nil)
}
