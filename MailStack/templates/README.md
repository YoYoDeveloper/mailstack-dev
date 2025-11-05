# MailStack - Embedded Templates

All configuration files are embedded in the binary using Go's `embed` package.

## Structure

```
templates/
├── postfix/
│   ├── main.cf         ✅ Main Postfix configuration
│   └── master.cf       ✅ Postfix master process config
├── dovecot/
│   └── (TODO)
├── rspamd/
│   └── (TODO)
└── nginx/
    └── (TODO)
```

## How It Works

1. **Templates are embedded at compile time**:
```go
//go:embed postfix/* dovecot/* rspamd/* nginx/*
var templatesFS embed.FS
```

2. **Templates use Go's `text/template` syntax**:
```
mydomain = {{ .Domain }}
myhostname = {{ index (split .HostnamesStr ",") 0 }}
```

3. **Variables come from config.json**:
```json
{
  "domain": "example.com",
  "hostname": "mail.example.com"
}
```

## Converting from Jinja2

### Jinja2 → Go Template

| Jinja2 | Go Template |
|--------|-------------|
| `{{ DOMAIN }}` | `{{ .Domain }}` |
| `{{ HOSTNAMES.split(",")[0] }}` | `{{ index (split .HostnamesStr ",") 0 }}` |
| `{% if SUBNET6 %}...{% endif %}` | `{{if .Subnet6}}...{{end}}` |
| `{{ RELAYNETS.split(",") \| join(" ") }}` | `{{ replace .RelayNetworks "," " " }}` |
| `{{ VALUE\|default('dane') }}` | `{{ default .Value "dane" }}` |

## Available Template Functions

- `split` - Split string: `{{ split .Hostnames "," }}`
- `join` - Join strings: `{{ join .List "," }}`
- `contains` - Check substring: `{{ if contains .String "test" }}`
- `replace` - Replace string: `{{ replace .String "," " " }}`
- `lower` - Lowercase: `{{ lower .String }}`
- `upper` - Uppercase: `{{ upper .String }}`
- `trim` - Trim whitespace: `{{ trim .String }}`
- `default` - Default value: `{{ default .Value "fallback" }}`

## Adding New Templates

1. Copy config from `mailu/core/SERVICE/conf/`
2. Convert Jinja2 syntax to Go template
3. Save to `templates/SERVICE/filename`
4. Add to `generateConfigs()` in `installer.go`

The binary will automatically include it on next build!
