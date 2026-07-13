package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"naivepanel/internal/models"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// New opens a SQLite database and runs migrations
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admin (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS proxy_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			traffic_up INTEGER DEFAULT 0,
			traffic_down INTEGER DEFAULT 0,
			traffic_limit INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_hwids (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			hwid TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, hwid)
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}

	for _, q := range queries {
		if _, err := db.conn.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Add new columns to existing proxy_users table (ignoring errors if they already exist)
	db.conn.Exec("ALTER TABLE proxy_users ADD COLUMN hwid_limit INTEGER DEFAULT 0")
	db.conn.Exec("ALTER TABLE proxy_users ADD COLUMN sub_token TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE proxy_users ADD COLUMN expires_at DATETIME")
	db.conn.Exec("ALTER TABLE proxy_users ADD COLUMN hwid_reset_interval INTEGER DEFAULT 0")
	db.conn.Exec("ALTER TABLE proxy_users ADD COLUMN last_hwid_reset DATETIME")

	return nil
}

// --- Admin ---

// GetAdmin retrieves the admin user
func (db *DB) GetAdmin() (*models.Admin, error) {
	var admin models.Admin
	err := db.conn.QueryRow("SELECT id, username, password_hash, created_at FROM admin LIMIT 1").
		Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &admin.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// CreateAdmin creates the admin user
func (db *DB) CreateAdmin(username, passwordHash string) error {
	_, err := db.conn.Exec(
		"INSERT INTO admin (username, password_hash) VALUES (?, ?)",
		username, passwordHash,
	)
	return err
}

// UpdateAdminPassword updates the admin's password hash
func (db *DB) UpdateAdminPassword(passwordHash string) error {
	_, err := db.conn.Exec("UPDATE admin SET password_hash = ?", passwordHash)
	return err
}

// UpdateAdminUsername updates the admin's username
func (db *DB) UpdateAdminUsername(username string) error {
	_, err := db.conn.Exec("UPDATE admin SET username = ?", username)
	return err
}

// --- Proxy Users ---

// ListUsers returns all proxy users
func (db *DB) ListUsers() ([]models.ProxyUser, error) {
	rows, err := db.conn.Query(
		"SELECT id, username, password, traffic_up, traffic_down, traffic_limit, hwid_limit, sub_token, expires_at, hwid_reset_interval, last_hwid_reset, enabled, created_at FROM proxy_users ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.ProxyUser
	for rows.Next() {
		var u models.ProxyUser
		var enabled int
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.TrafficUp, &u.TrafficDown, &u.TrafficLimit, &u.HWIDLimit, &u.SubToken, &u.ExpiresAt, &u.HWIDResetInterval, &u.LastHWIDReset, &enabled, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Enabled = enabled == 1
		users = append(users, u)
	}

	return users, nil
}

// GetUser returns a single proxy user by ID
func (db *DB) GetUser(id int64) (*models.ProxyUser, error) {
	var u models.ProxyUser
	var enabled int
	err := db.conn.QueryRow(
		"SELECT id, username, password, traffic_up, traffic_down, traffic_limit, hwid_limit, sub_token, expires_at, hwid_reset_interval, last_hwid_reset, enabled, created_at FROM proxy_users WHERE id = ?", id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.TrafficUp, &u.TrafficDown, &u.TrafficLimit, &u.HWIDLimit, &u.SubToken, &u.ExpiresAt, &u.HWIDResetInterval, &u.LastHWIDReset, &enabled, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.Enabled = enabled == 1
	return &u, nil
}

// CreateUser creates a new proxy user
func (db *DB) CreateUser(req models.CreateUserRequest) (*models.ProxyUser, error) {
	b := make([]byte, 16)
	rand.Read(b)
	subToken := hex.EncodeToString(b)

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt > 0 {
		tm := time.Unix(*req.ExpiresAt, 0)
		expiresAt = &tm
	}

	result, err := db.conn.Exec(
		"INSERT INTO proxy_users (username, password, traffic_limit, hwid_limit, sub_token, expires_at, hwid_reset_interval) VALUES (?, ?, ?, ?, ?, ?, ?)",
		req.Username, req.Password, req.TrafficLimit, req.HWIDLimit, subToken, expiresAt, req.HWIDResetInterval,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &models.ProxyUser{
		ID:                id,
		Username:          req.Username,
		Password:          req.Password,
		TrafficLimit:      req.TrafficLimit,
		HWIDLimit:         req.HWIDLimit,
		SubToken:          subToken,
		ExpiresAt:         expiresAt,
		HWIDResetInterval: req.HWIDResetInterval,
		Enabled:           true,
		CreatedAt:         time.Now(),
	}, nil
}

// UpdateUser updates a proxy user's fields
func (db *DB) UpdateUser(id int64, req models.UpdateUserRequest) error {
	if req.Password != nil {
		if _, err := db.conn.Exec("UPDATE proxy_users SET password = ? WHERE id = ?", *req.Password, id); err != nil {
			return err
		}
	}
	if req.TrafficLimit != nil {
		if _, err := db.conn.Exec("UPDATE proxy_users SET traffic_limit = ? WHERE id = ?", *req.TrafficLimit, id); err != nil {
			return err
		}
	}
	if req.HWIDLimit != nil {
		if _, err := db.conn.Exec("UPDATE proxy_users SET hwid_limit = ? WHERE id = ?", *req.HWIDLimit, id); err != nil {
			return err
		}
	}
	if req.ExpiresAt.Present {
		var t *time.Time
		if req.ExpiresAt.Value != nil && *req.ExpiresAt.Value > 0 {
			tm := time.Unix(*req.ExpiresAt.Value, 0)
			t = &tm
		}
		if _, err := db.conn.Exec("UPDATE proxy_users SET expires_at = ? WHERE id = ?", t, id); err != nil {
			return err
		}
	}
	if req.HWIDResetInterval != nil {
		if _, err := db.conn.Exec("UPDATE proxy_users SET hwid_reset_interval = ? WHERE id = ?", *req.HWIDResetInterval, id); err != nil {
			return err
		}
	}
	if req.Enabled != nil {
		enabled := 0
		if *req.Enabled {
			enabled = 1
		}
		if _, err := db.conn.Exec("UPDATE proxy_users SET enabled = ? WHERE id = ?", enabled, id); err != nil {
			return err
		}
	}
	return nil
}

// DeleteUser deletes a proxy user by ID
func (db *DB) DeleteUser(id int64) error {
	_, err := db.conn.Exec("DELETE FROM proxy_users WHERE id = ?", id)
	return err
}

// GetEnabledUsers returns only enabled proxy users
func (db *DB) GetEnabledUsers() ([]models.ProxyUser, error) {
	rows, err := db.conn.Query(
		"SELECT id, username, password, traffic_up, traffic_down, traffic_limit, hwid_limit, sub_token, expires_at, hwid_reset_interval, last_hwid_reset, enabled, created_at FROM proxy_users WHERE enabled = 1 ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.ProxyUser
	for rows.Next() {
		var u models.ProxyUser
		var enabled int
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.TrafficUp, &u.TrafficDown, &u.TrafficLimit, &u.HWIDLimit, &u.SubToken, &u.ExpiresAt, &u.HWIDResetInterval, &u.LastHWIDReset, &enabled, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Enabled = true
		users = append(users, u)
	}
	return users, nil
}

// UserCount returns total user count
func (db *DB) UserCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM proxy_users").Scan(&count)
	return count, err
}

// --- Settings ---

// GetSetting retrieves a setting value by key
func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetSetting sets a setting value
func (db *DB) SetSetting(key, value string) error {
	_, err := db.conn.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?",
		key, value, value,
	)
	return err
}

// GetAllSettings returns all settings as a map
func (db *DB) GetAllSettings() (map[string]string, error) {
	rows, err := db.conn.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}

// GetSettings returns parsed Settings struct
func (db *DB) GetSettings() (*models.Settings, error) {
	all, err := db.GetAllSettings()
	if err != nil {
		return nil, err
	}

	port := 443
	if v, ok := all["port"]; ok {
		fmt.Sscanf(v, "%d", &port)
	}

	subPath := all["sub_path"]
	if subPath == "" {
		subPath = "sub" // default Custom Sub URL Prefix
	}

	panelPath := all["panel_public_path"]
	if panelPath == "" {
		panelPath = "panel" // default publish path under the main domain
	}

	return &models.Settings{
		Domain:          all["domain"],
		Port:            port,
		TLSEmail:        all["tls_email"],
		DecoySite:       all["decoy_site"],
		SubPath:         subPath,
		PanelPublic:     all["panel_public"] == "true",
		PanelPublicPath: panelPath,
		PanelBasicUser:  all["panel_basic_user"],
		PanelBasicHash:  all["panel_basic_hash"],
	}, nil
}

// SaveSettings saves a Settings struct
func (db *DB) SaveSettings(s *models.Settings) error {
	panelPath := s.PanelPublicPath
	if panelPath == "" {
		panelPath = "panel"
	}

	pairs := map[string]string{
		"domain":            s.Domain,
		"port":              fmt.Sprintf("%d", s.Port),
		"tls_email":         s.TLSEmail,
		"decoy_site":        s.DecoySite,
		"sub_path":          s.SubPath,
		"panel_public":      boolToSetting(s.PanelPublic),
		"panel_public_path": panelPath,
		"panel_basic_user":  s.PanelBasicUser,
		"panel_basic_hash":  s.PanelBasicHash,
	}

	for key, value := range pairs {
		if err := db.SetSetting(key, value); err != nil {
			return err
		}
	}
	return nil
}

func boolToSetting(b bool) string {
	if b {
		return "true"
	}
	return ""
}

// --- Subscriptions & HWID ---

// GetUserBySubToken retrieves a user by their sub_token
func (db *DB) GetUserBySubToken(token string) (*models.ProxyUser, error) {
	var u models.ProxyUser
	var enabled int
	err := db.conn.QueryRow(
		"SELECT id, username, password, traffic_up, traffic_down, traffic_limit, hwid_limit, sub_token, expires_at, hwid_reset_interval, last_hwid_reset, enabled, created_at FROM proxy_users WHERE sub_token = ?", token,
	).Scan(&u.ID, &u.Username, &u.Password, &u.TrafficUp, &u.TrafficDown, &u.TrafficLimit, &u.HWIDLimit, &u.SubToken, &u.ExpiresAt, &u.HWIDResetInterval, &u.LastHWIDReset, &enabled, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.Enabled = enabled == 1
	return &u, nil
}

// GetHWIDs retrieves all recorded HWIDs for a user
func (db *DB) GetHWIDs(userID int64) ([]models.UserHWID, error) {
	rows, err := db.conn.Query("SELECT id, user_id, hwid, created_at FROM user_hwids WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hwids []models.UserHWID
	for rows.Next() {
		var h models.UserHWID
		if err := rows.Scan(&h.ID, &h.UserID, &h.HWID, &h.CreatedAt); err != nil {
			return nil, err
		}
		hwids = append(hwids, h)
	}
	return hwids, nil
}

// RegisterHWID records a new HWID for a user
func (db *DB) RegisterHWID(userID int64, hwid string) error {
	_, err := db.conn.Exec("INSERT OR IGNORE INTO user_hwids (user_id, hwid) VALUES (?, ?)", userID, hwid)
	return err
}

// ResetHWIDs clears all HWIDs for a user
func (db *DB) ResetHWIDs(userID int64) error {
	_, err := db.conn.Exec("DELETE FROM user_hwids WHERE user_id = ?", userID)
	return err
}

// UpdateHWIDResetTime updates the last_hwid_reset field for a user to now
func (db *DB) UpdateHWIDResetTime(userID int64) error {
	_, err := db.conn.Exec("UPDATE proxy_users SET last_hwid_reset = ? WHERE id = ?", time.Now(), userID)
	return err
}
