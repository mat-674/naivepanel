package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	PanelPort     int    `json:"panel_port"`
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

	return &cfg, nil
}

// Save writes configuration to a JSON file
func (c *Config) Save(path string) error {
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

func generateDefaults() *Config {
	port, _ := randomPort()
	secret := randomHex(32)

	dataDir := DefaultDataDir()

	return &Config{
		PanelPort:     port,
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
