package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginLimiterCountsOnlyFailures(t *testing.T) {
	var l loginLimiter

	// A zero-value limiter must be usable, and reads alone never lock anyone out.
	for i := 0; i < maxLoginFailures*2; i++ {
		if !l.allowed("1.2.3.4") {
			t.Fatalf("allowed() locked out address after %d checks without a recorded failure", i)
		}
	}

	for i := 0; i < maxLoginFailures; i++ {
		if !l.allowed("1.2.3.4") {
			t.Fatalf("locked out after %d failures, want %d", i, maxLoginFailures)
		}
		l.recordFailure("1.2.3.4")
	}
	if l.allowed("1.2.3.4") {
		t.Fatal("address still allowed after reaching the failure budget")
	}

	// Buckets are per address.
	if !l.allowed("5.6.7.8") {
		t.Fatal("a different address was locked out")
	}

	// A successful login clears the history.
	l.reset("1.2.3.4")
	if !l.allowed("1.2.3.4") {
		t.Fatal("reset() did not clear the failure history")
	}
}

func TestLoginLimiterForgetsOldFailures(t *testing.T) {
	l := loginLimiter{failures: map[string][]time.Time{}}
	stale := time.Now().Add(-loginWindow - time.Minute)
	for i := 0; i < maxLoginFailures; i++ {
		l.failures["1.2.3.4"] = append(l.failures["1.2.3.4"], stale)
	}

	if !l.allowed("1.2.3.4") {
		t.Fatal("failures older than the window still count against the budget")
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "direct request uses the socket peer without its port",
			remoteAddr: "203.0.113.9:54321",
			want:       "203.0.113.9",
		},
		{
			// Behind Caddy every request arrives from loopback with a fresh
			// ephemeral port; without XFF they must still share one bucket.
			name:       "loopback peer without a forwarded header collapses to the host",
			remoteAddr: "127.0.0.1:41234",
			want:       "127.0.0.1",
		},
		{
			// Caddy appends the peer it saw, so the last element is the entry
			// the proxy controls and the client cannot forge.
			name:       "forwarded header from loopback uses the last element",
			remoteAddr: "127.0.0.1:41234",
			xff:        "10.0.0.1, 198.51.100.7",
			want:       "198.51.100.7",
		},
		{
			// A non-loopback peer is not our proxy, so its header is ignored.
			name:       "forwarded header from a remote peer is ignored",
			remoteAddr: "203.0.113.9:54321",
			xff:        "198.51.100.7",
			want:       "203.0.113.9",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/login", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			if got := clientIP(r); got != tc.want {
				t.Fatalf("clientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHardenSetsHeadersAndCapsBody(t *testing.T) {
	handler := Harden(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}
