package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mailstack/mailstack/internal/config"
	"github.com/mailstack/mailstack/internal/osdetect"
	"github.com/mailstack/mailstack/internal/packages"
	"github.com/mailstack/mailstack/internal/system"
	"github.com/mailstack/mailstack/internal/templates"
)

// Installer handles the installation process
type Installer struct {
	config  *config.Config
	verbose bool
	osInfo  *osdetect.OSInfo
	pkgMgr  *packages.Manager
}

// New creates a new installer instance
func New(cfg *config.Config, verbose bool) *Installer {
	return &Installer{
		config:  cfg,
		verbose: verbose,
	}
}

// Install performs the complete installation
func (i *Installer) Install(force bool) error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Detecting OS", i.detectOS},
		{"Checking prerequisites", i.checkPrerequisites},
		{"Installing packages", i.installPackages},
		{"Creating system users", i.createSystemUsers},
		{"Creating directories", i.createDirectories},
		{"Generating configuration files", i.generateConfigs},
		{"Initializing database", i.initDatabase},
		{"Generating DKIM keys", i.generateDKIM},
		{"Setting up TLS certificates", i.setupTLS},
		{"Configuring services", i.configureServices},
		{"Starting services", i.startServices},
		{"Creating admin user", i.createAdminUser},
		{"Running health checks", i.healthCheck},
	}

	for idx, step := range steps {
		if i.verbose {
			fmt.Printf("[%d/%d] %s...\n", idx+1, len(steps), step.name)
		} else {
			fmt.Printf("⏳ %s...\n", step.name)
		}

		if err := step.fn(); err != nil {
			return fmt.Errorf("%s failed: %w", step.name, err)
		}

		if !i.verbose {
			fmt.Printf("✅ %s\n", step.name)
		}
	}

	return nil
}

// Update updates the mail stack components
func (i *Installer) Update() error {
	if err := i.detectOS(); err != nil {
		return err
	}

	fmt.Println("Updating package lists...")
	if err := i.pkgMgr.Update(); err != nil {
		return fmt.Errorf("failed to update package lists: %w", err)
	}

	fmt.Println("Upgrading packages...")
	requiredPkgs := packages.GetRequiredPackages(i.osInfo.Type)
	optionalPkgs := packages.GetOptionalPackages(i.osInfo.Type,
		i.config.Services.Antivirus, i.config.Services.Webmail)

	allPkgs := append(requiredPkgs, optionalPkgs...)

	if err := i.pkgMgr.Install(allPkgs); err != nil {
		return fmt.Errorf("failed to upgrade packages: %w", err)
	}

	return nil
}

func (i *Installer) detectOS() error {
	osInfo, err := osdetect.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect OS: %w", err)
	}

	if !osInfo.IsSupported() {
		return fmt.Errorf("unsupported operating system: %s", osInfo.String())
	}

	i.osInfo = osInfo
	i.pkgMgr = packages.NewManager(osInfo)

	if i.verbose {
		fmt.Printf("Detected: %s\n", osInfo.String())
	}

	return nil
}

func (i *Installer) checkPrerequisites() error {
	// Check if running as root
	if !system.IsRoot() {
		return fmt.Errorf("installation must be run as root")
	}

	// Check if systemd is available
	if !system.ServiceExists("systemd") {
		return fmt.Errorf("systemd is required but not found")
	}

	return nil
}

func (i *Installer) installPackages() error {
	// Update package lists first
	if i.verbose {
		fmt.Println("  Updating package lists...")
	}
	if err := i.pkgMgr.Update(); err != nil {
		return fmt.Errorf("failed to update packages: %w", err)
	}

	// Get required packages
	requiredPkgs := packages.GetRequiredPackages(i.osInfo.Type)

	if i.verbose {
		fmt.Printf("  Installing %d required packages...\n", len(requiredPkgs))
	}

	// Filter out already installed packages
	var toInstall []string
	for _, pkg := range requiredPkgs {
		if !i.pkgMgr.IsInstalled(pkg) {
			toInstall = append(toInstall, pkg)
		} else if i.verbose {
			fmt.Printf("  ✓ %s (already installed)\n", pkg)
		}
	}

	if len(toInstall) > 0 {
		if err := i.pkgMgr.Install(toInstall); err != nil {
			return fmt.Errorf("failed to install packages: %w", err)
		}
	}

	// Install optional packages
	optionalPkgs := packages.GetOptionalPackages(i.osInfo.Type,
		i.config.Services.Antivirus, i.config.Services.Webmail)

	if len(optionalPkgs) > 0 {
		if i.verbose {
			fmt.Printf("  Installing %d optional packages...\n", len(optionalPkgs))
		}

		var toInstallOptional []string
		for _, pkg := range optionalPkgs {
			if !i.pkgMgr.IsInstalled(pkg) {
				toInstallOptional = append(toInstallOptional, pkg)
			}
		}

		if len(toInstallOptional) > 0 {
			if err := i.pkgMgr.Install(toInstallOptional); err != nil {
				fmt.Printf("  Warning: failed to install optional packages: %v\n", err)
			}
		}
	}

	return nil
}

func (i *Installer) createSystemUsers() error {
	users := []struct {
		name  string
		home  string
		shell string
	}{
		{"mailu", "/var/lib/mailstack", "/bin/false"},
		{"postfix", "", "/bin/false"},
		{"dovecot", "", "/bin/false"},
	}

	for _, u := range users {
		if i.verbose {
			fmt.Printf("  Creating user: %s\n", u.name)
		}
		if err := system.CreateUser(u.name, u.home, u.shell); err != nil {
			return err
		}
	}

	// Create mail group if needed
	if err := system.CreateGroup("mail"); err != nil {
		return err
	}

	return nil
}

func (i *Installer) createDirectories() error {
	dirs := []struct {
		path  string
		owner string
		mode  os.FileMode
	}{
		{i.config.Paths.Data, "mailu", 0750},
		{i.config.Paths.Mail, "mailu", 0750},
		{i.config.Paths.DKIM, "mailu", 0700},
		{i.config.Paths.Queue, "postfix", 0750},
		{i.config.Paths.Filter, "mailu", 0750},
		{i.config.Paths.Certs, "mailu", 0750},
		{i.config.Paths.Overrides, "root", 0755},
		{"/etc/mailstack", "root", 0755},
		{"/var/log/mailstack", "mailu", 0750},
	}

	for _, d := range dirs {
		if i.verbose {
			fmt.Printf("  Creating directory: %s\n", d.path)
		}
		if err := system.CreateDirectory(d.path, d.owner, d.mode); err != nil {
			return err
		}
	}

	return nil
}

func (i *Installer) generateConfigs() error {
	renderer := templates.NewRenderer(i.config)

	// Generate Postfix configs
	postfixConfigs := map[string]string{
		"postfix/main.cf":   "/etc/postfix/main.cf",
		"postfix/master.cf": "/etc/postfix/master.cf",
	}

	for template, output := range postfixConfigs {
		if i.verbose {
			fmt.Printf("  Generating %s...\n", output)
		}
		if err := renderer.RenderToFile(template, output); err != nil {
			return fmt.Errorf("failed to render %s: %w", template, err)
		}
	}

	// Create empty database maps that Postfix needs
	emptyMaps := []string{
		filepath.Join(i.config.Paths.Data, "virtual_alias_maps"),
		filepath.Join(i.config.Paths.Data, "virtual_domains"),
		filepath.Join(i.config.Paths.Data, "virtual_mailbox_maps"),
		filepath.Join(i.config.Paths.Data, "sender_canonical_maps"),
		filepath.Join(i.config.Paths.Data, "recipient_canonical_maps"),
		filepath.Join(i.config.Paths.Data, "sender_login_maps"),
		"/etc/postfix/transport.map",
		"/etc/postfix/tls_policy.map",
	}

	for _, mapFile := range emptyMaps {
		// Create empty file if it doesn't exist
		if _, err := os.Stat(mapFile); os.IsNotExist(err) {
			if err := os.WriteFile(mapFile, []byte{}, 0644); err != nil {
				return fmt.Errorf("failed to create map file %s: %w", mapFile, err)
			}
			// Run postmap to create .lmdb files
			cmd := exec.Command("postmap", "lmdb:"+mapFile)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to run postmap on %s: %w", mapFile, err)
			}
		}
	}

	// TODO: Generate Dovecot configs
	// TODO: Generate Rspamd configs
	// TODO: Generate Nginx configs

	return nil
}

func (i *Installer) initDatabase() error {
	// TODO: Initialize database schema
	return nil
}

func (i *Installer) generateDKIM() error {
	// TODO: Generate DKIM keys for main domain
	return nil
}

func (i *Installer) setupTLS() error {
	// TODO: Set up TLS certificates
	return nil
}

func (i *Installer) configureServices() error {
	// TODO: Configure systemd services
	return nil
}

func (i *Installer) startServices() error {
	// TODO: Start all services
	return nil
}

func (i *Installer) createAdminUser() error {
	// TODO: Create initial admin user
	return nil
}

func (i *Installer) healthCheck() error {
	// TODO: Verify all services are running
	return nil
}
