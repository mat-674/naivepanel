package database

import (
	"database/sql"
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
		"SELECT id, username, password, traffic_up, traffic_down, traffic_limit, enabled, created_at FROM proxy_users ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.ProxyUser
	for rows.Next() {
		var u models.ProxyUser
		var enabled int
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.TrafficUp, &u.TrafficDown, &u.TrafficLimit, &enabled, &u.CreatedAt); err != nil {
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
		"SELECT id, username, password, traffic_up, traffic_down, traffic_limit, enabled, created_at FROM proxy_users WHERE id = ?", id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.TrafficUp, &u.TrafficDown, &u.TrafficLimit, &enabled, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.Enabled = enabled == 1
	return &u, nil
}

// CreateUser creates a new proxy user
func (db *DB) CreateUser(username, password string, trafficLimit int64) (*models.ProxyUser, error) {
	result, err := db.conn.Exec(
		"INSERT INTO proxy_users (username, password, traffic_limit) VALUES (?, ?, ?)",
		username, password, trafficLimit,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &models.ProxyUser{
		ID:           id,
		Username:     username,
		Password:     password,
		TrafficLimit: trafficLimit,
		Enabled:      true,
		CreatedAt:    time.Now(),
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
		"SELECT id, username, password, traffic_up, traffic_down, traffic_limit, enabled, created_at FROM proxy_users WHERE enabled = 1 ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.ProxyUser
	for rows.Next() {
		var u models.ProxyUser
		var enabled int
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.TrafficUp, &u.TrafficDown, &u.TrafficLimit, &enabled, &u.CreatedAt); err != nil {
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

	return &models.Settings{
		Domain:    all["domain"],
		Port:      port,
		TLSEmail:  all["tls_email"],
		DecoySite: all["decoy_site"],
	}, nil
}

// SaveSettings saves a Settings struct
func (db *DB) SaveSettings(s *models.Settings) error {
	pairs := map[string]string{
		"domain":     s.Domain,
		"port":       fmt.Sprintf("%d", s.Port),
		"tls_email":  s.TLSEmail,
		"decoy_site": s.DecoySite,
	}

	for key, value := range pairs {
		if err := db.SetSetting(key, value); err != nil {
			return err
		}
	}
	return nil
}
