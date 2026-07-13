package models

import (
	"encoding/json"
	"time"
)

// OptionalInt64 distinguishes a JSON field that is absent from one that is
// explicitly present, including an explicit null. UnmarshalJSON is only
// invoked by encoding/json when the key appears in the body, so Present stays
// false for an omitted key. This is what lets a PATCH-style update tell "clear
// this field" (null) apart from "leave it unchanged" (omitted).
type OptionalInt64 struct {
	Present bool
	Value   *int64
}

// UnmarshalJSON records that the key was present and captures its value (nil
// for an explicit null).
func (o *OptionalInt64) UnmarshalJSON(data []byte) error {
	o.Present = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// Admin represents the panel administrator
type Admin struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// ProxyUser represents a NaiveProxy client user
type ProxyUser struct {
	ID                int64      `json:"id"`
	Username          string     `json:"username"`
	Password          string     `json:"password"`
	TrafficUp         int64      `json:"traffic_up"`
	TrafficDown       int64      `json:"traffic_down"`
	TrafficLimit      int64      `json:"traffic_limit"` // 0 = unlimited, in bytes
	HWIDLimit         int        `json:"hwid_limit"`    // 0 = unlimited devices
	SubToken          string     `json:"sub_token"`
	ExpiresAt         *time.Time `json:"expires_at"`
	HWIDResetInterval int        `json:"hwid_reset_interval"` // in days
	LastHWIDReset     *time.Time `json:"last_hwid_reset"`
	Enabled           bool       `json:"enabled"`
	CreatedAt         time.Time  `json:"created_at"`
}

// UserHWID represents a recorded hardware ID for a user
type UserHWID struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	HWID      string    `json:"hwid"`
	CreatedAt time.Time `json:"created_at"`
}

// SubscriptionData is the response format for the NaiveUI client
type SubscriptionData struct {
	Version  int                   `json:"version"`
	Info     *SubscriptionInfo     `json:"info,omitempty"`
	Profiles []SubscriptionProfile `json:"profiles"`
}

// SubscriptionInfo contains user limits for NaiveUI
type SubscriptionInfo struct {
	UserTag           string `json:"user_tag"`
	ExpiresAt         int64  `json:"expires_at"`
	TrafficLimitBytes int64  `json:"traffic_limit_bytes"`
	TrafficUsedBytes  int64  `json:"traffic_used_bytes"`
	Message           string `json:"message"`
}

// SubscriptionProfile contains server connection details
type SubscriptionProfile struct {
	Name           string `json:"name"`
	Server         string `json:"server"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Protocol       string `json:"protocol"`
	ListenProtocol string `json:"listen_protocol,omitempty"`
	ListenPort     int    `json:"listen_port,omitempty"`
	Concurrency    int    `json:"concurrency,omitempty"`
	ExtraHeaders   string `json:"extra_headers,omitempty"`
}

// Settings represents server configuration
type Settings struct {
	Domain    string `json:"domain"`
	Port      int    `json:"port"`
	TLSEmail  string `json:"tls_email"`
	DecoySite string `json:"decoy_site"`
	SubPath   string `json:"sub_path"`

	// PanelPublic publishes the loopback admin panel at /<PanelPublicPath>/*
	// through Caddy with HTTP Basic Auth in front. Off by default; the panel
	// stays on 127.0.0.1 and is reached via SSH tunneling.
	PanelPublic     bool   `json:"panel_public"`
	PanelPublicPath string `json:"panel_public_path,omitempty"`
	PanelBasicUser  string `json:"panel_basic_user,omitempty"`
	// PanelBasicHash is a bcrypt hash of the Basic Auth password, written
	// verbatim into Caddy's basicauth directive. Plaintext is never stored.
	PanelBasicHash string `json:"panel_basic_hash,omitempty"`
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
	Username          string `json:"username"`
	Password          string `json:"password"`
	TrafficLimit      int64  `json:"traffic_limit"`
	HWIDLimit         int    `json:"hwid_limit"`
	ExpiresAt         *int64 `json:"expires_at"`          // unix timestamp
	HWIDResetInterval int    `json:"hwid_reset_interval"` // in days
}

// UpdateUserRequest is the update user API body
type UpdateUserRequest struct {
	Password          *string       `json:"password,omitempty"`
	TrafficLimit      *int64        `json:"traffic_limit,omitempty"`
	HWIDLimit         *int          `json:"hwid_limit,omitempty"`
	ExpiresAt         OptionalInt64 `json:"expires_at"`
	HWIDResetInterval *int          `json:"hwid_reset_interval,omitempty"`
	Enabled           *bool         `json:"enabled,omitempty"`
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
