package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"naivepanel/internal/database"
	"naivepanel/internal/models"
	"naivepanel/internal/naiveproxy"
)

// newUserHandler builds a UserHandler backed by a real SQLite file and a
// Caddyfile under t.TempDir(), so the mutating branches can run their
// regenerate step without touching anything outside the test.
func newUserHandler(t *testing.T) (*UserHandler, *models.ProxyUser) {
	t.Helper()

	dir := t.TempDir()
	db, err := database.New(filepath.Join(dir, "panel.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	user, err := db.CreateUser(models.CreateUserRequest{Username: "alice", Password: "s3cret"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	h := &UserHandler{
		DB:            db,
		Manager:       naiveproxy.NewManager(filepath.Join(dir, "naive"), filepath.Join(dir, "Caddyfile")),
		PanelUpstream: "127.0.0.1:8080",
	}
	return h, user
}

// doUserRequest dispatches one request through ServeHTTP and returns the recorder.
func doUserRequest(t *testing.T, h *UserHandler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// decodeUserResponse unmarshals the envelope every branch is expected to answer with.
func decodeUserResponse(t *testing.T, rec *httptest.ResponseRecorder) models.APIResponse {
	t.Helper()

	var resp models.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return resp
}

// TestDispatchReachesEveryLiveEndpoint pins the six endpoints the SPA calls to
// their handler, status code and message.
func TestDispatchReachesEveryLiveEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		target      string
		body        string
		wantStatus  int
		wantMessage string
	}{
		{name: "list", method: http.MethodGet, target: "/api/users", wantStatus: http.StatusOK},
		{name: "create", method: http.MethodPost, target: "/api/users", body: `{}`,
			wantStatus: http.StatusCreated, wantMessage: "user created"},
		{name: "update", method: http.MethodPut, target: "/api/users/1", body: `{"hwid_limit":3}`,
			wantStatus: http.StatusOK, wantMessage: "user updated"},
		{name: "delete", method: http.MethodDelete, target: "/api/users/1",
			wantStatus: http.StatusOK, wantMessage: "user deleted"},
		{name: "link", method: http.MethodGet, target: "/api/users/1/link", wantStatus: http.StatusOK},
		{name: "hwid reset", method: http.MethodPost, target: "/api/users/1/hwid/reset",
			wantStatus: http.StatusOK, wantMessage: "HWIDs reset successfully"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, user := newUserHandler(t)
			if user.ID != 1 {
				t.Fatalf("fixture user id = %d, want 1", user.ID)
			}

			rec := doUserRequest(t, h, tc.method, tc.target, tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			resp := decodeUserResponse(t, rec)
			if !resp.Success {
				t.Errorf("success = false, want true (message %q)", resp.Message)
			}
			if resp.Message != tc.wantMessage {
				t.Errorf("message = %q, want %q", resp.Message, tc.wantMessage)
			}
		})
	}
}

// TestDispatchWrongMethodOnKnownPath covers the bug class where a method check
// ran before the path check: PUT /api/users/{id}/link used to be parsed as the
// id "1/link" and answered 400 invalid user id.
func TestDispatchWrongMethodOnKnownPath(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		target    string
		wantAllow string
	}{
		{name: "put on collection", method: http.MethodPut, target: "/api/users", wantAllow: "GET, POST"},
		{name: "delete on collection", method: http.MethodDelete, target: "/api/users", wantAllow: "GET, POST"},
		{name: "get on user", method: http.MethodGet, target: "/api/users/1", wantAllow: "PUT, DELETE"},
		{name: "post on user", method: http.MethodPost, target: "/api/users/1", wantAllow: "PUT, DELETE"},
		{name: "put on link", method: http.MethodPut, target: "/api/users/1/link", wantAllow: "GET"},
		{name: "post on link", method: http.MethodPost, target: "/api/users/1/link", wantAllow: "GET"},
		{name: "delete on link", method: http.MethodDelete, target: "/api/users/1/link", wantAllow: "GET"},
		{name: "delete on hwid reset", method: http.MethodDelete, target: "/api/users/1/hwid/reset", wantAllow: "POST"},
		{name: "get on hwid reset", method: http.MethodGet, target: "/api/users/1/hwid/reset", wantAllow: "POST"},
		{name: "put on hwid reset", method: http.MethodPut, target: "/api/users/1/hwid/reset", wantAllow: "POST"},
	}

	h, _ := newUserHandler(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := doUserRequest(t, h, tc.method, tc.target, "")
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want 405 (body %q)", rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("Allow"); got != tc.wantAllow {
				t.Errorf("Allow = %q, want %q", got, tc.wantAllow)
			}
			if resp := decodeUserResponse(t, rec); resp.Success {
				t.Error("success = true on a 405")
			}
		})
	}
}

// TestDispatchUnknownPath keeps unrecognised shapes on 404, whatever the method.
func TestDispatchUnknownPath(t *testing.T) {
	targets := []struct {
		method string
		target string
	}{
		{http.MethodGet, "/api/users/1/bogus"},
		{http.MethodPost, "/api/users/1/bogus"},
		{http.MethodGet, "/api/users/1/hwid"},
		{http.MethodPost, "/api/users/1/hwid"},
		{http.MethodPost, "/api/users/1/hwid/reset/again"},
		{http.MethodGet, "/api/users/1/link/qr"},
		{http.MethodPut, "/api/users/1/2"},
	}

	h, _ := newUserHandler(t)
	for _, tc := range targets {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			rec := doUserRequest(t, h, tc.method, tc.target, "")
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
			}
			if resp := decodeUserResponse(t, rec); resp.Success {
				t.Error("success = true on a 404")
			}
		})
	}
}

// TestDispatchInvalidID reserves 400 for an id that cannot be parsed, on a path
// and method that are otherwise live.
func TestDispatchInvalidID(t *testing.T) {
	targets := []struct {
		method string
		target string
	}{
		{http.MethodPut, "/api/users/abc"},
		{http.MethodDelete, "/api/users/abc"},
		{http.MethodGet, "/api/users/abc/link"},
		{http.MethodPost, "/api/users/abc/hwid/reset"},
		{http.MethodPut, "/api/users/1.5"},
		{http.MethodGet, "/api/users/%20/link"},
	}

	h, _ := newUserHandler(t)
	for _, tc := range targets {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			rec := doUserRequest(t, h, tc.method, tc.target, "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
			}
			if resp := decodeUserResponse(t, rec); resp.Message != "invalid user id" {
				t.Errorf("message = %q, want %q", resp.Message, "invalid user id")
			}
		})
	}
}

// TestUnknownUserStillResolvesToItsHandler guards against a dispatcher that
// short-circuits on the id: a well-formed id for a missing row is the handler's
// 404, not the router's.
func TestUnknownUserStillResolvesToItsHandler(t *testing.T) {
	h, _ := newUserHandler(t)

	for _, tc := range []struct {
		method string
		target string
	}{
		{http.MethodPut, "/api/users/999"},
		{http.MethodDelete, "/api/users/999"},
		{http.MethodGet, "/api/users/999/link"},
		{http.MethodPost, "/api/users/999/hwid/reset"},
	} {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			rec := doUserRequest(t, h, tc.method, tc.target, `{}`)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
			}
			if resp := decodeUserResponse(t, rec); resp.Message != "user not found" {
				t.Errorf("message = %q, want %q", resp.Message, "user not found")
			}
		})
	}
}
