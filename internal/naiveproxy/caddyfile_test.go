package naiveproxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"naivepanel/internal/models"
)

func TestGenerateCaddyfileRoutesSubscriptionsToLoopbackPanel(t *testing.T) {
	settings := &models.Settings{
		Domain:    "proxy.example.com",
		Port:      443,
		TLSEmail:  "admin@example.com",
		DecoySite: "https://www.example.com",
		SubPath:   "subscriptions/v1",
	}
	users := []models.ProxyUser{{Username: "alice", Password: "secret"}}

	got, err := GenerateCaddyfile(settings, users, "127.0.0.1:43979")
	if err != nil {
		t.Fatalf("GenerateCaddyfile() error = %v", err)
	}

	for _, want := range []string{
		"handle /subscriptions/v1/* {",
		"reverse_proxy 127.0.0.1:43979",
		"handle {",
		"forward_proxy {",
		"basic_auth alice secret",
		"reverse_proxy https://www.example.com",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated Caddyfile does not contain %q:\n%s", want, got)
		}
	}

	if strings.Index(got, "handle /subscriptions/v1/*") > strings.Index(got, "forward_proxy {") {
		t.Errorf("subscription route must precede the proxy fallback:\n%s", got)
	}
}

func TestNormalizeSubscriptionPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"", "sub", true},
		{"/subscriptions/v1/", "subscriptions/v1", true},
		{"private_link-1", "private_link-1", true},
		{"../sub", "", false},
		{"sub path", "", false},
		{"sub\nroute", "", false},
	}

	for _, tt := range tests {
		got, err := NormalizeSubscriptionPath(tt.input)
		if tt.ok {
			if err != nil || got != tt.want {
				t.Errorf("NormalizeSubscriptionPath(%q) = %q, %v; want %q, nil", tt.input, got, err, tt.want)
			}
			continue
		}
		if err == nil {
			t.Errorf("NormalizeSubscriptionPath(%q) succeeded, want error", tt.input)
		}
	}
}

func TestGenerateCaddyfileRejectsUnsafeSettings(t *testing.T) {
	settings := &models.Settings{Domain: "proxy.example.com\nrespond bad", Port: 443}
	if _, err := GenerateCaddyfile(settings, nil, "127.0.0.1:43979"); err == nil {
		t.Fatal("expected unsafe domain to be rejected")
	}
}

func TestGenerateCaddyfileRejectsUnsafeCredentials(t *testing.T) {
	settings := &models.Settings{Domain: "proxy.example.com", Port: 443}
	cases := []models.ProxyUser{
		{Username: "alice", Password: "secret pass"},           // space breaks the token
		{Username: "alice", Password: "line\nreverse_proxy x"}, // newline injects a directive
		{Username: "al ice", Password: "secret"},               // space in username
		{Username: "alice", Password: ""},                      // empty password
	}
	for _, u := range cases {
		if _, err := GenerateCaddyfile(settings, []models.ProxyUser{u}, "127.0.0.1:43979"); err == nil {
			t.Errorf("expected credentials %+v to be rejected", u)
		}
	}
}

func TestValidateProxyCredentials(t *testing.T) {
	valid := []string{"alice", "user_1", "a-b.c~d", "AbC123"}
	for _, s := range valid {
		if err := ValidateProxyUsername(s); err != nil {
			t.Errorf("ValidateProxyUsername(%q) = %v, want nil", s, err)
		}
		if err := ValidateProxyPassword(s); err != nil {
			t.Errorf("ValidateProxyPassword(%q) = %v, want nil", s, err)
		}
	}

	invalid := []string{"", "with space", "new\nline", "quote\"", "at@sign", "colon:x", "slash/x", "back\\slash"}
	for _, s := range invalid {
		if err := ValidateProxyUsername(s); err == nil {
			t.Errorf("ValidateProxyUsername(%q) = nil, want error", s)
		}
		if err := ValidateProxyPassword(s); err == nil {
			t.Errorf("ValidateProxyPassword(%q) = nil, want error", s)
		}
	}
}

func TestWriteCaddyfileReplacesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Caddyfile")
	manager := NewManager("", path)

	if err := manager.WriteCaddyfile("first"); err != nil {
		t.Fatalf("first WriteCaddyfile() error = %v", err)
	}
	if err := manager.WriteCaddyfile("second"); err != nil {
		t.Fatalf("second WriteCaddyfile() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("Caddyfile = %q, want replacement content", got)
	}
}
