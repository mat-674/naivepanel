package naiveproxy

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Manager handles the NaiveProxy (Caddy) process lifecycle
type Manager struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	binaryPath string
	caddyfile  string
	startTime  time.Time
	running    bool
}

// NewManager creates a new NaiveProxy process manager
func NewManager(binaryPath, caddyfilePath string) *Manager {
	return &Manager{
		binaryPath: binaryPath,
		caddyfile:  caddyfilePath,
	}
}

// Start starts the NaiveProxy process
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running && m.cmd != nil && m.cmd.Process != nil {
		return fmt.Errorf("naiveproxy is already running")
	}

	if _, err := os.Stat(m.binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("naive binary not found at %s", m.binaryPath)
	}

	if _, err := os.Stat(m.caddyfile); os.IsNotExist(err) {
		return fmt.Errorf("caddyfile not found at %s", m.caddyfile)
	}

	m.cmd = exec.Command(m.binaryPath, "run", "--config", m.caddyfile, "--adapter", "caddyfile")
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start naiveproxy: %w", err)
	}

	m.running = true
	m.startTime = time.Now()

	// Monitor the process in background
	go func() {
		err := m.cmd.Wait()
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
		if err != nil {
			log.Printf("NaiveProxy process exited: %v", err)
		}
	}()

	log.Printf("NaiveProxy started with PID %d", m.cmd.Process.Pid)
	return nil
}

// Stop stops the NaiveProxy process
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.cmd == nil || m.cmd.Process == nil {
		return fmt.Errorf("naiveproxy is not running")
	}

	if err := m.cmd.Process.Signal(os.Interrupt); err != nil {
		// Force kill if interrupt fails
		if err := m.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill naiveproxy: %w", err)
		}
	}

	m.running = false
	log.Println("NaiveProxy stopped")
	return nil
}

// Restart restarts the NaiveProxy process
func (m *Manager) Restart() error {
	if m.IsRunning() {
		if err := m.Stop(); err != nil {
			log.Printf("Warning: stop failed during restart: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return m.Start()
}

// Reload reloads the Caddyfile without restarting
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.cmd == nil || m.cmd.Process == nil {
		return fmt.Errorf("naiveproxy is not running, cannot reload")
	}

	reloadCmd := exec.Command(m.binaryPath, "reload", "--config", m.caddyfile, "--adapter", "caddyfile")
	output, err := reloadCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload: %w, output: %s", err, string(output))
	}

	log.Println("NaiveProxy configuration reloaded")
	return nil
}

// IsRunning returns whether NaiveProxy is currently running
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// PID returns the process ID, or 0 if not running
func (m *Manager) PID() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running && m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

// Uptime returns the duration since NaiveProxy was started
func (m *Manager) Uptime() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return 0
	}
	return time.Since(m.startTime)
}

// FormatUptime returns a human-readable uptime string
func (m *Manager) FormatUptime() string {
	d := m.Uptime()
	if d == 0 {
		return "stopped"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// WriteCaddyfile writes the Caddyfile content to disk
func (m *Manager) WriteCaddyfile(content string) error {
	return os.WriteFile(m.caddyfile, []byte(content), 0644)
}

// GetSystemInfo returns basic system information
func GetSystemInfo() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

// GetSystemUptime returns the system uptime on Linux
func GetSystemUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}

	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return "unknown"
	}

	seconds, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return "unknown"
	}

	days := int(seconds) / 86400
	hours := (int(seconds) % 86400) / 3600
	minutes := (int(seconds) % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
