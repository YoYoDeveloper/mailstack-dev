package packages

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mailstack/mailstack/internal/osdetect"
)

// Manager handles package installation for different OS types
type Manager struct {
	osInfo *osdetect.OSInfo
}

// NewManager creates a new package manager
func NewManager(osInfo *osdetect.OSInfo) *Manager {
	return &Manager{osInfo: osInfo}
}

// Install installs a list of packages
func (m *Manager) Install(packages []string) error {
	switch m.osInfo.Type {
	case osdetect.Debian, osdetect.Ubuntu:
		return m.installApt(packages)
	case osdetect.RHEL, osdetect.CentOS, osdetect.Fedora:
		return m.installYum(packages)
	case osdetect.Alpine:
		return m.installApk(packages)
	default:
		return fmt.Errorf("unsupported OS type: %s", m.osInfo.Type)
	}
}

// Update updates package lists
func (m *Manager) Update() error {
	switch m.osInfo.Type {
	case osdetect.Debian, osdetect.Ubuntu:
		return m.runCommand("apt-get", "update", "-y")
	case osdetect.RHEL, osdetect.CentOS, osdetect.Fedora:
		return m.runCommand("yum", "check-update")
	case osdetect.Alpine:
		return m.runCommand("apk", "update")
	default:
		return fmt.Errorf("unsupported OS type: %s", m.osInfo.Type)
	}
}

// IsInstalled checks if a package is installed
func (m *Manager) IsInstalled(pkg string) bool {
	switch m.osInfo.Type {
	case osdetect.Debian, osdetect.Ubuntu:
		cmd := exec.Command("dpkg", "-l", pkg)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), "ii  "+pkg)
	case osdetect.RHEL, osdetect.CentOS, osdetect.Fedora:
		cmd := exec.Command("rpm", "-q", pkg)
		return cmd.Run() == nil
	case osdetect.Alpine:
		cmd := exec.Command("apk", "info", "-e", pkg)
		return cmd.Run() == nil
	default:
		return false
	}
}

// installApt installs packages using apt-get
func (m *Manager) installApt(packages []string) error {
	args := append([]string{"install", "-y"}, packages...)
	return m.runCommand("apt-get", args...)
}

// installYum installs packages using yum/dnf
func (m *Manager) installYum(packages []string) error {
	// Try dnf first (newer systems), fall back to yum
	cmd := "dnf"
	if !commandExists(cmd) {
		cmd = "yum"
	}
	args := append([]string{"install", "-y"}, packages...)
	return m.runCommand(cmd, args...)
}

// installApk installs packages using apk
func (m *Manager) installApk(packages []string) error {
	args := append([]string{"add", "--no-cache"}, packages...)
	return m.runCommand("apk", args...)
}

// runCommand executes a command and returns any error
func (m *Manager) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s\nOutput: %s", err, string(output))
	}
	return nil
}

// commandExists checks if a command exists in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// GetRequiredPackages returns the list of required packages for MailStack
func GetRequiredPackages(osType osdetect.OSType) []string {
	basePackages := []string{
		"postfix",
		"dovecot-core",
		"dovecot-imapd",
		"dovecot-pop3d",
		"dovecot-lmtpd",
		"dovecot-managesieved",
		"rspamd",
		"redis-server",
		"nginx",
		"ca-certificates",
		"curl",
		"gnupg",
		"openssl",
	}

	switch osType {
	case osdetect.Debian, osdetect.Ubuntu:
		return append(basePackages, []string{
			"dovecot-sieve",
			"python3",
			"python3-pip",
			"libsasl2-modules",
			"sasl2-bin",
		}...)
	case osdetect.RHEL, osdetect.CentOS, osdetect.Fedora:
		return append(basePackages, []string{
			"dovecot-pigeonhole",
			"python3",
			"python3-pip",
			"cyrus-sasl",
			"cyrus-sasl-plain",
		}...)
	case osdetect.Alpine:
		return append(basePackages, []string{
			"dovecot-pigeonhole-plugin",
			"python3",
			"py3-pip",
			"cyrus-sasl",
		}...)
	default:
		return basePackages
	}
}

// GetOptionalPackages returns optional packages based on configuration
func GetOptionalPackages(osType osdetect.OSType, enableAntivirus, enableWebmail bool) []string {
	var packages []string

	if enableAntivirus {
		switch osType {
		case osdetect.Debian, osdetect.Ubuntu:
			packages = append(packages, "clamav", "clamav-daemon")
		case osdetect.RHEL, osdetect.CentOS, osdetect.Fedora:
			packages = append(packages, "clamav", "clamav-update")
		case osdetect.Alpine:
			packages = append(packages, "clamav")
		}
	}

	if enableWebmail {
		switch osType {
		case osdetect.Debian, osdetect.Ubuntu:
			packages = append(packages, "php-fpm", "php-cli", "php-json", "php-mysql", 
				"php-pgsql", "php-sqlite3", "php-curl", "php-mbstring", "php-xml")
		case osdetect.RHEL, osdetect.CentOS, osdetect.Fedora:
			packages = append(packages, "php-fpm", "php-cli", "php-json", "php-mysqlnd",
				"php-pgsql", "php-pdo", "php-mbstring", "php-xml")
		case osdetect.Alpine:
			packages = append(packages, "php83-fpm", "php83-json", "php83-pdo",
				"php83-pdo_mysql", "php83-pdo_pgsql", "php83-mbstring", "php83-xml")
		}
	}

	return packages
}
