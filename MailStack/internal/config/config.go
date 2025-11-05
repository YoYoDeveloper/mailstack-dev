package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config represents the main MailStack configuration
type Config struct {
	Domain      string          `json:"domain"`
	Hostname    string          `json:"hostname"`
	Hostnames   []string        `json:"hostnames,omitempty"`
	Postmaster  string          `json:"postmaster"`
	Admin       AdminConfig     `json:"admin"`
	Database    DatabaseConfig  `json:"database"`
	TLS         TLSConfig       `json:"tls"`
	Mail        MailConfig      `json:"mail"`
	Web         WebConfig       `json:"web"`
	Services    ServicesConfig  `json:"services"`
	Network     NetworkConfig   `json:"network"`
	Paths       PathsConfig     `json:"paths"`
	DKIMPath    string          `json:"dkim_path"`
	SecretKey   string          `json:"secret_key"`
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
}

// TLSConfig for TLS/SSL configuration
type TLSConfig struct {
	Flavor string `json:"flavor"` // letsencrypt, cert, mail-letsencrypt, mail, notls
	Email  string `json:"email,omitempty"`
}

// MailConfig for mail server settings
type MailConfig struct {
	MessageSizeLimit    int64  `json:"message_size_limit"`
	MessageRateLimit    string `json:"message_ratelimit"`
	DefaultQuota        int64  `json:"default_quota"`
	RecipientDelimiter  string `json:"recipient_delimiter"`
	DKIMSelector        string `json:"dkim_selector"`
	RelayHost           string `json:"relay_host,omitempty"`
	RelayUser           string `json:"relay_user,omitempty"`
	RelayPassword       string `json:"relay_password,omitempty"`
}

// WebConfig for web interface
type WebConfig struct {
	AdminPath   string `json:"admin_path"`
	WebmailPath string `json:"webmail_path"`
	Sitename    string `json:"sitename"`
	Website     string `json:"website"`
}

// ServicesConfig for optional services
type ServicesConfig struct {
	Antivirus  bool `json:"antivirus"`
	Webmail    bool `json:"webmail"`
	Fetchmail  bool `json:"fetchmail"`
	Webdav     bool `json:"webdav"`
	Oletools   bool `json:"oletools"`
}

// NetworkConfig for network settings
type NetworkConfig struct {
	Subnet       string `json:"subnet"`
	Subnet6      string `json:"subnet6,omitempty"`
	BindIPv4     string `json:"bind_ipv4"`
	BindIPv6     string `json:"bind_ipv6,omitempty"`
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
}
