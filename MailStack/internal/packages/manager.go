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
	switch osType {
	case osdetect.Debian, osdetect.Ubuntu:
		return []string{
			// Mail Transfer Agent (MTA)
			"postfix",
			"postfix-pcre",

			// IMAP/POP3 Server
			"dovecot-core",
			"dovecot-imapd",
			"dovecot-pop3d",
			"dovecot-lmtpd",
			"dovecot-managesieved",
			"dovecot-sieve",
			"dovecot-mysql",
			"dovecot-pgsql",
			"dovecot-sqlite",

			// Anti-spam and filtering
			"rspamd",

			// Database and caching
			"redis-server",

			// Web server
			"nginx",

			// TLS/SSL
			"ca-certificates",
			"certbot",
			"openssl",

			// SASL authentication
			"libsasl2-modules",
			"sasl2-bin",

			// Python runtime
			"python3",
			"python3-pip",
			"python3-setuptools",

			// Database clients
			"sqlite3",
			"postgresql-client",
			"default-mysql-client",

			// Let's Encrypt
			"python3-certbot-nginx",

			// System utilities
			"curl",
			"gnupg",
			"supervisor",
			"logrotate",
			"net-tools",
			"dnsutils",
			"iputils-ping",
		}

	case osdetect.RHEL, osdetect.CentOS, osdetect.Fedora:
		return []string{
			// Mail Transfer Agent (MTA)
			"postfix",
			"postfix-pcre",

			// IMAP/POP3 Server
			"dovecot",
			"dovecot-pigeonhole",
			"dovecot-mysql",
			"dovecot-pgsql",

			// Anti-spam and filtering
			"rspamd",

			// Database and caching
			"redis",

			// Web server
			"nginx",

			// TLS/SSL
			"ca-certificates",
			"certbot",
			"openssl",

			// SASL authentication
			"cyrus-sasl",
			"cyrus-sasl-plain",
			"cyrus-sasl-md5",

			// Python runtime
			"python3",
			"python3-pip",

			// Database clients
			"sqlite",
			"postgresql",
			"mysql",

			// Let's Encrypt
			"python3-certbot-nginx",

			// System utilities
			"curl",
			"gnupg",
			"supervisor",
			"logrotate",
			"net-tools",
			"bind-utils",
			"iputils",
		}

	case osdetect.Alpine:
		return []string{
			// Mail Transfer Agent (MTA)
			"postfix",
			"postfix-pcre",

			// IMAP/POP3 Server
			"dovecot",
			"dovecot-lmtpd",
			"dovecot-pigeonhole-plugin",
			"dovecot-pop3d",
			"dovecot-mysql",
			"dovecot-pgsql",
			"dovecot-sqlite",

			// Anti-spam and filtering
			"rspamd",
			"rspamd-controller",
			"rspamd-fuzzy",
			"rspamd-proxy",

			// Database and caching
			"redis",

			// Web server
			"nginx",

			// TLS/SSL
			"ca-certificates",
			"certbot",
			"openssl",

			// SASL authentication
			"cyrus-sasl",

			// Python runtime
			"python3",
			"py3-pip",

			// Database clients
			"sqlite",
			"postgresql-client",
			"mysql-client",

			// System utilities
			"curl",
			"gnupg",
			"supervisor",
			"logrotate",
			"net-tools",
			"bind-tools",
		}

	default:
		return []string{}
	}
}

// GetOptionalPackages returns optional packages based on configuration
func GetOptionalPackages(osType osdetect.OSType, enableAntivirus, enableWebmail bool) []string {
	var packages []string

	// Antivirus (ClamAV)
	if enableAntivirus {
		switch osType {
		case osdetect.Debian, osdetect.Ubuntu:
			packages = append(packages,
				"clamav",
				"clamav-daemon",
				"clamav-freshclam",
			)
		case osdetect.RHEL, osdetect.CentOS, osdetect.Fedora:
			packages = append(packages,
				"clamav",
				"clamav-update",
				"clamd",
			)
		case osdetect.Alpine:
			packages = append(packages,
				"clamav",
				"clamav-daemon",
			)
		}
	}

	// Webmail and PHP dependencies
	if enableWebmail {
		switch osType {
		case osdetect.Debian, osdetect.Ubuntu:
			packages = append(packages,
				// PHP 8.1+ with all extensions
				"php8.1-fpm",
				"php8.1-cli",
				"php8.1-common",
				"php8.1-json",
				"php8.1-mysql",
				"php8.1-pgsql",
				"php8.1-sqlite3",
				"php8.1-curl",
				"php8.1-mbstring",
				"php8.1-xml",
				"php8.1-intl",
				"php8.1-zip",
				"php8.1-gd",
				"php8.1-imap",
				"php8.1-ldap",
				"php8.1-bcmath",
				"php8.1-opcache",

				// Snuffleupagus security module
				"php8.1-dev",
				"gcc",
				"make",

				// Composer (PHP package manager)
				"composer",
			)

		case osdetect.RHEL, osdetect.CentOS, osdetect.Fedora:
			packages = append(packages,
				// PHP with all extensions
				"php-fpm",
				"php-cli",
				"php-common",
				"php-json",
				"php-mysqlnd",
				"php-pgsql",
				"php-pdo",
				"php-mbstring",
				"php-xml",
				"php-intl",
				"php-zip",
				"php-gd",
				"php-imap",
				"php-ldap",
				"php-bcmath",
				"php-opcache",

				// Development tools
				"php-devel",
				"gcc",
				"make",

				// Composer
				"composer",
			)

		case osdetect.Alpine:
			packages = append(packages,
				// PHP 8.3 (Alpine's latest)
				"php83-fpm",
				"php83-common",
				"php83-json",
				"php83-pdo",
				"php83-pdo_mysql",
				"php83-pdo_pgsql",
				"php83-pdo_sqlite",
				"php83-curl",
				"php83-mbstring",
				"php83-xml",
				"php83-intl",
				"php83-zip",
				"php83-gd",
				"php83-imap",
				"php83-ldap",
				"php83-bcmath",
				"php83-opcache",

				// Development tools
				"php83-dev",
				"gcc",
				"make",

				// Composer
				"composer",
			)
		}
	}

	return packages
}
