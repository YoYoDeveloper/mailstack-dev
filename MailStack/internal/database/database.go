package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"github.com/mailstack/mailstack/internal/config"
)

// DB represents a database connection
type DB struct {
	config config.DatabaseConfig
	conn   *sql.DB
}

// User represents a mail user
type User struct {
	Email   string
	Quota   int64
	Enabled bool
}

// Domain represents a mail domain
type Domain struct {
	Name      string
	UserCount int
}

// Connect establishes a database connection
func Connect(cfg config.DatabaseConfig) (*DB, error) {
	var dbPath string

	// Parse DSN to get database path
	if cfg.DSN != "" {
		// DSN format: "sqlite:/path/to/db"
		dsn := cfg.DSN
		if strings.HasPrefix(dsn, "sqlite:") {
			dbPath = strings.TrimPrefix(dsn, "sqlite:")
		} else {
			dbPath = dsn
		}
	} else if cfg.Path != "" {
		dbPath = cfg.Path
	} else {
		return nil, fmt.Errorf("no database path specified")
	}

	// Open SQLite connection
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		config: cfg,
		conn:   conn,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// AddUser adds a new mail user
func (db *DB) AddUser(email, password string, quota int64) error {
	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format: %s", email)
	}
	domain := parts[1]

	// Check if domain exists
	var domainExists bool
	err = db.conn.QueryRow("SELECT COUNT(*) > 0 FROM domains WHERE name = ?", domain).Scan(&domainExists)
	if err != nil {
		return fmt.Errorf("failed to check domain: %w", err)
	}
	if !domainExists {
		return fmt.Errorf("domain %s does not exist - add it first with 'mailstack domain add %s'", domain, domain)
	}

	// Insert user
	_, err = db.conn.Exec(`
		INSERT INTO users (email, password_hash, quota_bytes, enabled, global_admin)
		VALUES (?, ?, ?, 1, 0)
	`, email, string(hashedPassword), quota)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("user %s already exists", email)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// DeleteUser removes a mail user
func (db *DB) DeleteUser(email string, removeMailbox bool) error {
	// Check if user exists
	var exists bool
	err := db.conn.QueryRow("SELECT COUNT(*) > 0 FROM users WHERE email = ?", email).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}
	if !exists {
		return fmt.Errorf("user %s does not exist", email)
	}

	// Delete user from database
	_, err = db.conn.Exec("DELETE FROM users WHERE email = ?", email)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// TODO: If removeMailbox is true, delete /var/mail/<domain>/<user>
	if removeMailbox {
		// This would require os.RemoveAll() but we need to be careful
		// For now, just note that mailbox removal should be done manually
		fmt.Printf("Note: Mailbox data not removed. Manually delete if needed.\n")
	}

	return nil
}

// ListUsers returns all mail users
func (db *DB) ListUsers() ([]User, error) {
	rows, err := db.conn.Query(`
		SELECT email, quota_bytes, enabled 
		FROM users 
		ORDER BY email
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.Email, &user.Quota, &user.Enabled); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// ChangePassword changes a user's password
func (db *DB) ChangePassword(email, password string) error {
	// Check if user exists
	var exists bool
	err := db.conn.QueryRow("SELECT COUNT(*) > 0 FROM users WHERE email = ?", email).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}
	if !exists {
		return fmt.Errorf("user %s does not exist", email)
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	_, err = db.conn.Exec(`
		UPDATE users 
		SET password_hash = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE email = ?
	`, string(hashedPassword), email)

	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// AddDomain adds a new mail domain
func (db *DB) AddDomain(domain string) error {
	// Validate domain format (basic check)
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("invalid domain format: %s", domain)
	}

	// Insert domain
	_, err := db.conn.Exec(`
		INSERT INTO domains (name, enabled)
		VALUES (?, 1)
	`, domain)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("domain %s already exists", domain)
		}
		return fmt.Errorf("failed to create domain: %w", err)
	}

	return nil
}

// DeleteDomain removes a mail domain
func (db *DB) DeleteDomain(domain string) error {
	// Check if domain exists
	var exists bool
	err := db.conn.QueryRow("SELECT COUNT(*) > 0 FROM domains WHERE name = ?", domain).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check domain: %w", err)
	}
	if !exists {
		return fmt.Errorf("domain %s does not exist", domain)
	}

	// Check if domain has users
	var userCount int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM users WHERE email LIKE ?
	`, "%@"+domain).Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to check users: %w", err)
	}
	if userCount > 0 {
		return fmt.Errorf("cannot delete domain %s: it has %d users (delete users first)", domain, userCount)
	}

	// Delete domain
	_, err = db.conn.Exec("DELETE FROM domains WHERE name = ?", domain)
	if err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}

	return nil
}

// ListDomains returns all mail domains
func (db *DB) ListDomains() ([]Domain, error) {
	rows, err := db.conn.Query(`
		SELECT d.name, COUNT(u.id) as user_count
		FROM domains d
		LEFT JOIN users u ON u.email LIKE '%@' || d.name
		GROUP BY d.name
		ORDER BY d.name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query domains: %w", err)
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var domain Domain
		if err := rows.Scan(&domain.Name, &domain.UserCount); err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, domain)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating domains: %w", err)
	}

	return domains, nil
}

// InitSchema initializes the database schema
func (db *DB) InitSchema() error {
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

	// Execute schema creation
	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// Alias represents an email alias
type Alias struct {
	Email       string
	Destination string
	Enabled     bool
}

// AddAlias creates a new email alias
func (db *DB) AddAlias(email, destination string) error {
	// Validate email format
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format: %s", email)
	}
	domain := parts[1]

	// Check if domain exists
	var domainExists bool
	err := db.conn.QueryRow("SELECT COUNT(*) > 0 FROM domains WHERE name = ?", domain).Scan(&domainExists)
	if err != nil {
		return fmt.Errorf("failed to check domain: %w", err)
	}
	if !domainExists {
		return fmt.Errorf("domain %s does not exist - add it first with 'mailstack domain add %s'", domain, domain)
	}

	// Check if alias already exists
	var exists bool
	err = db.conn.QueryRow("SELECT COUNT(*) > 0 FROM aliases WHERE email = ?", email).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check alias: %w", err)
	}
	if exists {
		return fmt.Errorf("alias %s already exists", email)
	}

	// Check if it conflicts with an actual user
	var userExists bool
	err = db.conn.QueryRow("SELECT COUNT(*) > 0 FROM users WHERE email = ?", email).Scan(&userExists)
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}
	if userExists {
		return fmt.Errorf("cannot create alias: %s is already a real user", email)
	}

	// Validate destination addresses
	destinations := strings.Split(destination, ",")
	for i, dest := range destinations {
		destinations[i] = strings.TrimSpace(dest)
		if destinations[i] == "" {
			return fmt.Errorf("empty destination address")
		}
		// Basic validation
		if !strings.Contains(destinations[i], "@") {
			return fmt.Errorf("invalid destination address: %s", destinations[i])
		}
	}

	// Insert alias
	_, err = db.conn.Exec(`
		INSERT INTO aliases (email, destination, enabled)
		VALUES (?, ?, 1)
	`, email, destination)

	if err != nil {
		return fmt.Errorf("failed to create alias: %w", err)
	}

	return nil
}

// DeleteAlias removes an email alias
func (db *DB) DeleteAlias(email string) error {
	// Check if alias exists
	var exists bool
	err := db.conn.QueryRow("SELECT COUNT(*) > 0 FROM aliases WHERE email = ?", email).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check alias: %w", err)
	}
	if !exists {
		return fmt.Errorf("alias %s does not exist", email)
	}

	// Delete alias
	_, err = db.conn.Exec("DELETE FROM aliases WHERE email = ?", email)
	if err != nil {
		return fmt.Errorf("failed to delete alias: %w", err)
	}

	return nil
}

// ListAliases returns all email aliases
func (db *DB) ListAliases() ([]Alias, error) {
	rows, err := db.conn.Query(`
		SELECT email, destination, enabled 
		FROM aliases 
		ORDER BY email
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query aliases: %w", err)
	}
	defer rows.Close()

	var aliases []Alias
	for rows.Next() {
		var alias Alias
		if err := rows.Scan(&alias.Email, &alias.Destination, &alias.Enabled); err != nil {
			return nil, fmt.Errorf("failed to scan alias: %w", err)
		}
		aliases = append(aliases, alias)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aliases: %w", err)
	}

	return aliases, nil
}

// GetAlias returns details for a specific alias
func (db *DB) GetAlias(email string) (*Alias, error) {
	var alias Alias
	err := db.conn.QueryRow(`
		SELECT email, destination, enabled 
		FROM aliases 
		WHERE email = ?
	`, email).Scan(&alias.Email, &alias.Destination, &alias.Enabled)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("alias %s not found", email)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alias: %w", err)
	}

	return &alias, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	// Check current schema version
	var version int
	err := db.conn.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	// Currently at version 0 (initial schema)
	// Future migrations would go here
	if version < 1 {
		// Migration example (none needed yet):
		// _, err := db.conn.Exec("ALTER TABLE users ADD COLUMN new_field TEXT")
		// if err != nil {
		//     return err
		// }
		// _, err = db.conn.Exec("PRAGMA user_version = 1")
	}

	return nil
}
