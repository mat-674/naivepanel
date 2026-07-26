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
	DB            *database.DB
	Manager       *naiveproxy.Manager
	PanelUpstream string
}

// ServeHTTP routes user requests. The path shape is resolved first and the
// method second: a known path reached with the wrong method answers 405, an
// unknown path 404, and only an unparseable id answers 400.
func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip the mount prefix, leaving "", "{id}", "{id}/link" or
	// "{id}/hwid/reset".
	path := strings.TrimPrefix(r.URL.Path, "/api/users")
	path = strings.TrimPrefix(path, "/")

	// The collection itself carries no id.
	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.List(w, r)
		case http.MethodPost:
			h.Create(w, r)
		default:
			methodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}

	idStr, action := path, ""
	if i := strings.Index(path, "/"); i >= 0 {
		idStr, action = path[:i], path[i+1:]
	}

	// Resolve the action to the handler for this method, plus the methods the
	// action does accept so a mismatch can advertise them.
	var handler func(w http.ResponseWriter, r *http.Request, id int64)
	var allow []string
	switch action {
	case "":
		allow = []string{http.MethodPut, http.MethodDelete}
		switch r.Method {
		case http.MethodPut:
			handler = h.Update
		case http.MethodDelete:
			handler = h.Delete
		}
	case "link":
		allow = []string{http.MethodGet}
		if r.Method == http.MethodGet {
			handler = h.GetLink
		}
	case "hwid/reset":
		allow = []string{http.MethodPost}
		if r.Method == http.MethodPost {
			handler = h.ResetHWID
		}
	default:
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	if handler == nil {
		methodNotAllowed(w, allow...)
		return
	}

	id, err := parseID(idStr)
	if err != nil {
		jsonError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	handler(w, r, id)
}

// methodNotAllowed answers 405 and advertises the methods the path does accept.
func methodNotAllowed(w http.ResponseWriter, allow ...string) {
	w.Header().Set("Allow", strings.Join(allow, ", "))
	jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
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

	if err := naiveproxy.ValidateProxyUsername(req.Username); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := naiveproxy.ValidateProxyPassword(req.Password); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.DB.CreateUser(req)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		jsonError(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	// Regenerate Caddyfile
	RegenerateCaddyfile(h.DB, h.Manager, h.PanelUpstream)

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

	if req.Password != nil {
		if err := naiveproxy.ValidateProxyPassword(*req.Password); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := h.DB.UpdateUser(id, req); err != nil {
		jsonError(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	// Regenerate Caddyfile
	RegenerateCaddyfile(h.DB, h.Manager, h.PanelUpstream)

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
	RegenerateCaddyfile(h.DB, h.Manager, h.PanelUpstream)

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
