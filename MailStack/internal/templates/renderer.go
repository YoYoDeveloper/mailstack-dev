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

//go:embed postfix/* dovecot/* rspamd/* nginx/*
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
		"Domain":              r.config.Domain,
		"Hostname":            r.config.Hostname,
		"Hostnames":           hostnames,
		"HostnamesStr":        strings.Join(hostnames, ","),
		"Postmaster":          r.config.Postmaster,
		"MessageSizeLimit":    r.config.Mail.MessageSizeLimit,
		"MessageRateLimit":    r.config.Mail.MessageRateLimit,
		"DefaultQuota":        r.config.Mail.DefaultQuota,
		"RecipientDelimiter":  r.config.Mail.RecipientDelimiter,
		"DKIMSelector":        r.config.Mail.DKIMSelector,
		"RelayHost":           r.config.Mail.RelayHost,
		"RelayUser":           r.config.Mail.RelayUser,
		"RelayPassword":       r.config.Mail.RelayPassword,
		"Subnet":              r.config.Network.Subnet,
		"Subnet6":             r.config.Network.Subnet6,
		"BindIPv4":            r.config.Network.BindIPv4,
		"BindIPv6":            r.config.Network.BindIPv6,
		"RelayNetworks":       r.config.Network.RelayNetworks,
		"TLSFlavor":           r.config.TLS.Flavor,
		"AdminPath":           r.config.Web.AdminPath,
		"WebmailPath":         r.config.Web.WebmailPath,
		"Sitename":            r.config.Web.Sitename,
		"DataPath":            r.config.Paths.Data,
		"MailPath":            r.config.Paths.Mail,
		"DKIMPath":            r.config.Paths.DKIM,
		"QueuePath":           r.config.Paths.Queue,
		"FilterPath":          r.config.Paths.Filter,
		"CertsPath":           r.config.Paths.Certs,
		"OverridesPath":       r.config.Paths.Overrides,
		"SecretKey":           r.config.SecretKey,
		"EnableAntivirus":     r.config.Services.Antivirus,
		"EnableWebmail":       r.config.Services.Webmail,
		"EnableFetchmail":     r.config.Services.Fetchmail,
		"EnableWebdav":        r.config.Services.Webdav,
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
		// Jinja2-like conditionals
		"default": func(value, defaultValue interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
	}
}
