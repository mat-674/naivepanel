package handlers

import (
	"encoding/json"
	"naivepanel/internal/auth"
	"naivepanel/internal/database"
	"naivepanel/internal/models"
	"net/http"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	DB        *database.DB
	JWTSecret string
}

// Login handles POST /api/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		jsonError(w, "username and password required", http.StatusBadRequest)
		return
	}

	admin, err := h.DB.GetAdmin()
	if err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if admin.Username != req.Username || !auth.ComparePassword(admin.PasswordHash, req.Password) {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(admin.ID, h.JWTSecret)
	if err != nil {
		jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	jsonSuccess(w, "login successful", models.LoginResponse{
		Token:   token,
		Message: "login successful",
	})
}
