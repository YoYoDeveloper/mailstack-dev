package templates

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mailstack/mailstack/internal/config"
)

//go:embed templates/postfix/* templates/dovecot/* templates/rspamd/* templates/nginx/* templates/webmails/**
var templatesFS embed.FS

// Renderer handles template rendering
type Renderer struct {
	config *config.Config
}

// NewRenderer creates a new template renderer
func NewRenderer(cfg *config.Config) *Renderer {
	return &Renderer{config: cfg}
}

// Render renders a template file with the given data
func (r *Renderer) Render(templatePath string) ([]byte, error) {
	// Read template file from embedded FS
	content, err := templatesFS.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(filepath.Base(templatePath)).
		Funcs(r.getFuncMap()).
		Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	// Render template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r.getTemplateData()); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return buf.Bytes(), nil
}

// RenderToFile renders a template and writes it to a file
func (r *Renderer) RenderToFile(templatePath, outputPath string) error {
	content, err := r.Render(templatePath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	return nil
}

// ListTemplates returns all available templates
func ListTemplates(prefix string) ([]string, error) {
	var templates []string

	err := fs.WalkDir(templatesFS, prefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			templates = append(templates, path)
		}
		return nil
	})

	return templates, err
}

// getTemplateData prepares data for template rendering
func (r *Renderer) getTemplateData() map[string]interface{} {
	// Build hostnames list
	hostnames := r.config.Hostnames
	if len(hostnames) == 0 {
		hostnames = []string{r.config.Hostname}
	}

	return map[string]interface{}{
		// Basic domain config
		"Domain":       r.config.Domain,
		"Hostname":     r.config.Hostname,
		"Hostnames":    hostnames,
		"HostnamesStr": strings.Join(hostnames, ","),
		"Postmaster":   r.config.Postmaster,

		// Mail settings
		"MessageSizeLimit":   r.config.Mail.MessageSizeLimit,
		"MessageRateLimit":   r.config.Mail.MessageRateLimit,
		"DefaultQuota":       r.config.Mail.DefaultQuota,
		"RecipientDelimiter": r.config.Mail.RecipientDelimiter,
		"DKIMSelector":       r.config.Mail.DKIMSelector,
		"RelayHost":          r.config.Mail.RelayHost,
		"RelayUser":          r.config.Mail.RelayUser,
		"RelayPassword":      r.config.Mail.RelayPassword,

		// Network settings
		"Subnet":        r.config.Network.Subnet,
		"Subnet6":       r.config.Network.Subnet6,
		"BindIPv4":      r.config.Network.BindIPv4,
		"BindIPv6":      r.config.Network.BindIPv6,
		"RelayNetworks": r.config.Network.RelayNetworks,
		"RelayNets":     r.config.RelayNets,
		"RealIPHeader":  r.config.RealIPHeader,
		"RealIPFrom":    r.config.RealIPFrom,

		// TLS settings
		"TLSFlavor":     r.config.TLS.Flavor,
		"TLS":           r.config.TLS.TLS,
		"TLS443":        r.config.TLS443,
		"TLSError":      r.config.TLSError,
		"TLSPermissive": r.config.TLSPermissive,

		// Web paths
		"AdminPath":       r.config.Web.AdminPath,
		"WebmailPath":     r.config.Web.WebmailPath,
		"WebAdmin":        r.config.Web.WebAdmin,
		"WebWebmail":      r.config.Web.WebWebmail,
		"WebAPI":          r.config.Web.WebAPI,
		"Sitename":        r.config.Web.Sitename,
		"WebrootRedirect": r.config.WebrootRedirect,

		// Data paths (Linux-specific)
		"DataPath":      r.config.Paths.Data,
		"MailPath":      r.config.Paths.Mail,
		"DKIMPath":      r.config.Paths.DKIM,
		"QueuePath":     r.config.Paths.Queue,
		"FilterPath":    r.config.Paths.Filter,
		"CertsPath":     r.config.Paths.Certs,
		"OverridesPath": r.config.Paths.Overrides,

		// Service addresses
		"FrontAddress":    r.config.FrontAddress,
		"AdminAddress":    r.config.AdminAddress,
		"AntispamAddress": r.config.AntispamAddress,
		"WebmailAddress":  r.config.WebmailAddress,
		"WebdavAddress":   r.config.WebdavAddress,
		"RedisAddress":    r.config.RedisAddress,
		"Resolver":        r.config.Resolver,

		// Security keys
		"SecretKey":        r.config.SecretKey,
		"RoundcubeKey":     r.config.RoundcubeKey,
		"SnuffleupagusKey": r.config.SnuffleupagusKey,

		// Database
		"DBDsnw": r.config.Database.DBDsnw,

		// Webmail settings
		"Webmail":                  r.config.Webmail,
		"Plugins":                  r.config.Plugins,
		"Includes":                 r.config.Includes,
		"PermanentSessionLifetime": r.config.PermanentSessionLifetime,
		"FullTextSearch":           r.config.FullTextSearch,

		// Additional settings
		"Timezone":    r.config.Timezone,
		"MaxFilesize": r.config.MaxFilesize,

		// Port and protocol settings
		"Port80":           r.config.Port80,
		"ProxyProtocol25":  r.config.ProxyProtocol25,
		"ProxyProtocol80":  r.config.ProxyProtocol80,
		"ProxyProtocol443": r.config.ProxyProtocol443,

		// Feature flags
		"Admin":           r.config.Admin.Email != "",
		"API":             r.config.API,
		"EnableAntivirus": r.config.Services.Antivirus,
		// Enable webmail when both top-level webmail selection is not "none"
		// and Services.Webmail is set to a non-empty/non-"none" value.
		"EnableWebmail":   (r.config.Webmail != "none" && r.config.Services.Webmail != "" && r.config.Services.Webmail != "none"),
		"EnableFetchmail": r.config.Services.Fetchmail,
		"EnableWebdav":    r.config.Services.Webdav,
		"EnableOletools":  r.config.EnableOletools,
		"Webdav":          r.config.Services.Webdav,
	}
}

// getFuncMap returns custom template functions
func (r *Renderer) getFuncMap() template.FuncMap {
	return template.FuncMap{
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"join": func(elems []string, sep string) string {
			return strings.Join(elems, sep)
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"trim": func(s string) string {
			return strings.TrimSpace(s)
		},
		"default": func(value, defaultValue interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
		// Math functions
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		// Comparison functions (these are built-in but explicit for clarity)
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"le": func(a, b int) bool {
			return a <= b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"ge": func(a, b int) bool {
			return a >= b
		},
		// Index function for arrays
		"index": func(slice []string, i int) string {
			if i >= 0 && i < len(slice) {
				return slice[i]
			}
			return ""
		},
	}
}
