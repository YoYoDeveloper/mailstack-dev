# MailStack Testing Guide

## Prerequisites

- Fresh Ubuntu 22.04 or Debian 11 server
- Root access
- Public IP address
- Domain name with DNS control

## Quick Test Installation

### 1. Transfer Binary

```bash
# From your Windows machine
scp mailstack root@your-server:/tmp/

# On server
chmod +x /tmp/mailstack
```

### 2. Create Minimal Config

```bash
cat > /tmp/config.json << 'EOF'
{
  "domain": "example.com",
  "hostname": "mail.example.com",
  "postmaster": "postmaster",
  "database": {
    "type": "sqlite"
  },
  "tls": {
    "flavor": "notls"
  },
  "mail": {
    "message_size_limit": 52428800,
    "default_quota": 1073741824,
    "dkim_selector": "dkim"
  },
  "services": {
    "antivirus": false,
    "webmail": "roundcube"
  },
  "paths": {
    "data": "/var/lib/mailstack",
    "mail": "/var/mail",
    "dkim": "/var/lib/mailstack/dkim",
    "queue": "/var/spool/postfix",
    "filter": "/var/lib/mailstack/filter",
    "certs": "/var/lib/mailstack/certs",
    "overrides": "/var/lib/mailstack/overrides"
  },
  "secret_key": "CHANGE-THIS-TO-RANDOM-STRING-HERE"
}
EOF
```

### 3. Run Installation

```bash
cd /tmp
sudo ./mailstack install --config config.json --verbose
```

**Expected Output:**
```
MailStack Installation
━━━━━━━━━━━━━━━━━━━━━

✓ Detecting operating system...
  OS: Ubuntu 22.04 (ubuntu)

✓ Validating configuration...
  Domain: example.com
  Hostname: mail.example.com

✓ Installing packages...
  Installing 30+ packages (this may take 5-10 minutes)...

✓ Creating system users...
  Creating user: mailu
  Creating user: postfix
  ... (7 users total)

✓ Setting up directories...
  Creating directory: /var/lib/mailstack
  ... (9 directories)

✓ Generating configuration files...
  Generating Postfix configuration...
  Generating Dovecot configuration...
  Generating Rspamd configuration...
  Generating Nginx configuration...
  ✓ All configuration files generated

✓ Initializing database...
  Setting up SQLite database...
  ✓ SQLite database initialized: /var/lib/mailstack/mailstack.db
  ✓ Default domain 'example.com' added to database

✓ Generating DKIM keys...
  Generating 2048-bit RSA key for example.com...
  ✓ DKIM keys generated for example.com
  Add this DNS TXT record to your domain:
  dkim._domainkey.example.com. IN TXT "v=DKIM1; k=rsa; p=..."

✓ Setting up TLS...
  TLS disabled, skipping certificate setup

✓ Configuring systemd services...
  Reloading systemd daemon...
  ✓ Systemd services configured

✓ Starting services...
  Enabling redis...
  Starting redis...
  ... (all services)
  ✓ All services started

✓ Creating admin user helper...
  ✓ Admin user creation script installed

✓ Running health check...
  ✓ redis: running
  ✓ rspamd: running
  ✓ postfix: running
  ✓ dovecot: running
  ✓ nginx: running
  ✓ All health checks passed!

━━━━━━━━━━━━━━━━━━━━━

✨ Installation completed successfully!
```

## Test Database Commands

### Check Status

```bash
./mailstack status --config /tmp/config.json
```

**Expected:**
```
📊 MailStack Service Status:
✅ postfix        active
✅ dovecot        active
✅ rspamd         active
✅ nginx          active
✅ redis          active
✅ php8.1-fpm     active
```

### Add a Domain

```bash
./mailstack domain add example.com --config /tmp/config.json
```

**Expected:**
```
✅ Domain example.com added successfully

📝 Don't forget to:
  1. Add DNS records (MX, SPF, DKIM, DMARC)
  2. Generate DKIM keys: mailstack dkim generate example.com
```

### List Domains

```bash
./mailstack domain list --config /tmp/config.json
```

**Expected:**
```
🌐 Mail Domains:
  - example.com (0 users)
```

### Add a User

```bash
./mailstack user add john@example.com --password secret123 --config /tmp/config.json
```

**Expected:**
```
✅ User john@example.com added successfully
```

### List Users

```bash
./mailstack user list --config /tmp/config.json
```

**Expected:**
```
📧 Mail Users:
  - john@example.com (quota: 1024 MB)
```

### Change Password

```bash
./mailstack user password john@example.com --password newpass456 --config /tmp/config.json
```

**Expected:**
```
✅ Password changed for john@example.com
```

### Generate DKIM

```bash
./mailstack dkim generate example.com --config /tmp/config.json
```

**Expected:**
```
✅ DKIM key generated successfully
📁 Key saved to: /var/lib/mailstack/dkim/example.com.dkim.key

📝 Add this TXT record to your DNS:
   dkim._domainkey.example.com IN TXT "v=DKIM1; k=rsa; p=MIIBIj..."
```

## Test Mail Sending

### Using Telnet

```bash
telnet localhost 25
```

```smtp
EHLO localhost
MAIL FROM: john@example.com
RCPT TO: test@external.com
DATA
Subject: Test Email

This is a test message.
.
QUIT
```

### Check Queue

```bash
postqueue -p
```

### Check Logs

```bash
journalctl -u postfix -f
journalctl -u dovecot -f
```

## Test IMAP Login

```bash
openssl s_client -connect localhost:993
```

```imap
A1 LOGIN john@example.com secret123
A2 LIST "" "*"
A3 SELECT INBOX
A4 LOGOUT
```

## Common Issues

### Services Won't Start

```bash
# Check service status
systemctl status postfix dovecot rspamd nginx redis

# Check logs
journalctl -xe

# Verify configs
postfix check
dovecot -n
```

### Database Errors

```bash
# Check database exists
ls -la /var/lib/mailstack/mailstack.db

# Check permissions
chown mailu:mailu /var/lib/mailstack/mailstack.db

# Query manually
sqlite3 /var/lib/mailstack/mailstack.db "SELECT * FROM users;"
```

### Port Conflicts

```bash
# Check what's listening
ss -tlnp | grep -E ':(25|587|143|993|80|443)'

# Kill conflicting services
systemctl stop apache2  # If nginx needs port 80
```

## Cleanup

```bash
# Stop all services
systemctl stop postfix dovecot rspamd nginx redis php8.1-fpm

# Remove database
rm /var/lib/mailstack/mailstack.db

# Remove configs
rm -rf /etc/postfix/* /etc/dovecot/* /etc/rspamd/* /etc/nginx/*

# Remove data
rm -rf /var/lib/mailstack
```

## Success Criteria

✅ All 13 installation steps complete without errors  
✅ All services show as "active" in status check  
✅ Can add domains and users via CLI  
✅ DKIM keys generated successfully  
✅ Can send test email via SMTP  
✅ Can login via IMAP  
✅ Webmail accessible (if enabled)  
✅ No errors in logs  

---

**If all tests pass, MailStack is production-ready! 🎉**
