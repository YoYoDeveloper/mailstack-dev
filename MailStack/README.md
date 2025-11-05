# MailStack - Complete Mail Server Installer# MailStack



MailStack is a single-binary installer that deploys a complete, production-ready mail server on Linux. It's a Go-based reimplementation of Mailu that runs natively on your system without Docker containers.A complete mail server installer and management tool written in Go.



## 🎯 Status: ~85% CompleteMailStack installs and configures all necessary components for a production-ready mail server on bare metal or VMs, without Docker.



**Working:**## Components

✅ OS detection (6 distros)  

✅ Package installation  - **Postfix** - SMTP mail server

✅ Configuration generation (48 files)  - **Dovecot** - IMAP/POP3 server

✅ Systemd service management  - **Rspamd** - Anti-spam filtering

✅ DKIM key generation  - **Nginx** - Web server and TLS termination

✅ TLS/Let's Encrypt setup  - **Redis** - Caching and session storage

✅ Database initialization (SQLite)  - **Admin Interface** - Web-based management

✅ Health checks  - **Database** - SQLite/PostgreSQL/MySQL



**TODO:**## Installation

⏳ Admin user creation (database integration)  

⏳ MySQL/PostgreSQL support  ```bash

⏳ Web admin panel  # Download MailStack

⏳ Testing on live systems  wget https://github.com/mailstack/mailstack/releases/latest/download/mailstack



---# Make it executable

chmod +x mailstack

## Features

# Install mail server

### Complete Mail Server Stacksudo ./mailstack install --config=mailstack.json

- **Postfix** - SMTP server for sending/receiving mail```

- **Dovecot** - IMAP/POP3/LMTP with Sieve filtering

- **Rspamd** - Advanced anti-spam with learning## Configuration

- **Redis** - High-performance caching

- **Nginx** - Web proxy & TLS terminationCreate a `mailstack.json` file:



### Optional Components```json

- **ClamAV** - Antivirus scanning{

- **Roundcube** or **SnappyMail** - Modern webmail  "domain": "example.com",

- **PHP 8.1+** with FPM for webmail  "hostname": "mail.example.com",

  "admin": {

### Database Support    "email": "admin@example.com",

- **SQLite** - Zero-config, perfect for small/medium deployments    "password": "changeme"

- **MySQL/MariaDB** - Coming soon  },

- **PostgreSQL** - Coming soon  "database": {

    "type": "sqlite",

### TLS/SSL    "path": "/var/lib/mailstack/mailstack.db"

- **Let's Encrypt** - Automatic free certificates with auto-renewal  },

- **Custom certificates** - Bring your own certs  "tls": {

- **Mail-specific TLS** - Separate certs for mail vs web    "flavor": "letsencrypt",

    "email": "admin@example.com"

### Multi-OS Support  }

- Debian 11+}

- Ubuntu 20.04+```

- RHEL 8+ / CentOS 8+ / Fedora 35+

- Alpine Linux 3.16+## Usage



---```bash

# Install mail server

## Quick Startmailstack install --config=mailstack.json



### Prerequisites# Manage users

- Linux server with root accessmailstack user add user@example.com

- Public IP addressmailstack user delete user@example.com

- Domain with DNS accessmailstack user list

- Ports 25, 587, 143, 993, 80, 443 open

# Manage domains

### 1. Downloadmailstack domain add example.com

mailstack domain list

```bash

# Download the latest release# Generate DKIM keys

wget https://github.com/yourusername/mailstack/releases/latest/download/mailstackmailstack dkim generate example.com

chmod +x mailstack

```# Check service status

mailstack status

### 2. Configure

# Update components

```bashmailstack update

# Generate example config```

./mailstack config generate > config.json

## Features

# Edit with your settings

nano config.json- ✅ Single binary deployment

```- ✅ Native system integration (systemd)

- ✅ Automatic TLS/SSL (Let's Encrypt)

**Required settings:**- ✅ Web-based admin interface

```json- ✅ CLI management tools

{- ✅ Database migrations

  "domain": "example.com",- ✅ Service health checks

  "hostname": "mail.example.com",- ✅ Configuration validation

  "tls": {

    "flavor": "letsencrypt",## Requirements

    "email": "admin@example.com"

  },- Linux (Debian/Ubuntu/RHEL/CentOS/Alpine)

  "secret_key": "CHANGE-ME-TO-RANDOM-32-CHAR-STRING"- Root access

}- 1GB+ RAM

```- 10GB+ disk space



### 3. Install## License



```bashMIT

# Run installer (requires root)
sudo ./mailstack install --config config.json --verbose
```

Installation takes 5-10 minutes and includes:
1. OS detection
2. Package installation (~30+ packages)
3. System user creation
4. Directory structure setup
5. Configuration generation (48 files)
6. Database initialization
7. DKIM key generation
8. TLS certificate setup
9. Service configuration
10. Service startup
11. Health verification

### 4. Configure DNS

Add these DNS records:

```dns
# Required - MX record for receiving mail
example.com.              IN MX  10 mail.example.com.

# Required - A record pointing to your server
mail.example.com.         IN A      1.2.3.4

# Recommended - SPF for sender authentication
example.com.              IN TXT    "v=spf1 mx ~all"

# Recommended - DKIM (key shown after installation)
dkim._domainkey.example.com. IN TXT "v=DKIM1; k=rsa; p=YOUR_KEY_HERE"

# Recommended - DMARC for email security
_dmarc.example.com.       IN TXT    "v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com"

# Optional - Webmail subdomain
webmail.example.com.      IN A      1.2.3.4
```

### 5. Create Admin User

```bash
# Use the helper script created during installation
sudo mailstack-create-admin admin@example.com YourSecurePassword123
```

### 6. Access

- **Webmail**: https://webmail.example.com
- **IMAP**: mail.example.com:993 (SSL)
- **SMTP**: mail.example.com:587 (STARTTLS)

---

## Commands

```bash
# Installation
mailstack install --config config.json --verbose

# Configuration management
mailstack config generate              # Create example config
mailstack config validate --config FILE  # Validate config file
mailstack config show --config FILE      # Display current config

# Version info
mailstack version
```

---

## Configuration

See `config.example.json` for all options. Key sections:

### Basic Settings
```json
{
  "domain": "example.com",
  "hostname": "mail.example.com",
  "postmaster": "postmaster",
  "subnet": "192.168.1.0/24"
}
```

### TLS Configuration
```json
{
  "tls": {
    "flavor": "letsencrypt",
    "email": "admin@example.com"
  }
}
```

**TLS Flavors:**
- `letsencrypt` - Auto certificates for web + mail
- `mail-letsencrypt` - Auto certificates for mail only
- `cert` - Custom certificates (provide paths)
- `notls` - Disable TLS (not recommended!)

### Database Configuration
```json
{
  "database": {
    "type": "sqlite",
    "path": "/var/lib/mailstack/mailstack.db"
  }
}
```

### Mail Settings
```json
{
  "mail": {
    "message_size_limit": 52428800,
    "message_ratelimit": "200/day",
    "default_quota": 1073741824,
    "dkim_selector": "dkim"
  }
}
```

### Services
```json
{
  "services": {
    "antivirus": true,
    "webmail": "roundcube"
  }
}
```

**Webmail options:** `roundcube`, `snappymail`, `none`

---

## Architecture

```
         ┌──────────────────────┐
         │   Internet Traffic   │
         └──────────┬───────────┘
                    │
             ┌──────▼──────┐
             │    Nginx    │  Ports 80/443
             │  TLS Proxy  │
             └──┬────────┬─┘
                │        │
    ┌───────────▼──┐  ┌──▼─────────┐
    │   Postfix    │  │  Webmail   │
    │    (SMTP)    │  │   (PHP)    │
    │  Port 25/587 │  └────────────┘
    └──────┬───────┘
           │
    ┌──────▼────────┐
    │    Rspamd     │◄───┐
    │  (Anti-spam)  │    │  Redis
    └──────┬────────┘    │  (Cache)
           │             │
    ┌──────▼────────┐    │
    │   Dovecot     │────┘
    │  (IMAP/Sieve) │
    │  Port 143/993 │
    └──────┬────────┘
           │
    ┌──────▼────────┐
    │ Mail Storage  │
    │   /var/mail   │
    └───────────────┘
```

---

## File Locations

```
/etc/postfix/          - Postfix configuration
/etc/dovecot/          - Dovecot configuration  
/etc/rspamd/           - Rspamd configuration
/etc/nginx/            - Nginx configuration
/var/lib/mailstack/    - MailStack data
  ├── mailstack.db     - Database (SQLite)
  ├── dkim/            - DKIM keys
  ├── certs/           - TLS certificates
  └── overrides/       - Custom overrides
/var/mail/             - Mail storage
/var/log/mailstack/    - Logs
```

---

## Troubleshooting

### Check Services
```bash
sudo systemctl status postfix dovecot rspamd nginx redis
```

### View Logs
```bash
sudo journalctl -u postfix -f
sudo journalctl -u dovecot -f
sudo journalctl -u rspamd -f
```

### Test Mail Flow
```bash
# Check mail queue
sudo postqueue -p

# Test SMTP
telnet mail.example.com 25

# Test IMAP
openssl s_client -connect mail.example.com:993
```

### Common Issues

**❌ Port 25 blocked**
- Most ISPs block port 25
- Contact hosting provider to unblock
- Use port 587 for sending

**❌ Let's Encrypt fails**
- Check DNS: `dig mail.example.com`
- Verify ports 80/443 open: `sudo ufw status`
- Ensure domain points to server IP

**❌ Services won't start**
- Check logs: `sudo journalctl -xe`
- Verify config: `sudo postfix check`
- Check disk space: `df -h`

---

## System Requirements

### Minimum
- **OS**: Linux (see supported distros)
- **RAM**: 2 GB
- **Disk**: 10 GB
- **CPU**: 1 core
- **Network**: Public IP, ports 25/587/143/993/80/443

### Recommended
- **RAM**: 4+ GB
- **Disk**: 20+ GB (SSD preferred)
- **CPU**: 2+ cores
- **Backup**: Regular backups of /var/lib/mailstack and /var/mail

---

## Security

### Best Practices
1. ✅ Use strong passwords (12+ characters)
2. ✅ Enable fail2ban for brute-force protection
3. ✅ Keep system updated: `apt update && apt upgrade`
4. ✅ Configure SPF/DKIM/DMARC
5. ✅ Use TLS for all connections
6. ✅ Monitor logs regularly
7. ✅ Backup database and mail daily
8. ✅ Configure reverse DNS (PTR record)

### Firewall Setup
```bash
# UFW (Ubuntu/Debian)
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 25/tcp    # SMTP
sudo ufw allow 587/tcp   # Submission
sudo ufw allow 143/tcp   # IMAP
sudo ufw allow 993/tcp   # IMAPS
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
```

---

## Building from Source

```bash
# Clone repository
git clone https://github.com/yourusername/mailstack.git
cd mailstack

# Install dependencies
go mod download

# Build for Linux (from any OS)
GOOS=linux GOARCH=amd64 go build -o mailstack .

# Build for current OS
go build -o mailstack .
```

Binary size: ~7.9 MB (includes all 48 config templates!)

---

## Project Structure

```
MailStack/
├── main.go                    # Entry point
├── internal/
│   ├── cli/                   # Cobra commands
│   ├── config/                # Configuration structs
│   ├── installer/             # Installation orchestration
│   ├── osdetect/              # OS detection
│   ├── packages/              # Package management
│   ├── system/                # System operations
│   └── templates/             # Template renderer
│       └── templates/         # Embedded configs (48 files)
│           ├── postfix/
│           ├── dovecot/
│           ├── rspamd/
│           ├── nginx/
│           └── webmails/
├── config.example.json        # Example configuration
└── README.md                  # This file
```

---

## Credits

- Based on [Mailu](https://mailu.io/) architecture
- Uses [Cobra](https://github.com/spf13/cobra) for CLI
- Built with ❤️ in Go

---

## License

MIT License - See LICENSE file

---

## Roadmap

### Phase 1: Core Installation ✅ (Current)
- [x] OS detection
- [x] Package installation
- [x] Configuration generation
- [x] Service management
- [x] DKIM generation
- [x] TLS setup
- [x] Database init (SQLite)
- [x] Health checks

### Phase 2: Database & Users ⏳
- [ ] Admin user creation
- [ ] MySQL support
- [ ] PostgreSQL support
- [ ] User management CLI
- [ ] Domain management CLI

### Phase 3: Web Interface
- [ ] Web admin panel
- [ ] User self-service portal
- [ ] Statistics dashboard
- [ ] Log viewer

### Phase 4: Advanced Features
- [ ] Backup/restore automation
- [ ] Migration tools (from Mailu, iRedMail, etc.)
- [ ] Monitoring & alerting
- [ ] Email quarantine interface
- [ ] Multiple server support (clustering)

---

## Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/mailstack/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/mailstack/discussions)
- **Email**: support@example.com

---

**Built for production. Designed for simplicity. Powered by Go.**
