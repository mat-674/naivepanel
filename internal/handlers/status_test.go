package handlers

import (
	"encoding/json"
	"io"
	"log"
	"naivepanel/internal/models"
	"naivepanel/internal/naiveproxy"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// muteLog silences the panel log for one test. These tests deliberately trigger
// failures whose detail is supposed to be logged, and that noise is not a result.
func muteLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

// serviceAction drives StatusHandler.ServiceAction against a manager built over
// paths that do not exist, so every lifecycle call fails. It returns the status,
// the decoded body, and the temp dir whose path must not appear in that body.
func serviceAction(t *testing.T, action string) (int, models.APIResponse, string) {
	t.Helper()
	muteLog(t)

	dir := t.TempDir()
	h := &StatusHandler{
		Manager: naiveproxy.NewManager(
			filepath.Join(dir, "naive"),
			filepath.Join(dir, "Caddyfile"),
		),
	}

	rec := httptest.NewRecorder()
	h.ServiceAction(rec, httptest.NewRequest(http.MethodPost, "/api/service/"+action, nil))

	var body models.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v (%q)", err, rec.Body.String())
	}
	return rec.Code, body, dir
}

func TestServiceActionHidesManagerInternals(t *testing.T) {
	// Both paths below are internals: an operator debugging a failed start reads
	// the panel log, the HTTP client must not learn filesystem layout.
	for _, action := range []string{"start", "stop", "restart"} {
		t.Run(action, func(t *testing.T) {
			code, body, dir := serviceAction(t, action)

			if code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", code)
			}
			if body.Success {
				t.Fatal("failed action reported success")
			}
			want := "failed to " + action + " naiveproxy, check the panel log"
			if body.Message != want {
				t.Fatalf("message = %q, want %q", body.Message, want)
			}
			if strings.Contains(body.Message, dir) {
				t.Fatalf("message leaked the install path: %q", body.Message)
			}
		})
	}
}

func TestServiceActionRelaysUpdateGuidance(t *testing.T) {
	// Without the systemd units UpdatePanel cannot hand off to systemd-run. That
	// is a deployment fact the operator can act on, so it is relayed verbatim.
	code, body, _ := serviceAction(t, "update")

	if code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", code)
	}
	if body.Message != naiveproxy.ErrUpdateUnsupported.Error() {
		t.Fatalf("message = %q, want the ErrUpdateUnsupported text", body.Message)
	}
	if !strings.Contains(body.Message, "--update") {
		t.Fatalf("message dropped the actionable hint: %q", body.Message)
	}
}

func TestServiceActionRejectsUnknownAction(t *testing.T) {
	code, body, _ := serviceAction(t, "reticulate")

	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
	if !strings.Contains(body.Message, "invalid action") {
		t.Fatalf("message = %q, want an invalid-action complaint", body.Message)
	}
}

func TestServiceActionRejectsNonPost(t *testing.T) {
	muteLog(t)

	dir := t.TempDir()
	h := &StatusHandler{
		Manager: naiveproxy.NewManager(filepath.Join(dir, "naive"), filepath.Join(dir, "Caddyfile")),
	}

	rec := httptest.NewRecorder()
	h.ServiceAction(rec, httptest.NewRequest(http.MethodGet, "/api/service/start", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
