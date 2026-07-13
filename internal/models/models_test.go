package models

import (
	"encoding/json"
	"testing"
)

// TestUpdateUserRequestExpiresAtTriState locks in the fix for clearing an
// expiration: an omitted key must leave the field untouched, an explicit null
// must clear it, and a value must set it. The old *int64 field could not tell
// the first two apart, so a cleared expiry was silently ignored.
func TestUpdateUserRequestExpiresAtTriState(t *testing.T) {
	t.Run("omitted leaves field untouched", func(t *testing.T) {
		var req UpdateUserRequest
		if err := json.Unmarshal([]byte(`{"enabled":true}`), &req); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if req.ExpiresAt.Present {
			t.Fatal("ExpiresAt.Present = true for an omitted key, want false")
		}
	})

	t.Run("explicit null clears field", func(t *testing.T) {
		var req UpdateUserRequest
		if err := json.Unmarshal([]byte(`{"expires_at":null}`), &req); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if !req.ExpiresAt.Present {
			t.Fatal("ExpiresAt.Present = false for explicit null, want true")
		}
		if req.ExpiresAt.Value != nil {
			t.Fatalf("ExpiresAt.Value = %v for null, want nil", *req.ExpiresAt.Value)
		}
	})

	t.Run("value sets field", func(t *testing.T) {
		var req UpdateUserRequest
		if err := json.Unmarshal([]byte(`{"expires_at":1893456000}`), &req); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if !req.ExpiresAt.Present {
			t.Fatal("ExpiresAt.Present = false for a value, want true")
		}
		if req.ExpiresAt.Value == nil || *req.ExpiresAt.Value != 1893456000 {
			t.Fatalf("ExpiresAt.Value = %v, want 1893456000", req.ExpiresAt.Value)
		}
	})
}
