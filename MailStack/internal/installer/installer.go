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
	if i.verbose {
		fmt.Println("  Generating Postfix configuration...")
	}
	postfixConfigs := map[string]string{
		"templates/postfix/main.cf":                     "/etc/postfix/main.cf",
		"templates/postfix/master.cf":                   "/etc/postfix/master.cf",
		"templates/postfix/sasl_passwd":                 "/etc/postfix/sasl_passwd",
		"templates/postfix/outclean_header_filter.cf":   "/etc/postfix/outclean_header_filter.cf",
		"templates/postfix/mta-sts-daemon.yml":          "/etc/mta-sts-daemon.yml",
		"templates/postfix/logrotate.conf":              "/etc/logrotate.d/postfix",
	}

	for template, output := range postfixConfigs {
		if i.verbose {
			fmt.Printf("    %s\n", output)
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
		if _, err := os.Stat(mapFile); os.IsNotExist(err) {
			if err := os.WriteFile(mapFile, []byte{}, 0644); err != nil {
				return fmt.Errorf("failed to create map file %s: %w", mapFile, err)
			}
			cmd := exec.Command("postmap", "lmdb:"+mapFile)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to run postmap on %s: %w", mapFile, err)
			}
		}
	}

	// Generate Dovecot configs
	if i.verbose {
		fmt.Println("  Generating Dovecot configuration...")
	}
	dovecotConfigs := map[string]string{
		"templates/dovecot/dovecot.conf":        "/etc/dovecot/dovecot.conf",
		"templates/dovecot/auth.conf":           "/etc/dovecot/conf.d/auth.conf",
		"templates/dovecot/report-spam.sieve":   "/etc/dovecot/report-spam.sieve",
		"templates/dovecot/report-ham.sieve":    "/etc/dovecot/report-ham.sieve",
		"templates/dovecot/spam.script":         "/etc/dovecot/spam.script",
		"templates/dovecot/ham.script":          "/etc/dovecot/ham.script",
	}

	for template, output := range dovecotConfigs {
		if i.verbose {
			fmt.Printf("    %s\n", output)
		}
		if err := renderer.RenderToFile(template, output); err != nil {
			return fmt.Errorf("failed to render %s: %w", template, err)
		}
	}

	// Generate Rspamd configs
	if i.verbose {
		fmt.Println("  Generating Rspamd configuration...")
	}
	rspamdConfigs := map[string]string{
		"templates/rspamd/antivirus.conf":                  "/etc/rspamd/local.d/antivirus.conf",
		"templates/rspamd/arc.conf":                        "/etc/rspamd/local.d/arc.conf",
		"templates/rspamd/classifier-bayes.conf":           "/etc/rspamd/local.d/classifier-bayes.conf",
		"templates/rspamd/composites.conf":                 "/etc/rspamd/local.d/composites.conf",
		"templates/rspamd/dkim_signing.conf":               "/etc/rspamd/local.d/dkim_signing.conf",
		"templates/rspamd/external_services.conf":          "/etc/rspamd/local.d/external_services.conf",
		"templates/rspamd/external_services_group.conf":    "/etc/rspamd/local.d/external_services_group.conf",
		"templates/rspamd/forbidden_file_extension.map":    "/etc/rspamd/local.d/forbidden_file_extension.map",
		"templates/rspamd/force_actions.conf":              "/etc/rspamd/local.d/force_actions.conf",
		"templates/rspamd/fuzzy_check.conf":                "/etc/rspamd/local.d/fuzzy_check.conf",
		"templates/rspamd/headers_group.conf":              "/etc/rspamd/local.d/headers_group.conf",
		"templates/rspamd/history_redis.conf":              "/etc/rspamd/local.d/history_redis.conf",
		"templates/rspamd/local_subnet.map":                "/etc/rspamd/local.d/local_subnet.map",
		"templates/rspamd/metrics.conf":                    "/etc/rspamd/local.d/metrics.conf",
		"templates/rspamd/milter_headers.conf":             "/etc/rspamd/local.d/milter_headers.conf",
		"templates/rspamd/multimap.conf":                   "/etc/rspamd/local.d/multimap.conf",
		"templates/rspamd/redis.conf":                      "/etc/rspamd/local.d/redis.conf",
		"templates/rspamd/whitelist.conf":                  "/etc/rspamd/local.d/whitelist.conf",
		"templates/rspamd/options.inc":                     "/etc/rspamd/local.d/options.inc",
		"templates/rspamd/logging.inc":                     "/etc/rspamd/local.d/logging.inc",
		"templates/rspamd/worker-controller.inc":           "/etc/rspamd/local.d/worker-controller.inc",
		"templates/rspamd/worker-fuzzy.inc":                "/etc/rspamd/local.d/worker-fuzzy.inc",
		"templates/rspamd/worker-normal.inc":               "/etc/rspamd/local.d/worker-normal.inc",
		"templates/rspamd/worker-proxy.inc":                "/etc/rspamd/local.d/worker-proxy.inc",
	}

	for template, output := range rspamdConfigs {
		if i.verbose {
			fmt.Printf("    %s\n", output)
		}
		if err := renderer.RenderToFile(template, output); err != nil {
			return fmt.Errorf("failed to render %s: %w", template, err)
		}
	}

	// Generate Nginx configs
	if i.verbose {
		fmt.Println("  Generating Nginx configuration...")
	}
	
	// Copy dhparam.pem (it's not a template, just a static file)
	dhparamSrc := filepath.Join(i.config.Paths.Data, "dhparam.pem")
	if _, err := os.Stat(dhparamSrc); os.IsNotExist(err) {
		// Generate dhparam if not exists (this takes a while)
		if i.verbose {
			fmt.Println("    Generating DH parameters (this may take several minutes)...")
		}
		cmd := exec.Command("openssl", "dhparam", "-out", dhparamSrc, "2048")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to generate dhparam: %w", err)
		}
	}
	
	nginxConfigs := map[string]string{
		"templates/nginx/nginx.conf":       "/etc/nginx/nginx.conf",
		"templates/nginx/proxy.conf":       "/etc/nginx/proxy.conf",
		"templates/nginx/tls.conf":         "/etc/nginx/tls.conf",
	}

	for template, output := range nginxConfigs {
		if i.verbose {
			fmt.Printf("    %s\n", output)
		}
		if err := renderer.RenderToFile(template, output); err != nil {
			return fmt.Errorf("failed to render %s: %w", template, err)
		}
	}

	// Generate Webmail configs if enabled
	if i.config.Webmail != "none" && i.config.Services.Webmail {
		if i.verbose {
			fmt.Printf("  Generating Webmail configuration (%s)...\n", i.config.Webmail)
		}
		
		webmailConfigs := map[string]string{
			"templates/webmails/nginx-webmail.conf": "/etc/nginx/sites-available/webmail.conf",
			"templates/webmails/php-webmail.conf":   "/etc/php/8.1/fpm/pool.d/webmail.conf",
			"templates/webmails/php.ini":            "/etc/php/8.1/fpm/conf.d/99-mailstack.ini",
			"templates/webmails/snuffleupagus.rules": "/etc/snuffleupagus.rules",
		}

		for template, output := range webmailConfigs {
			if i.verbose {
				fmt.Printf("    %s\n", output)
			}
			if err := renderer.RenderToFile(template, output); err != nil {
				return fmt.Errorf("failed to render %s: %w", template, err)
			}
		}

		// Generate webmail-specific configs
		if i.config.Webmail == "roundcube" {
			roundcubeConfigs := map[string]string{
				"templates/webmails/roundcube/config.inc.php":         "/var/www/roundcube/config/config.inc.php",
				"templates/webmails/roundcube/config.inc.carddav.php": "/var/www/roundcube/config/config.inc.carddav.php",
			}
			for template, output := range roundcubeConfigs {
				if i.verbose {
					fmt.Printf("    %s\n", output)
				}
				if err := renderer.RenderToFile(template, output); err != nil {
					return fmt.Errorf("failed to render %s: %w", template, err)
				}
			}
		} else if i.config.Webmail == "snappymail" {
			snappymailConfigs := map[string]string{
				"templates/webmails/snappymail/application.ini": "/var/www/snappymail/data/_data_/_default_/configs/application.ini",
				"templates/webmails/snappymail/default.json":    "/var/www/snappymail/data/_data_/_default_/domains/default.json",
			}
			for template, output := range snappymailConfigs {
				if i.verbose {
					fmt.Printf("    %s\n", output)
				}
				if err := renderer.RenderToFile(template, output); err != nil {
					return fmt.Errorf("failed to render %s: %w", template, err)
				}
			}
		}
		
		// Enable webmail nginx site
		webmailLink := "/etc/nginx/sites-enabled/webmail.conf"
		if _, err := os.Stat(webmailLink); os.IsNotExist(err) {
			if err := os.Symlink("/etc/nginx/sites-available/webmail.conf", webmailLink); err != nil {
				return fmt.Errorf("failed to enable webmail site: %w", err)
			}
		}
	}

	if i.verbose {
		fmt.Println("  ✓ All configuration files generated")
	}

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
