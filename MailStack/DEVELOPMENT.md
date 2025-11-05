# MailStack Development Guide

## Building

```bash
# Download dependencies
go mod download
go mod tidy

# Build the binary
make build

# Or use go directly
go build -o bin/mailstack ./cmd/mailstack
```

## Testing on a Linux VM

### Prerequisites
- Linux VM (Ubuntu 22.04, Debian 12, or similar)
- Root access
- At least 2GB RAM, 10GB disk

### Quick Test

1. Create a test config:
```bash
cat > mailstack-test.json <<EOF
{
  "domain": "test.local",
  "hostname": "mail.test.local",
  "admin": {
    "email": "admin@test.local",
    "password": "Test123!@#"
  },
  "database": {
    "type": "sqlite",
    "path": "/var/lib/mailstack/mailstack.db"
  },
  "tls": {
    "flavor": "cert"
  }
}
EOF
```

2. Run installation:
```bash
sudo ./bin/mailstack install --config=mailstack-test.json -v
```

3. Check status:
```bash
sudo ./bin/mailstack status
```

## Development Workflow

### Phase 1 (Current) - Foundation ✅
- [x] OS detection
- [x] Package management
- [x] System user creation
- [x] Directory structure

### Phase 2 (Next) - Configuration
- [ ] Template engine for configs
- [ ] Postfix configuration
- [ ] Dovecot configuration
- [ ] Rspamd configuration
- [ ] Nginx configuration

### Phase 3 - Services
- [ ] Systemd service files
- [ ] Service management
- [ ] Health checks

### Phase 4 - Database & Users
- [ ] Database schema
- [ ] User/domain management
- [ ] DKIM key generation

### Phase 5 - TLS/Admin
- [ ] Let's Encrypt integration
- [ ] Admin web interface

## Testing Individual Components

### Test OS Detection
```go
package main

import (
    "fmt"
    "github.com/mailstack/mailstack/internal/osdetect"
)

func main() {
    info, err := osdetect.Detect()
    if err != nil {
        panic(err)
    }
    fmt.Printf("OS: %s\n", info.String())
    fmt.Printf("Type: %s\n", info.Type)
    fmt.Printf("Supported: %v\n", info.IsSupported())
}
```

### Test Package Installation
```bash
# Check if packages are installed
dpkg -l | grep postfix
dpkg -l | grep dovecot
dpkg -l | grep rspamd
```

## Project Structure

```
MailStack/
├── cmd/mailstack/          # Main CLI entry point
├── internal/
│   ├── osdetect/          # OS detection logic ✅
│   ├── packages/          # Package management ✅
│   ├── system/            # System operations (users, dirs, services) ✅
│   ├── installer/         # Main installation orchestration ✅
│   ├── config/            # Configuration management ✅
│   ├── templates/         # Config file templates (TODO)
│   ├── database/          # Database operations (TODO)
│   ├── services/          # Service management (TODO)
│   └── dkim/              # DKIM key management (Stub)
└── configs/               # Example configurations
```

## Next Steps

1. **Config Templating**: Port Mailu configs to Go templates
   - Copy configs from `mailu/core/*/conf/` 
   - Replace Jinja2 variables with Go template syntax
   - Create template renderer

2. **Database Schema**: Port Mailu's SQLAlchemy models
   - User table
   - Domain table  
   - Alias table
   - Token table

3. **Service Files**: Create systemd service definitions
   - mailstack-admin.service
   - Configure existing services (postfix, dovecot, etc.)

## Debugging

```bash
# Verbose installation
sudo ./bin/mailstack install -c config.json -v

# Check logs
sudo journalctl -u postfix -f
sudo journalctl -u dovecot -f
sudo journalctl -u rspamd -f

# Test mail delivery
echo "Test" | mail -s "Test" test@example.com
```

## Contributing

When implementing new features:
1. Keep Mailu's proven configurations
2. Only modify variable substitution
3. Test on multiple Linux distributions
4. Add error handling and logging
