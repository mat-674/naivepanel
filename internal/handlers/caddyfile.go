package handlers

import (
	"log"
	"naivepanel/internal/database"
	"naivepanel/internal/naiveproxy"
)

// RegenerateCaddyfile is a shared helper that regenerates and reloads the
// Caddyfile. Used by both UserHandler and SettingsHandler to avoid code
// duplication and ensure consistent behavior across mutations.
func RegenerateCaddyfile(db *database.DB, manager *naiveproxy.Manager, panelUpstream string) {
	settings, err := db.GetSettings()
	if err != nil {
		log.Printf("Failed to get settings for caddyfile regen: %v", err)
		return
	}

	users, err := db.GetEnabledUsers()
	if err != nil {
		log.Printf("Failed to get users for caddyfile regen: %v", err)
		return
	}

	content, err := naiveproxy.GenerateCaddyfile(settings, users, panelUpstream)
	if err != nil {
		log.Printf("Failed to generate caddyfile: %v", err)
		return
	}

	if err := manager.WriteCaddyfile(content); err != nil {
		log.Printf("Failed to write caddyfile: %v", err)
		return
	}

	// Reload if running
	if manager.IsRunning() {
		if err := manager.Reload(); err != nil {
			log.Printf("Failed to reload naiveproxy: %v", err)
		}
	}
}
