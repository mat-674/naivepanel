package database

import (
	"errors"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"naivepanel/internal/models"
)

// openTestDB opens a fresh database in a temp dir and closes it on cleanup.
func openTestDB(t *testing.T) (*DB, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New(%q) = %v", path, err)
	}
	t.Cleanup(func() { db.Close() })
	return db, path
}

func TestSettingsRoundTrip(t *testing.T) {
	db, _ := openTestDB(t)

	want := &models.Settings{
		Domain:          "example.com",
		Port:            8443,
		TLSEmail:        "admin@example.com",
		DecoySite:       "https://decoy.example",
		SubPath:         "mysub",
		PanelPublic:     true,
		PanelPublicPath: "adminpanel",
		PanelBasicUser:  "opsuser",
		PanelBasicHash:  "$2a$10$abcdefghijklmnopqrstuv",
	}

	if err := db.SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	got, err := db.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if *got != *want {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", *got, *want)
	}
}

// A saved Settings with PanelPublic false must read back false, and must not
// leave a stale "true" row behind from a previous publish.
func TestSettingsPanelPublicToggleOff(t *testing.T) {
	db, _ := openTestDB(t)

	on := &models.Settings{Domain: "example.com", Port: 443, PanelPublic: true, PanelPublicPath: "p", PanelBasicUser: "u", PanelBasicHash: "h"}
	if err := db.SaveSettings(on); err != nil {
		t.Fatalf("SaveSettings(on): %v", err)
	}
	got, err := db.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if !got.PanelPublic {
		t.Fatalf("PanelPublic = false after saving true")
	}

	off := *on
	off.PanelPublic = false
	if err := db.SaveSettings(&off); err != nil {
		t.Fatalf("SaveSettings(off): %v", err)
	}
	got, err = db.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.PanelPublic {
		t.Errorf("PanelPublic = true after saving false")
	}
}

// Empty SubPath / PanelPublicPath and a zero Port must come back as the
// documented defaults, both straight off a fresh database and after a save that
// carried empty values.
func TestSettingsDefaults(t *testing.T) {
	t.Run("fresh database", func(t *testing.T) {
		db, _ := openTestDB(t)

		got, err := db.GetSettings()
		if err != nil {
			t.Fatalf("GetSettings: %v", err)
		}
		if got.Port != 443 {
			t.Errorf("Port = %d, want 443", got.Port)
		}
		if got.SubPath != "sub" {
			t.Errorf("SubPath = %q, want \"sub\"", got.SubPath)
		}
		if got.PanelPublicPath != "panel" {
			t.Errorf("PanelPublicPath = %q, want \"panel\"", got.PanelPublicPath)
		}
		if got.PanelPublic {
			t.Errorf("PanelPublic = true on a fresh database")
		}
	})

	t.Run("saved empty values", func(t *testing.T) {
		db, _ := openTestDB(t)

		if err := db.SaveSettings(&models.Settings{Domain: "example.com", Port: 443}); err != nil {
			t.Fatalf("SaveSettings: %v", err)
		}
		got, err := db.GetSettings()
		if err != nil {
			t.Fatalf("GetSettings: %v", err)
		}
		if got.SubPath != "sub" {
			t.Errorf("SubPath = %q, want \"sub\"", got.SubPath)
		}
		if got.PanelPublicPath != "panel" {
			t.Errorf("PanelPublicPath = %q, want \"panel\"", got.PanelPublicPath)
		}
	})
}

// A malformed port row falls back to 443 instead of propagating garbage.
func TestSettingsPortFallback(t *testing.T) {
	for _, stored := range []string{"", "not-a-number", "44 3", "8443x"} {
		db, _ := openTestDB(t)
		if err := db.SetSetting("port", stored); err != nil {
			t.Fatalf("SetSetting(port, %q): %v", stored, err)
		}
		got, err := db.GetSettings()
		if err != nil {
			t.Fatalf("GetSettings: %v", err)
		}
		if got.Port != 443 {
			t.Errorf("stored port %q: Port = %d, want 443", stored, got.Port)
		}
	}
}

// Every field of models.Settings must be covered by settingsFields, so adding a
// field without a mapping entry fails here rather than silently not persisting.
func TestSettingsFieldsCoverAllKeys(t *testing.T) {
	wantKeys := []string{
		"domain", "port", "tls_email", "decoy_site", "sub_path",
		"panel_public", "panel_public_path", "panel_basic_user", "panel_basic_hash",
	}

	if len(settingsFields) != len(wantKeys) {
		t.Fatalf("settingsFields has %d entries, want %d", len(settingsFields), len(wantKeys))
	}
	seen := make(map[string]bool, len(settingsFields))
	for _, f := range settingsFields {
		if seen[f.key] {
			t.Errorf("duplicate key %q in settingsFields", f.key)
		}
		seen[f.key] = true
	}
	for _, k := range wantKeys {
		if !seen[k] {
			t.Errorf("settingsFields is missing key %q", k)
		}
	}
}

// migrate() runs on every New(); it must be safe against an already-migrated
// file, which is what the swallowed "duplicate column name" errors are for.
func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrate.db")

	db, err := New(path)
	if err != nil {
		t.Fatalf("first New: %v", err)
	}
	if err := db.SaveSettings(&models.Settings{Domain: "example.com", Port: 443}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	// Direct re-run on the same handle.
	if err := db.migrate(); err != nil {
		t.Fatalf("second migrate on open handle: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopening an existing file is the real-world path (every panel restart).
	db2, err := New(path)
	if err != nil {
		t.Fatalf("New on existing file: %v", err)
	}
	defer db2.Close()

	if err := db2.migrate(); err != nil {
		t.Fatalf("migrate on reopened file: %v", err)
	}

	// The added columns still work and existing data survived.
	got, err := db2.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings after remigrate: %v", err)
	}
	if got.Domain != "example.com" {
		t.Errorf("Domain = %q after remigrate, want \"example.com\"", got.Domain)
	}
	u, err := db2.CreateUser(models.CreateUserRequest{Username: "u1", Password: "p1", HWIDLimit: 2})
	if err != nil {
		t.Fatalf("CreateUser after remigrate: %v", err)
	}
	if _, err := db2.GetUserBySubToken(u.SubToken); err != nil {
		t.Fatalf("GetUserBySubToken (sub_token column): %v", err)
	}
}

func TestIsDuplicateColumnErr(t *testing.T) {
	db, _ := openTestDB(t)

	if _, err := db.conn.Exec("ALTER TABLE proxy_users ADD COLUMN hwid_limit INTEGER DEFAULT 0"); err == nil {
		t.Fatal("re-adding hwid_limit succeeded; expected a duplicate column error")
	} else if !isDuplicateColumnErr(err) {
		t.Errorf("isDuplicateColumnErr(%q) = false, want true", err)
	}

	// A genuine failure must not be mistaken for the idempotency case.
	if _, err := db.conn.Exec("ALTER TABLE no_such_table ADD COLUMN x INTEGER"); err == nil {
		t.Fatal("altering a missing table succeeded")
	} else if isDuplicateColumnErr(err) {
		t.Errorf("isDuplicateColumnErr(%q) = true for a non-duplicate error", err)
	}

	if isDuplicateColumnErr(nil) {
		t.Error("isDuplicateColumnErr(nil) = true")
	}
	if isDuplicateColumnErr(errors.New("disk I/O error")) {
		t.Error("isDuplicateColumnErr(disk I/O error) = true")
	}
}

// Probe for "database is locked" under concurrent writers on the single
// *sql.DB from New() (WAL + _busy_timeout=5000, connection pool left at Go's
// defaults). Failures here would justify SetMaxOpenConns(1).
func TestConcurrentWrites(t *testing.T) {
	db, _ := openTestDB(t)

	const writers = 16
	const iterations = 25

	var wg sync.WaitGroup
	errs := make(chan error, writers*iterations)

	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				s := &models.Settings{
					Domain:          "example.com",
					Port:            443,
					SubPath:         "sub",
					PanelPublic:     i%2 == 0,
					PanelPublicPath: "panel",
				}
				if err := db.SaveSettings(s); err != nil {
					errs <- err
					return
				}
				if _, err := db.GetSettings(); err != nil {
					errs <- err
					return
				}
				if _, err := db.CreateUser(models.CreateUserRequest{
					Username: "user-" + strconv.Itoa(w) + "-" + strconv.Itoa(i),
					Password: "password",
				}); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent write failed: %v", err)
	}

	n, err := db.UserCount()
	if err != nil {
		t.Fatalf("UserCount: %v", err)
	}
	if n != writers*iterations {
		t.Errorf("UserCount = %d, want %d", n, writers*iterations)
	}
}
