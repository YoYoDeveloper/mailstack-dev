package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config represents the main MailStack configuration
type Config struct {
	Domain     string         `json:"domain"`
	Hostname   string         `json:"hostname"`
	Hostnames  []string       `json:"hostnames,omitempty"`
	Postmaster string         `json:"postmaster"`
	Admin      AdminConfig    `json:"admin"`
	Database   DatabaseConfig `json:"database"`
	TLS        TLSConfig      `json:"tls"`
	Mail       MailConfig     `json:"mail"`
	Web        WebConfig      `json:"web"`
	Services   ServicesConfig `json:"services"`
	Network    NetworkConfig  `json:"network"`
	Paths      PathsConfig    `json:"paths"`
	DKIMPath   string         `json:"dkim_path"`
	SecretKey  string         `json:"secret_key"`

	// Service addresses
	FrontAddress    string `json:"front_address,omitempty"`
	AdminAddress    string `json:"admin_address,omitempty"`
	AntispamAddress string `json:"antispam_address,omitempty"`
	WebmailAddress  string `json:"webmail_address,omitempty"`
	WebdavAddress   string `json:"webdav_address,omitempty"`
	RedisAddress    string `json:"redis_address,omitempty"`
	Resolver        string `json:"resolver,omitempty"`

	// Security keys
	RoundcubeKey     string `json:"roundcube_key,omitempty"`
	SnuffleupagusKey string `json:"snuffleupagus_key,omitempty"`

	// Webmail settings
	Webmail                  string   `json:"webmail,omitempty"` // roundcube, snappymail, none
	Plugins                  string   `json:"plugins,omitempty"`
	Includes                 []string `json:"includes,omitempty"`
	PermanentSessionLifetime int      `json:"permanent_session_lifetime,omitempty"`
	FullTextSearch           bool     `json:"full_text_search,omitempty"`

	// Additional settings
	Timezone        string `json:"timezone,omitempty"`
	MaxFilesize     int    `json:"max_filesize,omitempty"` // in MB
	RealIPHeader    string `json:"real_ip_header,omitempty"`
	RealIPFrom      string `json:"real_ip_from,omitempty"`
	RelayNets       string `json:"relay_nets,omitempty"`
	WebrootRedirect string `json:"webroot_redirect,omitempty"`

	// Port and protocol settings
	Port80           bool `json:"port_80,omitempty"`
	ProxyProtocol25  bool `json:"proxy_protocol_25,omitempty"`
	ProxyProtocol80  bool `json:"proxy_protocol_80,omitempty"`
	ProxyProtocol443 bool `json:"proxy_protocol_443,omitempty"`
	TLS443           bool `json:"tls_443,omitempty"`
	TLSError         bool `json:"tls_error,omitempty"`
	TLSPermissive    bool `json:"tls_permissive,omitempty"`

	// Feature flags
	API            bool `json:"api,omitempty"`
	EnableOletools bool `json:"enable_oletools,omitempty"`
}

// AdminConfig for admin user
type AdminConfig struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// DatabaseConfig for database connection
type DatabaseConfig struct {
	Type     string `json:"type"` // sqlite, postgresql, mysql
	Path     string `json:"path,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Name     string `json:"name,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	DSN      string `json:"dsn,omitempty"`     // Full DSN string
	DBDsnw   string `json:"db_dsnw,omitempty"` // For roundcube
}

// TLSConfig for TLS/SSL configuration
type TLSConfig struct {
	Flavor   string   `json:"flavor"` // letsencrypt, cert, mail-letsencrypt, mail, notls
	Email    string   `json:"email,omitempty"`
	CertPath string   `json:"cert_path,omitempty"`
	KeyPath  string   `json:"key_path,omitempty"`
	TLS      []string `json:"tls,omitempty"` // Array of cert/key paths for nginx
}

// MailConfig for mail server settings
type MailConfig struct {
	MessageSizeLimit   int64  `json:"message_size_limit"`
	MessageRateLimit   string `json:"message_ratelimit"`
	DefaultQuota       int64  `json:"default_quota"`
	RecipientDelimiter string `json:"recipient_delimiter"`
	DKIMSelector       string `json:"dkim_selector"`
	RelayHost          string `json:"relay_host,omitempty"`
	RelayUser          string `json:"relay_user,omitempty"`
	RelayPassword      string `json:"relay_password,omitempty"`
}

// WebConfig for web interface
type WebConfig struct {
	AdminPath   string `json:"admin_path"`
	WebmailPath string `json:"webmail_path"`
	WebAdmin    string `json:"web_admin,omitempty"`   // Full admin URL path
	WebWebmail  string `json:"web_webmail,omitempty"` // Full webmail URL path
	WebAPI      string `json:"web_api,omitempty"`     // API URL path
	Sitename    string `json:"sitename"`
	Website     string `json:"website"`
}

// ServicesConfig for optional services
type ServicesConfig struct {
	Antivirus bool `json:"antivirus"`
	// Webmail can be a string to indicate which webmail package to enable
	// (e.g. "roundcube", "snappymail") or "none" to disable.
	Webmail   string `json:"webmail,omitempty"`
	Fetchmail bool `json:"fetchmail"`
	Webdav    bool `json:"webdav"`
	Oletools  bool `json:"oletools"`
}

// NetworkConfig for network settings
type NetworkConfig struct {
	Subnet        string `json:"subnet"`
	Subnet6       string `json:"subnet6,omitempty"`
	BindIPv4      string `json:"bind_ipv4"`
	BindIPv6      string `json:"bind_ipv6,omitempty"`
	RelayNetworks string `json:"relay_networks,omitempty"`
}

// PathsConfig for data paths
type PathsConfig struct {
	Data      string `json:"data"`
	Mail      string `json:"mail"`
	DKIM      string `json:"dkim"`
	Queue     string `json:"queue"`
	Filter    string `json:"filter"`
	Certs     string `json:"certs"`
	Overrides string `json:"overrides"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	cfg.setDefaults()

	return &cfg, nil
}

// Save writes the configuration to a file
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Domain == "" {
		return fmt.Errorf("domain is required")
	}

	if c.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}

	if c.Admin.Email == "" {
		return fmt.Errorf("admin email is required")
	}

	if c.Admin.Password == "" {
		return fmt.Errorf("admin password is required")
	}

	if c.Database.Type == "" {
		return fmt.Errorf("database type is required")
	}

	validDBTypes := map[string]bool{"sqlite": true, "postgresql": true, "mysql": true}
	if !validDBTypes[c.Database.Type] {
		return fmt.Errorf("invalid database type: %s (must be sqlite, postgresql, or mysql)", c.Database.Type)
	}

	if c.TLS.Flavor == "" {
		return fmt.Errorf("TLS flavor is required")
	}

	validTLSFlavors := map[string]bool{
		"letsencrypt": true, "cert": true, "mail-letsencrypt": true, "mail": true, "notls": true,
	}
	if !validTLSFlavors[c.TLS.Flavor] {
		return fmt.Errorf("invalid TLS flavor: %s", c.TLS.Flavor)
	}

	if strings.HasPrefix(c.TLS.Flavor, "letsencrypt") && c.TLS.Email == "" {
		return fmt.Errorf("TLS email is required for Let's Encrypt")
	}

	return nil
}

// setDefaults sets default values for optional fields
func (c *Config) setDefaults() {
	if c.Postmaster == "" {
		c.Postmaster = "postmaster"
	}

	if c.Mail.MessageSizeLimit == 0 {
		c.Mail.MessageSizeLimit = 50000000 // 50MB
	}

	if c.Mail.MessageRateLimit == "" {
		c.Mail.MessageRateLimit = "200/day"
	}

	if c.Mail.DefaultQuota == 0 {
		c.Mail.DefaultQuota = 1000000000 // 1GB
	}

	if c.Mail.DKIMSelector == "" {
		c.Mail.DKIMSelector = "dkim"
	}

	if c.Web.AdminPath == "" {
		c.Web.AdminPath = "/admin"
	}

	if c.Web.WebmailPath == "" {
		c.Web.WebmailPath = "/webmail"
	}

	if c.Web.Sitename == "" {
		c.Web.Sitename = "MailStack"
	}

	if c.Network.Subnet == "" {
		c.Network.Subnet = "192.168.203.0/24"
	}

	if c.Network.BindIPv4 == "" {
		c.Network.BindIPv4 = "0.0.0.0"
	}

	// Default paths
	if c.Paths.Data == "" {
		c.Paths.Data = "/var/lib/mailstack/data"
	}
	if c.Paths.Mail == "" {
		c.Paths.Mail = "/var/lib/mailstack/mail"
	}
	if c.Paths.DKIM == "" {
		c.Paths.DKIM = "/var/lib/mailstack/dkim"
	}
	if c.Paths.Queue == "" {
		c.Paths.Queue = "/var/lib/mailstack/queue"
	}
	if c.Paths.Filter == "" {
		c.Paths.Filter = "/var/lib/mailstack/filter"
	}
	if c.Paths.Certs == "" {
		c.Paths.Certs = "/var/lib/mailstack/certs"
	}
	if c.Paths.Overrides == "" {
		c.Paths.Overrides = "/etc/mailstack/overrides"
	}

	if c.DKIMPath == "" {
		c.DKIMPath = c.Paths.DKIM + "/{domain}.{selector}.key"
	}

	// Build hostnames list
	if len(c.Hostnames) == 0 {
		c.Hostnames = []string{c.Hostname}
	}

	// Service addresses
	if c.FrontAddress == "" {
		c.FrontAddress = "front"
	}
	if c.AdminAddress == "" {
		c.AdminAddress = "admin"
	}
	if c.AntispamAddress == "" {
		c.AntispamAddress = "antispam"
	}
	if c.RedisAddress == "" {
		c.RedisAddress = "redis:6379"
	}
	if c.Resolver == "" {
		c.Resolver = "8.8.8.8"
	}

	// Webmail defaults
	if c.Webmail == "" {
		c.Webmail = "none"
	}
	if c.PermanentSessionLifetime == 0 {
		c.PermanentSessionLifetime = 10800 // 3 hours
	}
	if c.Plugins == "" && c.Webmail == "roundcube" {
		c.Plugins = "'managesieve', 'markasjunk', 'password'"
	}

	// Web paths
	if c.Web.WebAdmin == "" {
		c.Web.WebAdmin = c.Web.AdminPath
	}
	if c.Web.WebWebmail == "" {
		c.Web.WebWebmail = c.Web.WebmailPath
	}
	if c.Web.WebAPI == "" {
		c.Web.WebAPI = "/api"
	}

	// Timezone
	if c.Timezone == "" {
		c.Timezone = "UTC"
	}

	// Max filesize in MB
	if c.MaxFilesize == 0 {
		c.MaxFilesize = int(c.Mail.MessageSizeLimit / 1048576) // Convert bytes to MB
		if c.MaxFilesize == 0 {
			c.MaxFilesize = 50
		}
	}

	// Port defaults
	if c.TLS.Flavor != "notls" {
		c.Port80 = true
		c.TLS443 = true
	}

	// TLS certificate paths array for nginx template
	if len(c.TLS.TLS) == 0 && c.TLS.Flavor == "cert" {
		c.TLS.TLS = []string{
			c.TLS.CertPath,
			c.TLS.KeyPath,
		}
	}

	// Database DSN construction
	if c.Database.DSN == "" {
		c.Database.DSN = c.buildDSN()
	}
	if c.Database.DBDsnw == "" {
		c.Database.DBDsnw = c.Database.DSN
	}

	// Generate security keys if not provided
	if c.SecretKey == "" {
		c.SecretKey = generateRandomKey(32)
	}
	if c.RoundcubeKey == "" && c.Webmail == "roundcube" {
		c.RoundcubeKey = generateRandomKey(24)
	}
	if c.SnuffleupagusKey == "" && c.Webmail != "none" {
		c.SnuffleupagusKey = generateRandomKey(32)
	}

	// Copy EnableOletools to Services.Oletools for compatibility
	if c.EnableOletools {
		c.Services.Oletools = true
	}

	// Set API flag based on admin settings
	c.API = c.Admin.Email != ""
}

// buildDSN constructs a database DSN string
func (c *Config) buildDSN() string {
	switch c.Database.Type {
	case "sqlite":
		if c.Database.Path != "" {
			return "sqlite:" + c.Database.Path
		}
		return "sqlite:" + c.Paths.Data + "/mailstack.db"
	case "postgresql":
		if c.Database.Host == "" {
			c.Database.Host = "localhost"
		}
		if c.Database.Port == 0 {
			c.Database.Port = 5432
		}
		return fmt.Sprintf("pgsql:host=%s;port=%d;dbname=%s;user=%s;password=%s",
			c.Database.Host, c.Database.Port, c.Database.Name, c.Database.User, c.Database.Password)
	case "mysql":
		if c.Database.Host == "" {
			c.Database.Host = "localhost"
		}
		if c.Database.Port == 0 {
			c.Database.Port = 3306
		}
		return fmt.Sprintf("mysql:host=%s;port=%d;dbname=%s;user=%s;password=%s",
			c.Database.Host, c.Database.Port, c.Database.Name, c.Database.User, c.Database.Password)
	default:
		return ""
	}
}

// generateRandomKey generates a random hex key of specified length
func generateRandomKey(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to deterministic if random fails
		const chars = "0123456789abcdef"
		result := make([]byte, length)
		for i := range result {
			result[i] = chars[i%len(chars)]
		}
		return string(result)
	}
	return hex.EncodeToString(bytes)
}
