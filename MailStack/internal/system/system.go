package system

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
)

// CreateUser creates a system user
func CreateUser(username string, home string, shell string) error {
	// Check if user already exists
	if _, err := user.Lookup(username); err == nil {
		return nil // User already exists
	}

	args := []string{
		"--system",
		"--no-create-home",
		"--shell", shell,
	}

	if home != "" {
		args = append(args, "--home", home)
	}

	args = append(args, username)

	cmd := exec.Command("useradd", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create user %s: %w", username, err)
	}

	return nil
}

// CreateGroup creates a system group
func CreateGroup(groupname string) error {
	cmd := exec.Command("groupadd", "--system", groupname)
	if err := cmd.Run(); err != nil {
		// Ignore error if group already exists
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 9 { // Group already exists
				return nil
			}
		}
		return fmt.Errorf("failed to create group %s: %w", groupname, err)
	}
	return nil
}

// AddUserToGroup adds a user to a group
func AddUserToGroup(username, groupname string) error {
	cmd := exec.Command("usermod", "-a", "-G", groupname, username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add user %s to group %s: %w", username, groupname, err)
	}
	return nil
}

// CreateDirectory creates a directory with specific owner and permissions
func CreateDirectory(path string, owner string, mode os.FileMode) error {
	// Create directory
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	// Set ownership if owner specified
	if owner != "" {
		if err := Chown(path, owner); err != nil {
			return err
		}
	}

	// Set permissions
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", path, err)
	}

	return nil
}

// Chown changes ownership of a file or directory
func Chown(path string, owner string) error {
	u, err := user.Lookup(owner)
	if err != nil {
		return fmt.Errorf("failed to lookup user %s: %w", owner, err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("invalid uid: %w", err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return fmt.Errorf("invalid gid: %w", err)
	}

	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("failed to chown %s: %w", path, err)
	}

	return nil
}

// ChownRecursive changes ownership recursively
func ChownRecursive(path string, owner string) error {
	u, err := user.Lookup(owner)
	if err != nil {
		return fmt.Errorf("failed to lookup user %s: %w", owner, err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("invalid uid: %w", err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return fmt.Errorf("invalid gid: %w", err)
	}

	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			err = os.Chown(name, uid, gid)
		}
		return err
	})
}

// IsRoot checks if the current process is running as root
func IsRoot() bool {
	return os.Geteuid() == 0
}

// WriteFile writes content to a file with specific permissions
func WriteFile(path string, content []byte, mode os.FileMode) error {
	if err := os.WriteFile(path, content, mode); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}

// ServiceExists checks if a systemd service exists
func ServiceExists(name string) bool {
	// Special case: check if systemd itself is available
	if name == "systemd" {
		return IsSystemdAvailable()
	}
	cmd := exec.Command("systemctl", "list-unit-files", name+".service")
	return cmd.Run() == nil
}

// IsSystemdAvailable checks if systemd is available on the system
func IsSystemdAvailable() bool {
	// Check if systemctl command exists
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	// Check if systemd is running as PID 1
	cmd := exec.Command("systemctl", "--version")
	return cmd.Run() == nil
}

// EnableService enables a systemd service
func EnableService(name string) error {
	cmd := exec.Command("systemctl", "enable", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service %s: %w", name, err)
	}
	return nil
}

// StartService starts a systemd service
func StartService(name string) error {
	cmd := exec.Command("systemctl", "start", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service %s: %w", name, err)
	}
	return nil
}

// StopService stops a systemd service
func StopService(name string) error {
	cmd := exec.Command("systemctl", "stop", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service %s: %w", name, err)
	}
	return nil
}

// RestartService restarts a systemd service
func RestartService(name string) error {
	cmd := exec.Command("systemctl", "restart", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart service %s: %w", name, err)
	}
	return nil
}

// ReloadService reloads a systemd service
func ReloadService(name string) error {
	cmd := exec.Command("systemctl", "reload", name)
	if err := cmd.Run(); err != nil {
		// If reload is not supported, try restart
		return RestartService(name)
	}
	return nil
}

// IsServiceRunning checks if a service is running
func IsServiceRunning(name string) bool {
	cmd := exec.Command("systemctl", "is-active", name)
	return cmd.Run() == nil
}
