package config

import (
	"path/filepath"
	"testing"
)

func TestGeneratedDefaultsKeepPanelOnLoopback(t *testing.T) {
	cfg := generateDefaults()

	if cfg.PanelBind != "127.0.0.1" {
		t.Fatalf("PanelBind = %q, want loopback default", cfg.PanelBind)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("generated config is invalid: %v", err)
	}
}

func TestOverrideDataDirUpdatesDefaultDependentPaths(t *testing.T) {
	cfg := &Config{
		PanelPort:     12345,
		PanelBind:     DefaultPanelBind(),
		DataDir:       "/opt/naivepanel",
		NaiveBinary:   "/opt/naivepanel/naive",
		CaddyfilePath: "/opt/naivepanel/Caddyfile",
	}

	cfg.OverrideDataDir("/tmp/naivepanel")

	if got, want := cfg.DataDir, "/tmp/naivepanel"; got != want {
		t.Fatalf("DataDir = %q, want %q", got, want)
	}
	if got, want := cfg.NaiveBinary, filepath.Join("/tmp/naivepanel", "naive"); got != want {
		t.Fatalf("NaiveBinary = %q, want %q", got, want)
	}
	if got, want := cfg.CaddyfilePath, filepath.Join("/tmp/naivepanel", "Caddyfile"); got != want {
		t.Fatalf("CaddyfilePath = %q, want %q", got, want)
	}
}

func TestValidateRejectsPublicPanelBind(t *testing.T) {
	cfg := &Config{PanelPort: 12345, PanelBind: "0.0.0.0"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected public panel bind to be rejected")
	}
}
