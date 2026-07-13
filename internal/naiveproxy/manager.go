package naiveproxy

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Manager handles the NaiveProxy (Caddy) process lifecycle
type Manager struct {
	mu          sync.Mutex
	cmd         *exec.Cmd
	binaryPath  string
	caddyfile   string
	serviceName string
	useSystemd  bool
	startTime   time.Time
	running     bool
}

const defaultServiceName = "naiveproxy"

// NewManager creates a new NaiveProxy process manager
func NewManager(binaryPath, caddyfilePath string) *Manager {
	manager := &Manager{
		binaryPath: binaryPath,
		caddyfile:  caddyfilePath,
	}

	if systemdServiceAvailable(defaultServiceName) {
		manager.serviceName = defaultServiceName
		manager.useSystemd = true
	}

	return manager
}

// Start starts the NaiveProxy process
func (m *Manager) Start() error {
	if m.useSystemd {
		return m.systemctl("start")
	}

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
	if m.useSystemd {
		return m.systemctl("stop")
	}

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
	if m.useSystemd {
		return m.systemctl("restart")
	}

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
	if m.useSystemd {
		return m.systemctl("reload")
	}

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
	if m.useSystemd {
		return exec.Command("systemctl", "is-active", "--quiet", m.serviceName).Run() == nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// PID returns the process ID, or 0 if not running
func (m *Manager) PID() int {
	if m.useSystemd {
		output, err := exec.Command("systemctl", "show", m.serviceName, "--property=MainPID", "--value").Output()
		if err != nil {
			return 0
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			return 0
		}
		return pid
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running && m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

// Uptime returns the duration since NaiveProxy was started
func (m *Manager) Uptime() time.Duration {
	if m.useSystemd {
		return m.systemdUptime()
	}

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
	dir := filepath.Dir(m.caddyfile)
	tmp, err := os.CreateTemp(dir, ".Caddyfile-*")
	if err != nil {
		return fmt.Errorf("create temporary Caddyfile: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return fmt.Errorf("set temporary Caddyfile permissions: %w", err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary Caddyfile: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary Caddyfile: %w", err)
	}
	if err := os.Rename(tmpPath, m.caddyfile); err != nil {
		if runtime.GOOS == "windows" {
			return os.WriteFile(m.caddyfile, []byte(content), 0644)
		}
		return fmt.Errorf("replace Caddyfile: %w", err)
	}

	return nil
}

// UpdatePanel triggers the installation script to update the panel
func (m *Manager) UpdatePanel() error {
	if !m.useSystemd {
		return fmt.Errorf("in-panel update requires the dedicated systemd services; run the installer with --update manually")
	}

	// Execute the update script in the background
	scriptPath := "/opt/naivepanel/scripts/install.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("update script not found at %s", scriptPath)
	}

	// The updater must not belong to the panel service cgroup: the update
	// script intentionally stops that service before replacing its binary.
	cmd := exec.Command(
		"systemd-run",
		"--unit=naivepanel-update",
		"--collect",
		"--service-type=exec",
		"bash", scriptPath, "--update",
	)

	// Start the command asynchronously
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start update script: %w", err)
	}

	// We don't wait for completion because the script will restart this process.
	// But we try to release resources.
	go func() {
		cmd.Wait()
	}()

	return nil
}

func (m *Manager) systemctl(action string) error {
	output, err := exec.Command("systemctl", action, m.serviceName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to %s %s: %w, output: %s", action, m.serviceName, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *Manager) systemdUptime() time.Duration {
	if !m.IsRunning() {
		return 0
	}

	output, err := exec.Command("systemctl", "show", m.serviceName, "--property=ActiveEnterTimestampMonotonic", "--value").Output()
	if err != nil {
		return 0
	}
	activeMicros, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil || activeMicros <= 0 {
		return 0
	}

	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return 0
	}
	secondsSinceBoot, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	elapsedSeconds := secondsSinceBoot - float64(activeMicros)/1_000_000
	if elapsedSeconds <= 0 {
		return 0
	}
	return time.Duration(elapsedSeconds * float64(time.Second))
}

func systemdServiceAvailable(serviceName string) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return false
	}

	output, err := exec.Command("systemctl", "show", serviceName, "--property=LoadState", "--value").Output()
	return err == nil && strings.TrimSpace(string(output)) != "" && strings.TrimSpace(string(output)) != "not-found"
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
