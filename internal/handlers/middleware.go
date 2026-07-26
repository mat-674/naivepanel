package handlers

import (
	"encoding/json"
	"log"
	"naivepanel/internal/auth"
	"naivepanel/internal/models"
	"net/http"
	"strings"
)

// MaxRequestSize caps request bodies. Every write endpoint takes a small JSON
// document, so this only ever trips on a malformed or hostile request.
const MaxRequestSize = 1 << 20 // 1MB

// Harden wraps the whole server: it caps request bodies and sets response
// headers that apply to the SPA, the subscription route and the API alike.
// Applying it once at the mux boundary keeps individual route registrations
// readable and means a new route cannot forget it.
func Harden(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestSize)

		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

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
