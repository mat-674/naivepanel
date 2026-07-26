package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"naivepanel/internal/auth"
	"naivepanel/internal/database"
	"naivepanel/internal/models"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	DB        *database.DB
	JWTSecret string

	limiter loginLimiter
}

const (
	// maxLoginFailures is how many failed logins one client address may
	// accumulate inside loginWindow before it is locked out. Successful
	// logins are not counted, so an operator who signs in normally never
	// walks into the limit.
	maxLoginFailures = 10
	loginWindow      = 15 * time.Minute
)

// loginLimiter throttles failed logins per client address. The zero value is
// usable: the map is created under the mutex on first write, so a handler
// built as a plain struct literal is still race-free.
type loginLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

// allowed reports whether key is still under the failure budget.
func (l *loginLimiter) allowed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(key, time.Now())) < maxLoginFailures
}

// recordFailure adds one failed attempt for key.
func (l *loginLimiter) recordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if l.failures == nil {
		l.failures = make(map[string][]time.Time)
	}
	l.failures[key] = append(l.recent(key, now), now)

	// Drop addresses whose failures have all aged out so a long-running
	// panel under a spray of source addresses cannot grow the map forever.
	if len(l.failures) > 1000 {
		for k := range l.failures {
			if len(l.recent(k, now)) == 0 {
				delete(l.failures, k)
			}
		}
	}
}

// reset clears the failure history for key after a successful login.
func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}

// recent returns the failures for key inside the current window. Callers must
// hold l.mu.
func (l *loginLimiter) recent(key string, now time.Time) []time.Time {
	windowStart := now.Add(-loginWindow)
	kept := l.failures[key][:0:0]
	for _, t := range l.failures[key] {
		if t.After(windowStart) {
			kept = append(kept, t)
		}
	}
	return kept
}

// clientIP resolves the address that login attempts are counted against.
// The panel listens on loopback and is reached through Caddy, so RemoteAddr
// is normally 127.0.0.1 with a fresh ephemeral port on every request — using
// it verbatim would give each attempt its own bucket and defeat the limit.
// Caddy appends the peer it saw to X-Forwarded-For, so the last element is
// the one entry the proxy controls; earlier elements are client-supplied and
// forgeable. Fall back to the socket peer for direct (tunnelled) requests.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return host
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
			return last
		}
	}
	return host
}

// Login handles POST /api/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	addr := clientIP(r)
	if !h.limiter.allowed(addr) {
		jsonError(w, "too many failed login attempts, try again later", http.StatusTooManyRequests)
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
		log.Printf("Login failed to read admin: %v", err)
		h.limiter.recordFailure(addr)
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Compare both factors before branching so a wrong username costs the
	// same as a wrong password, and use a constant-time comparison for the
	// username (bcrypt already provides that property for the password).
	usernameMatch := subtle.ConstantTimeCompare([]byte(admin.Username), []byte(req.Username)) == 1
	passwordMatch := auth.ComparePassword(admin.PasswordHash, req.Password)
	if !usernameMatch || !passwordMatch {
		h.limiter.recordFailure(addr)
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(admin.ID, h.JWTSecret)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	h.limiter.reset(addr)
	jsonSuccess(w, "login successful", models.LoginResponse{
		Token:   token,
		Message: "login successful",
	})
}
