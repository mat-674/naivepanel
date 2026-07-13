package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds the application configuration
type Config struct {
	PanelPort     int    `json:"panel_port"`
	PanelBind     string `json:"panel_bind"`
	JWTSecret     string `json:"jwt_secret"`
	DataDir       string `json:"data_dir"`
	NaiveBinary   string `json:"naive_binary"`
	CaddyfilePath string `json:"caddyfile_path"`
	LogLevel      string `json:"log_level"`
}

// DefaultDataDir returns the default data directory
func DefaultDataDir() string {
	return "/opt/naivepanel"
}

// DefaultPanelBind keeps the administrative panel off the public network.
// Public subscription requests are served by Caddy and proxied back to this
// loopback listener.
func DefaultPanelBind() string {
	return "127.0.0.1"
}

// Load reads configuration from a JSON file, creating defaults if missing
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := generateDefaults()
		if err := cfg.Save(path); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes configuration to a JSON file
func (c *Config) Save(path string) error {
	c.applyDefaults()
	if err := c.Validate(); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// DBPath returns the path to the SQLite database
func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "naivepanel.db")
}

// PanelAddress returns the listener address for the administrative panel.
func (c *Config) PanelAddress() string {
	return net.JoinHostPort(c.PanelBind, strconv.Itoa(c.PanelPort))
}

// PanelUpstream returns the loopback address Caddy uses to reach the panel.
func (c *Config) PanelUpstream() string {
	return c.PanelAddress()
}

// OverrideDataDir updates default dependent paths together with the data
// directory. Explicit custom binary and Caddyfile paths are preserved.
func (c *Config) OverrideDataDir(dataDir string) {
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" || dataDir == c.DataDir {
		return
	}

	oldDataDir := c.DataDir
	if c.NaiveBinary == "" || samePath(c.NaiveBinary, filepath.Join(oldDataDir, "naive")) {
		c.NaiveBinary = filepath.Join(dataDir, "naive")
	}
	if c.CaddyfilePath == "" || samePath(c.CaddyfilePath, filepath.Join(oldDataDir, "Caddyfile")) {
		c.CaddyfilePath = filepath.Join(dataDir, "Caddyfile")
	}
	c.DataDir = dataDir
}

// OverridePanelBind applies a command-line bind-address override.
func (c *Config) OverridePanelBind(bind string) {
	if bind = strings.TrimSpace(bind); bind != "" {
		c.PanelBind = bind
	}
}

// Validate rejects unsafe public bind addresses for the administrative panel.
func (c *Config) Validate() error {
	if c.PanelPort < 1 || c.PanelPort > 65535 {
		return fmt.Errorf("panel port must be between 1 and 65535")
	}

	bind := strings.Trim(strings.TrimSpace(c.PanelBind), "[]")
	if bind == "localhost" {
		return nil
	}

	ip := net.ParseIP(bind)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("panel_bind must be a loopback address (for example 127.0.0.1 or ::1)")
	}

	return nil
}

func (c *Config) applyDefaults() {
	if c.PanelBind == "" {
		c.PanelBind = DefaultPanelBind()
	}
	if c.DataDir == "" {
		c.DataDir = DefaultDataDir()
	}
	if c.NaiveBinary == "" {
		c.NaiveBinary = filepath.Join(c.DataDir, "naive")
	}
	if c.CaddyfilePath == "" {
		c.CaddyfilePath = filepath.Join(c.DataDir, "Caddyfile")
	}
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func generateDefaults() *Config {
	port, _ := randomPort()
	secret := randomHex(32)

	dataDir := DefaultDataDir()

	return &Config{
		PanelPort:     port,
		PanelBind:     DefaultPanelBind(),
		JWTSecret:     secret,
		DataDir:       dataDir,
		NaiveBinary:   filepath.Join(dataDir, "naive"),
		CaddyfilePath: filepath.Join(dataDir, "Caddyfile"),
		LogLevel:      "info",
	}
}

func randomPort() (int, error) {
	// Random port between 10000 and 65000
	n, err := rand.Int(rand.Reader, big.NewInt(55000))
	if err != nil {
		return 39000, err
	}
	return int(n.Int64()) + 10000, nil
}

func randomHex(n int) string {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateRandomPassword creates a random alphanumeric password
func GenerateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// GenerateRandomUsername creates a random username
func GenerateRandomUsername(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
