# MailStack

A complete mail server installer and management tool written in Go.

MailStack installs and configures all necessary components for a production-ready mail server on bare metal or VMs, without Docker.

## Components

- **Postfix** - SMTP mail server
- **Dovecot** - IMAP/POP3 server
- **Rspamd** - Anti-spam filtering
- **Nginx** - Web server and TLS termination
- **Redis** - Caching and session storage
- **Admin Interface** - Web-based management
- **Database** - SQLite/PostgreSQL/MySQL

## Installation

```bash
# Download MailStack
wget https://github.com/mailstack/mailstack/releases/latest/download/mailstack

# Make it executable
chmod +x mailstack

# Install mail server
sudo ./mailstack install --config=mailstack.json
```

## Configuration

Create a `mailstack.json` file:

```json
{
  "domain": "example.com",
  "hostname": "mail.example.com",
  "admin": {
    "email": "admin@example.com",
    "password": "changeme"
  },
  "database": {
    "type": "sqlite",
    "path": "/var/lib/mailstack/mailstack.db"
  },
  "tls": {
    "flavor": "letsencrypt",
    "email": "admin@example.com"
  }
}
```

## Usage

```bash
# Install mail server
mailstack install --config=mailstack.json

# Manage users
mailstack user add user@example.com
mailstack user delete user@example.com
mailstack user list

# Manage domains
mailstack domain add example.com
mailstack domain list

# Generate DKIM keys
mailstack dkim generate example.com

# Check service status
mailstack status

# Update components
mailstack update
```

## Features

- ✅ Single binary deployment
- ✅ Native system integration (systemd)
- ✅ Automatic TLS/SSL (Let's Encrypt)
- ✅ Web-based admin interface
- ✅ CLI management tools
- ✅ Database migrations
- ✅ Service health checks
- ✅ Configuration validation

## Requirements

- Linux (Debian/Ubuntu/RHEL/CentOS/Alpine)
- Root access
- 1GB+ RAM
- 10GB+ disk space

## License

MIT
