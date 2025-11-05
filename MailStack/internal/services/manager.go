package services

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mailstack/mailstack/internal/config"
)

// Manager manages system services
type Manager struct {
	config *config.Config
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name    string
	Running bool
	Healthy bool
	Status  string
}

// NewManager creates a new service manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{config: cfg}
}

// GetStatus returns the status of all services
func (m *Manager) GetStatus() ([]ServiceStatus, error) {
	services := []string{
		"postfix",
		"dovecot",
		"rspamd",
		"nginx",
		"redis",
	}

	// Add optional services
	if m.config.Webmail != "" && m.config.Webmail != "none" {
		services = append(services, "php8.1-fpm")
	}
	if m.config.Services.Antivirus {
		services = append(services, "clamav-daemon")
	}

	var status []ServiceStatus
	for _, svc := range services {
		stat := m.checkService(svc)
		status = append(status, stat)
	}

	return status, nil
}

// checkService checks if a single service is running
func (m *Manager) checkService(name string) ServiceStatus {
	// Use systemctl to check service status
	cmd := exec.Command("systemctl", "is-active", name)
	output, err := cmd.Output()

	running := err == nil && string(output) == "active\n"

	// Get more detailed status
	var statusText string
	if running {
		statusText = "active"
	} else {
		// Try to get why it's not running
		cmd = exec.Command("systemctl", "is-failed", name)
		failOutput, _ := cmd.Output()
		statusText = strings.TrimSpace(string(failOutput))
		if statusText == "" {
			statusText = "inactive"
		}
	}

	return ServiceStatus{
		Name:    name,
		Running: running,
		Healthy: running, // For now, running = healthy
		Status:  statusText,
	}
}

// Start starts a service
func (m *Manager) Start(name string) error {
	cmd := exec.Command("systemctl", "start", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start %s: %w\nOutput: %s", name, err, output)
	}
	return nil
}

// Stop stops a service
func (m *Manager) Stop(name string) error {
	cmd := exec.Command("systemctl", "stop", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop %s: %w\nOutput: %s", name, err, output)
	}
	return nil
}

// Restart restarts a service
func (m *Manager) Restart(name string) error {
	cmd := exec.Command("systemctl", "restart", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart %s: %w\nOutput: %s", name, err, output)
	}
	return nil
}

// Reload reloads a service
func (m *Manager) Reload(name string) error {
	cmd := exec.Command("systemctl", "reload", name)
	if err := cmd.Run(); err != nil {
		// If reload fails, try restart
		return m.Restart(name)
	}
	return nil
}
