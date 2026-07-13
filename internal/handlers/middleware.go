package handlers

import (
	"encoding/json"
	"log"
	"naivepanel/internal/auth"
	"naivepanel/internal/models"
	"net/http"
	"strings"
)

// AuthMiddleware validates JWT tokens on protected routes
func AuthMiddleware(jwtSecret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			jsonError(w, "authorization required", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			jsonError(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		_, err := auth.ValidateToken(parts[1], jwtSecret)
		if err != nil {
			jsonError(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// jsonResponse sends a JSON response
func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// jsonError sends a JSON error response
func jsonError(w http.ResponseWriter, message string, status int) {
	jsonResponse(w, models.APIResponse{
		Success: false,
		Message: message,
	}, status)
}

// jsonSuccess sends a JSON success response
func jsonSuccess(w http.ResponseWriter, message string, data interface{}) {
	jsonResponse(w, models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}, http.StatusOK)
}
