package models

import "time"

// Admin represents the panel administrator
type Admin struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// ProxyUser represents a NaiveProxy client user
type ProxyUser struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Password     string    `json:"password"`
	TrafficUp    int64     `json:"traffic_up"`
	TrafficDown  int64     `json:"traffic_down"`
	TrafficLimit int64     `json:"traffic_limit"` // 0 = unlimited, in bytes
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

// Settings represents server configuration
type Settings struct {
	Domain    string `json:"domain"`
	Port      int    `json:"port"`
	TLSEmail  string `json:"tls_email"`
	DecoySite string `json:"decoy_site"`
}

// ServerStatus represents the current state of NaiveProxy
type ServerStatus struct {
	Running      bool    `json:"running"`
	PID          int     `json:"pid"`
	Uptime       string  `json:"uptime"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemoryMB     float64 `json:"memory_mb"`
	TotalUp      int64   `json:"total_up"`
	TotalDown    int64   `json:"total_down"`
	UserCount    int     `json:"user_count"`
	Version      string  `json:"version"`
	SystemOS     string  `json:"system_os"`
	SystemArch   string  `json:"system_arch"`
	SystemUptime string  `json:"system_uptime"`
}

// LoginRequest is the login API body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is the login API response
type LoginResponse struct {
	Token   string `json:"token"`
	Message string `json:"message"`
}

// CreateUserRequest is the create user API body
type CreateUserRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	TrafficLimit int64  `json:"traffic_limit"`
}

// UpdateUserRequest is the update user API body
type UpdateUserRequest struct {
	Password     *string `json:"password,omitempty"`
	TrafficLimit *int64  `json:"traffic_limit,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
}

// UserLink contains connection info for a proxy user
type UserLink struct {
	URI    string `json:"uri"`
	QRCode string `json:"qr_code"` // base64 PNG
}

// APIResponse is a generic API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
