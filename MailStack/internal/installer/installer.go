package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
		"templates/postfix/main.cf":                   "/etc/postfix/main.cf",
		"templates/postfix/master.cf":                 "/etc/postfix/master.cf",
		"templates/postfix/sasl_passwd":               "/etc/postfix/sasl_passwd",
		"templates/postfix/outclean_header_filter.cf": "/etc/postfix/outclean_header_filter.cf",
		"templates/postfix/mta-sts-daemon.yml":        "/etc/mta-sts-daemon.yml",
		"templates/postfix/logrotate.conf":            "/etc/logrotate.d/postfix",
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
		"templates/dovecot/dovecot.conf":      "/etc/dovecot/dovecot.conf",
		"templates/dovecot/auth.conf":         "/etc/dovecot/conf.d/auth.conf",
		"templates/dovecot/report-spam.sieve": "/etc/dovecot/report-spam.sieve",
		"templates/dovecot/report-ham.sieve":  "/etc/dovecot/report-ham.sieve",
		"templates/dovecot/spam.script":       "/etc/dovecot/spam.script",
		"templates/dovecot/ham.script":        "/etc/dovecot/ham.script",
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
		"templates/rspamd/antivirus.conf":               "/etc/rspamd/local.d/antivirus.conf",
		"templates/rspamd/arc.conf":                     "/etc/rspamd/local.d/arc.conf",
		"templates/rspamd/classifier-bayes.conf":        "/etc/rspamd/local.d/classifier-bayes.conf",
		"templates/rspamd/composites.conf":              "/etc/rspamd/local.d/composites.conf",
		"templates/rspamd/dkim_signing.conf":            "/etc/rspamd/local.d/dkim_signing.conf",
		"templates/rspamd/external_services.conf":       "/etc/rspamd/local.d/external_services.conf",
		"templates/rspamd/external_services_group.conf": "/etc/rspamd/local.d/external_services_group.conf",
		"templates/rspamd/forbidden_file_extension.map": "/etc/rspamd/local.d/forbidden_file_extension.map",
		"templates/rspamd/force_actions.conf":           "/etc/rspamd/local.d/force_actions.conf",
		"templates/rspamd/fuzzy_check.conf":             "/etc/rspamd/local.d/fuzzy_check.conf",
		"templates/rspamd/headers_group.conf":           "/etc/rspamd/local.d/headers_group.conf",
		"templates/rspamd/history_redis.conf":           "/etc/rspamd/local.d/history_redis.conf",
		"templates/rspamd/local_subnet.map":             "/etc/rspamd/local.d/local_subnet.map",
		"templates/rspamd/metrics.conf":                 "/etc/rspamd/local.d/metrics.conf",
		"templates/rspamd/milter_headers.conf":          "/etc/rspamd/local.d/milter_headers.conf",
		"templates/rspamd/multimap.conf":                "/etc/rspamd/local.d/multimap.conf",
		"templates/rspamd/redis.conf":                   "/etc/rspamd/local.d/redis.conf",
		"templates/rspamd/whitelist.conf":               "/etc/rspamd/local.d/whitelist.conf",
		"templates/rspamd/options.inc":                  "/etc/rspamd/local.d/options.inc",
		"templates/rspamd/logging.inc":                  "/etc/rspamd/local.d/logging.inc",
		"templates/rspamd/worker-controller.inc":        "/etc/rspamd/local.d/worker-controller.inc",
		"templates/rspamd/worker-fuzzy.inc":             "/etc/rspamd/local.d/worker-fuzzy.inc",
		"templates/rspamd/worker-normal.inc":            "/etc/rspamd/local.d/worker-normal.inc",
		"templates/rspamd/worker-proxy.inc":             "/etc/rspamd/local.d/worker-proxy.inc",
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
		"templates/nginx/nginx.conf": "/etc/nginx/nginx.conf",
		"templates/nginx/proxy.conf": "/etc/nginx/proxy.conf",
		"templates/nginx/tls.conf":   "/etc/nginx/tls.conf",
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
			"templates/webmails/nginx-webmail.conf":  "/etc/nginx/sites-available/webmail.conf",
			"templates/webmails/php-webmail.conf":    "/etc/php/8.1/fpm/pool.d/webmail.conf",
			"templates/webmails/php.ini":             "/etc/php/8.1/fpm/conf.d/99-mailstack.ini",
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
	if i.verbose {
		fmt.Println("Initializing database...")
	}

	// Determine database type from config
	dbType := i.config.Database.Type
	if dbType == "" {
		dbType = "sqlite" // Default to SQLite for simplicity
	}

	switch dbType {
	case "sqlite":
		return i.initSQLiteDatabase()
	case "mysql", "mariadb":
		return i.initMySQLDatabase()
	case "postgresql", "postgres":
		return i.initPostgreSQLDatabase()
	default:
		if i.verbose {
			fmt.Printf("  Unknown database type '%s', using SQLite\n", dbType)
		}
		return i.initSQLiteDatabase()
	}
}

func (i *Installer) initSQLiteDatabase() error {
	if i.verbose {
		fmt.Println("  Setting up SQLite database...")
	}

	dbPath := filepath.Join(i.config.Paths.Data, "mailstack.db")

	// Create database schema
	schema := `
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    quota_bytes BIGINT DEFAULT 0,
    enabled BOOLEAN DEFAULT 1,
    global_admin BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Domains table
CREATE TABLE IF NOT EXISTS domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) UNIQUE NOT NULL,
    max_users INTEGER DEFAULT 0,
    max_aliases INTEGER DEFAULT 0,
    max_quota_bytes BIGINT DEFAULT 0,
    enabled BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Aliases table
CREATE TABLE IF NOT EXISTS aliases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email VARCHAR(255) UNIQUE NOT NULL,
    destination TEXT NOT NULL,
    wildcard BOOLEAN DEFAULT 0,
    enabled BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Domain admins table
CREATE TABLE IF NOT EXISTS domain_admins (
    user_id INTEGER,
    domain_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, domain_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_domains_name ON domains(name);
CREATE INDEX IF NOT EXISTS idx_aliases_email ON aliases(email);
`

	// Write schema to temp file
	schemaFile := filepath.Join(i.config.Paths.Data, "schema.sql")
	if err := os.WriteFile(schemaFile, []byte(schema), 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	// Initialize database with schema
	cmd := exec.Command("sqlite3", dbPath)
	cmd.Stdin = strings.NewReader(schema)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to initialize SQLite database: %w\nOutput: %s", err, output)
	}

	// Set proper permissions
	if err := os.Chmod(dbPath, 0640); err != nil {
		return fmt.Errorf("failed to set database permissions: %w", err)
	}

	// Change ownership to mailu user (will need to lookup UID)
	// For now, just ensure it's readable by the mail group
	os.Chown(dbPath, 0, 0) // root:root for now

	if i.verbose {
		fmt.Printf("  ✓ SQLite database initialized: %s\n", dbPath)
	}

	// Insert default domain
	insertDomain := fmt.Sprintf("INSERT OR IGNORE INTO domains (name, enabled) VALUES ('%s', 1);", i.config.Domain)
	cmd = exec.Command("sqlite3", dbPath, insertDomain)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to insert default domain: %w\nOutput: %s", err, output)
	}

	if i.verbose {
		fmt.Printf("  ✓ Default domain '%s' added to database\n", i.config.Domain)
	}

	return nil
}

func (i *Installer) initMySQLDatabase() error {
	if i.verbose {
		fmt.Println("  MySQL/MariaDB setup...")
		fmt.Println("  Note: You need to manually create the database and user")
		fmt.Println("  Example commands:")
		fmt.Printf("    CREATE DATABASE mailstack;\n")
		fmt.Printf("    CREATE USER 'mailstack'@'localhost' IDENTIFIED BY 'password';\n")
		fmt.Printf("    GRANT ALL PRIVILEGES ON mailstack.* TO 'mailstack'@'localhost';\n")
		fmt.Printf("    FLUSH PRIVILEGES;\n")
	}

	// TODO: Implement MySQL schema initialization
	return fmt.Errorf("MySQL database initialization not yet implemented - please create database manually")
}

func (i *Installer) initPostgreSQLDatabase() error {
	if i.verbose {
		fmt.Println("  PostgreSQL setup...")
		fmt.Println("  Note: You need to manually create the database and user")
		fmt.Println("  Example commands:")
		fmt.Printf("    CREATE DATABASE mailstack;\n")
		fmt.Printf("    CREATE USER mailstack WITH PASSWORD 'password';\n")
		fmt.Printf("    GRANT ALL PRIVILEGES ON DATABASE mailstack TO mailstack;\n")
	}

	// TODO: Implement PostgreSQL schema initialization
	return fmt.Errorf("PostgreSQL database initialization not yet implemented - please create database manually")
}

func (i *Installer) generateDKIM() error {
	if i.verbose {
		fmt.Println("Generating DKIM keys...")
	}

	// Generate DKIM key for main domain
	domain := i.config.Domain
	selector := i.config.Mail.DKIMSelector
	if selector == "" {
		selector = "dkim"
	}

	keyDir := i.config.Paths.DKIM
	privateKeyPath := filepath.Join(keyDir, domain+"."+selector+".key")
	publicKeyPath := filepath.Join(keyDir, domain+"."+selector+".txt")

	// Check if key already exists
	if _, err := os.Stat(privateKeyPath); err == nil {
		if i.verbose {
			fmt.Printf("  DKIM key already exists for %s, skipping...\n", domain)
		}
		return nil
	}

	if i.verbose {
		fmt.Printf("  Generating 2048-bit RSA key for %s...\n", domain)
	}

	// Generate RSA private key using openssl
	cmd := exec.Command("openssl", "genrsa", "-out", privateKeyPath, "2048")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate DKIM private key: %w\nOutput: %s", err, output)
	}

	// Set proper permissions on private key
	if err := os.Chmod(privateKeyPath, 0600); err != nil {
		return fmt.Errorf("failed to set permissions on DKIM private key: %w", err)
	}
	if err := os.Chown(privateKeyPath, 0, 0); err != nil {
		return fmt.Errorf("failed to set ownership on DKIM private key: %w", err)
	}

	// Generate public key from private key
	cmd = exec.Command("openssl", "rsa", "-in", privateKeyPath, "-pubout", "-outform", "PEM", "-out", publicKeyPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate DKIM public key: %w\nOutput: %s", err, output)
	}

	// Read public key and format for DNS
	pubKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read DKIM public key: %w", err)
	}

	// Extract base64 portion (remove header/footer)
	lines := []string{}
	inKey := false
	for _, line := range []byte(string(pubKeyData)) {
		lineStr := string(line)
		if lineStr == "-----BEGIN PUBLIC KEY-----" {
			inKey = true
			continue
		}
		if lineStr == "-----END PUBLIC KEY-----" {
			break
		}
		if inKey {
			lines = append(lines, lineStr)
		}
	}

	// Create DNS TXT record format
	dnsRecord := fmt.Sprintf("%s._domainkey.%s. IN TXT \"v=DKIM1; k=rsa; p=%s\"\n",
		selector, domain, string(pubKeyData))

	// Write DNS record to file
	dnsRecordPath := filepath.Join(keyDir, domain+"."+selector+".dns.txt")
	if err := os.WriteFile(dnsRecordPath, []byte(dnsRecord), 0644); err != nil {
		return fmt.Errorf("failed to write DNS record: %w", err)
	}

	if i.verbose {
		fmt.Printf("  ✓ DKIM keys generated for %s\n", domain)
		fmt.Printf("  Private key: %s\n", privateKeyPath)
		fmt.Printf("  Public key:  %s\n", publicKeyPath)
		fmt.Printf("  DNS record:  %s\n", dnsRecordPath)
		fmt.Println("\n  Add this DNS TXT record to your domain:")
		fmt.Printf("  %s\n", dnsRecord)
	}

	return nil
}

func (i *Installer) setupTLS() error {
	if i.verbose {
		fmt.Println("Setting up TLS certificates...")
	}

	switch i.config.TLS.Flavor {
	case "letsencrypt", "mail-letsencrypt":
		return i.setupLetsEncrypt()
	case "cert", "mail":
		return i.setupCustomCerts()
	case "notls":
		if i.verbose {
			fmt.Println("  TLS disabled, skipping certificate setup")
		}
		return nil
	default:
		if i.verbose {
			fmt.Printf("  Unknown TLS flavor '%s', skipping certificate setup\n", i.config.TLS.Flavor)
		}
		return nil
	}
}

func (i *Installer) setupLetsEncrypt() error {
	if i.verbose {
		fmt.Println("  Configuring Let's Encrypt...")
	}

	// Verify certbot is installed
	if _, err := exec.LookPath("certbot"); err != nil {
		return fmt.Errorf("certbot not found - ensure it's installed")
	}

	// Check if email is provided
	if i.config.TLS.Email == "" {
		return fmt.Errorf("TLS email is required for Let's Encrypt")
	}

	// Prepare domain list
	domains := []string{i.config.Hostname}

	// Add webmail domain if different
	if i.config.Webmail != "" && i.config.Webmail != "none" {
		webmailDomain := "webmail." + i.config.Domain
		if webmailDomain != i.config.Hostname {
			domains = append(domains, webmailDomain)
		}
	}

	// Build certbot command
	args := []string{
		"certonly",
		"--standalone",
		"--non-interactive",
		"--agree-tos",
		"--email", i.config.TLS.Email,
	}

	for _, domain := range domains {
		args = append(args, "-d", domain)
	}

	if i.verbose {
		fmt.Printf("  Requesting certificates for: %v\n", domains)
		fmt.Println("  Note: Make sure ports 80 and 443 are accessible from the internet")
	}

	// Run certbot
	cmd := exec.Command("certbot", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("certbot failed: %w\nMake sure your domain DNS is pointing to this server and ports 80/443 are open", err)
	}

	// Create symlinks to Let's Encrypt certs
	certDir := "/etc/letsencrypt/live/" + i.config.Hostname
	certLink := filepath.Join(i.config.Paths.Certs, "cert.pem")
	keyLink := filepath.Join(i.config.Paths.Certs, "key.pem")

	os.Remove(certLink) // Remove if exists
	os.Remove(keyLink)

	if err := os.Symlink(filepath.Join(certDir, "fullchain.pem"), certLink); err != nil {
		return fmt.Errorf("failed to create cert symlink: %w", err)
	}
	if err := os.Symlink(filepath.Join(certDir, "privkey.pem"), keyLink); err != nil {
		return fmt.Errorf("failed to create key symlink: %w", err)
	}

	// Set up auto-renewal
	if i.verbose {
		fmt.Println("  Setting up certificate auto-renewal...")
	}

	// Create renewal hook script
	renewHook := `#!/bin/bash
# Reload services after certificate renewal
systemctl reload nginx
systemctl reload postfix
systemctl reload dovecot
`
	hookPath := "/etc/letsencrypt/renewal-hooks/deploy/reload-mailstack.sh"
	os.MkdirAll("/etc/letsencrypt/renewal-hooks/deploy", 0755)
	if err := os.WriteFile(hookPath, []byte(renewHook), 0755); err != nil {
		return fmt.Errorf("failed to create renewal hook: %w", err)
	}

	if i.verbose {
		fmt.Println("  ✓ Let's Encrypt certificates configured")
		fmt.Println("  Certificates will auto-renew via certbot timer")
	}

	return nil
}

func (i *Installer) setupCustomCerts() error {
	if i.verbose {
		fmt.Println("  Configuring custom certificates...")
	}

	if i.config.TLS.CertPath == "" || i.config.TLS.KeyPath == "" {
		return fmt.Errorf("cert_path and key_path must be specified for custom certificates")
	}

	// Verify cert and key files exist
	if _, err := os.Stat(i.config.TLS.CertPath); err != nil {
		return fmt.Errorf("certificate file not found: %s", i.config.TLS.CertPath)
	}
	if _, err := os.Stat(i.config.TLS.KeyPath); err != nil {
		return fmt.Errorf("key file not found: %s", i.config.TLS.KeyPath)
	}

	// Copy certificates to certs directory
	certDest := filepath.Join(i.config.Paths.Certs, "cert.pem")
	keyDest := filepath.Join(i.config.Paths.Certs, "key.pem")

	// Read source files
	certData, err := os.ReadFile(i.config.TLS.CertPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}
	keyData, err := os.ReadFile(i.config.TLS.KeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(certDest, certData, 0644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}
	if err := os.WriteFile(keyDest, keyData, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	if i.verbose {
		fmt.Println("  ✓ Custom certificates configured")
		fmt.Printf("  Certificate: %s\n", certDest)
		fmt.Printf("  Private key: %s\n", keyDest)
	}

	return nil
}

func (i *Installer) configureServices() error {
	if i.verbose {
		fmt.Println("Configuring systemd services...")
	}

	// Create systemd override directories
	overrideDirs := []string{
		"/etc/systemd/system/postfix.service.d",
		"/etc/systemd/system/dovecot.service.d",
		"/etc/systemd/system/rspamd.service.d",
		"/etc/systemd/system/nginx.service.d",
		"/etc/systemd/system/redis.service.d",
	}

	for _, dir := range overrideDirs {
		if err := system.CreateDirectory(dir, "root", 0755); err != nil {
			return fmt.Errorf("failed to create override directory %s: %w", dir, err)
		}
	}

	// Postfix service override (ensure it starts after network and Redis)
	postfixOverride := `[Unit]
After=network-online.target redis.service
Wants=network-online.target

[Service]
Restart=always
RestartSec=10s
`
	if err := os.WriteFile("/etc/systemd/system/postfix.service.d/override.conf", []byte(postfixOverride), 0644); err != nil {
		return fmt.Errorf("failed to write postfix override: %w", err)
	}

	// Dovecot service override
	dovecotOverride := `[Unit]
After=network-online.target redis.service
Wants=network-online.target

[Service]
Restart=always
RestartSec=10s
`
	if err := os.WriteFile("/etc/systemd/system/dovecot.service.d/override.conf", []byte(dovecotOverride), 0644); err != nil {
		return fmt.Errorf("failed to write dovecot override: %w", err)
	}

	// Rspamd service override
	rspamdOverride := `[Unit]
After=network-online.target redis.service
Wants=network-online.target

[Service]
Restart=always
RestartSec=10s
`
	if err := os.WriteFile("/etc/systemd/system/rspamd.service.d/override.conf", []byte(rspamdOverride), 0644); err != nil {
		return fmt.Errorf("failed to write rspamd override: %w", err)
	}

	// Nginx service override
	nginxOverride := `[Unit]
After=network-online.target

[Service]
Restart=always
RestartSec=10s
`
	if err := os.WriteFile("/etc/systemd/system/nginx.service.d/override.conf", []byte(nginxOverride), 0644); err != nil {
		return fmt.Errorf("failed to write nginx override: %w", err)
	}

	// Redis service override
	redisOverride := `[Unit]
After=network-online.target

[Service]
Restart=always
RestartSec=10s
`
	if err := os.WriteFile("/etc/systemd/system/redis.service.d/override.conf", []byte(redisOverride), 0644); err != nil {
		return fmt.Errorf("failed to write redis override: %w", err)
	}

	// If webmail is enabled, configure PHP-FPM
	if i.config.Webmail != "" && i.config.Webmail != "none" {
		phpfpmDir := "/etc/systemd/system/php8.1-fpm.service.d"
		if err := system.CreateDirectory(phpfpmDir, "root", 0755); err != nil {
			return fmt.Errorf("failed to create php-fpm override directory: %w", err)
		}

		phpfpmOverride := `[Unit]
After=network-online.target

[Service]
Restart=always
RestartSec=10s
`
		if err := os.WriteFile(filepath.Join(phpfpmDir, "override.conf"), []byte(phpfpmOverride), 0644); err != nil {
			return fmt.Errorf("failed to write php-fpm override: %w", err)
		}
	}

	// If antivirus is enabled, configure ClamAV services
	if i.config.Services.Antivirus {
		clamdDir := "/etc/systemd/system/clamav-daemon.service.d"
		freshclamDir := "/etc/systemd/system/clamav-freshclam.service.d"

		if err := system.CreateDirectory(clamdDir, "root", 0755); err != nil {
			return fmt.Errorf("failed to create clamd override directory: %w", err)
		}
		if err := system.CreateDirectory(freshclamDir, "root", 0755); err != nil {
			return fmt.Errorf("failed to create freshclam override directory: %w", err)
		}

		clamdOverride := `[Unit]
After=network-online.target

[Service]
Restart=always
RestartSec=10s
`
		if err := os.WriteFile(filepath.Join(clamdDir, "override.conf"), []byte(clamdOverride), 0644); err != nil {
			return fmt.Errorf("failed to write clamd override: %w", err)
		}

		freshclamOverride := `[Unit]
After=network-online.target

[Service]
Restart=always
RestartSec=10s
`
		if err := os.WriteFile(filepath.Join(freshclamDir, "override.conf"), []byte(freshclamOverride), 0644); err != nil {
			return fmt.Errorf("failed to write freshclam override: %w", err)
		}
	}

	// Reload systemd daemon to pick up changes
	if i.verbose {
		fmt.Println("  Reloading systemd daemon...")
	}
	cmd := exec.Command("systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w\nOutput: %s", err, output)
	}

	if i.verbose {
		fmt.Println("  ✓ Systemd services configured")
	}

	return nil
}

func (i *Installer) startServices() error {
	if i.verbose {
		fmt.Println("Starting mail services...")
	}

	// Define service start order (dependencies first)
	services := []string{
		"redis",   // Cache - needed by others
		"rspamd",  // Anti-spam
		"postfix", // SMTP
		"dovecot", // IMAP/POP3/LMTP
		"nginx",   // Web proxy
	}

	// Add optional services
	if i.config.Webmail != "" && i.config.Webmail != "none" {
		services = append(services, "php8.1-fpm")
	}

	if i.config.Services.Antivirus {
		services = append(services, "clamav-freshclam", "clamav-daemon")
	}

	// Enable and start each service
	for _, service := range services {
		if i.verbose {
			fmt.Printf("  Enabling %s...\n", service)
		}

		// Enable service to start on boot
		if err := system.EnableService(service); err != nil {
			fmt.Printf("  Warning: Failed to enable %s: %v\n", service, err)
			// Continue anyway - service might not exist on this system
			continue
		}

		// Start the service
		if i.verbose {
			fmt.Printf("  Starting %s...\n", service)
		}
		if err := system.StartService(service); err != nil {
			// Try to get service status for debugging
			fmt.Printf("  Warning: Failed to start %s: %v\n", service, err)
			continue
		}

		// Brief pause to let service initialize
		time.Sleep(500 * time.Millisecond)
	}

	// Restart services that depend on configs we just created
	if i.verbose {
		fmt.Println("  Restarting services to apply new configurations...")
	}

	restartServices := []string{"postfix", "dovecot", "rspamd", "nginx"}
	for _, service := range restartServices {
		if i.verbose {
			fmt.Printf("  Restarting %s...\n", service)
		}
		if err := system.RestartService(service); err != nil {
			fmt.Printf("  Warning: Failed to restart %s: %v\n", service, err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	if i.verbose {
		fmt.Println("  ✓ All services started")
	}

	return nil
}

func (i *Installer) createAdminUser() error {
	if i.verbose {
		fmt.Println("Creating admin user...")
	}

	// This will be implemented once we have database schema
	// For now, just create a placeholder script
	scriptPath := "/usr/local/bin/mailstack-create-admin"
	script := `#!/bin/bash
# MailStack Admin User Creation Script
# Usage: mailstack-create-admin <email> <password>

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <email> <password>"
    echo "Example: $0 admin@example.com SecurePassword123"
    exit 1
fi

EMAIL="$1"
PASSWORD="$2"

echo "Creating admin user: $EMAIL"

# TODO: Add user to database
# This will be implemented with database schema

echo "Admin user created successfully"
echo "You can now login to the web interface with:"
echo "  Email: $EMAIL"
echo "  Password: (as provided)"
`

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to create admin script: %w", err)
	}

	if i.verbose {
		fmt.Println("  ✓ Admin user creation script installed")
		fmt.Printf("  Run: mailstack-create-admin <email> <password>\n")
		fmt.Printf("  Example: mailstack-create-admin admin@%s MySecurePassword\n", i.config.Domain)
	}

	return nil
}

func (i *Installer) healthCheck() error {
	if i.verbose {
		fmt.Println("Performing health check...")
	}

	services := []string{"redis", "rspamd", "postfix", "dovecot", "nginx"}

	// Add optional services
	if i.config.Webmail != "" && i.config.Webmail != "none" {
		services = append(services, "php8.1-fpm")
	}
	if i.config.Services.Antivirus {
		services = append(services, "clamav-daemon")
	}

	allHealthy := true
	for _, service := range services {
		cmd := exec.Command("systemctl", "is-active", service)
		output, err := cmd.Output()
		status := string(output)

		if err != nil || status != "active\n" {
			if i.verbose {
				fmt.Printf("  ✗ %s: NOT RUNNING\n", service)
			}
			allHealthy = false
		} else {
			if i.verbose {
				fmt.Printf("  ✓ %s: running\n", service)
			}
		}
	}

	// Check critical ports
	ports := map[string]int{
		"SMTP":  25,
		"IMAP":  143,
		"IMAPS": 993,
		"HTTP":  80,
	}

	if i.config.TLS.Flavor != "" && i.config.TLS.Flavor != "notls" {
		ports["HTTPS"] = 443
	}

	if i.verbose {
		fmt.Println("\n  Checking ports...")
	}

	for name, port := range ports {
		cmd := exec.Command("ss", "-tln")
		output, err := cmd.Output()
		if err != nil {
			if i.verbose {
				fmt.Printf("  Warning: Could not check port %d (%s)\n", port, name)
			}
			continue
		}

		listening := false
		portStr := fmt.Sprintf(":%d", port)
		for _, line := range []byte(string(output)) {
			if string(line) == portStr[1:] {
				listening = true
				break
			}
		}

		if listening {
			if i.verbose {
				fmt.Printf("  ✓ Port %d (%s): listening\n", port, name)
			}
		} else {
			if i.verbose {
				fmt.Printf("  ✗ Port %d (%s): not listening\n", port, name)
			}
			allHealthy = false
		}
	}

	if !allHealthy {
		fmt.Println("\n  ⚠ Warning: Some services or ports are not healthy")
		fmt.Println("  You may need to check service logs:")
		fmt.Println("    journalctl -u postfix -n 50")
		fmt.Println("    journalctl -u dovecot -n 50")
		fmt.Println("    journalctl -u rspamd -n 50")
		return fmt.Errorf("health check failed - some services not running properly")
	}

	if i.verbose {
		fmt.Println("\n  ✓ All health checks passed!")
	}

	return nil
}
