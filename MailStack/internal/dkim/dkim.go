package dkim

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Generate creates a new DKIM key pair for a domain
func Generate(domain, selector string, bits int, pathTemplate string) (string, string, error) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Determine key path
	keyPath := strings.ReplaceAll(pathTemplate, "{domain}", domain)
	keyPath = strings.ReplaceAll(keyPath, "{selector}", selector)

	// Create directory if needed
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write private key
	keyFile, err := os.Create(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	if err := pem.Encode(keyFile, privateKeyPEM); err != nil {
		return "", "", fmt.Errorf("failed to write key: %w", err)
	}

	// Set permissions
	if err := os.Chmod(keyPath, 0600); err != nil {
		return "", "", fmt.Errorf("failed to set permissions: %w", err)
	}

	// Generate public key for DNS
	publicKey := &privateKey.PublicKey
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	// Convert to DNS TXT record format
	dnsRecord := formatDNSRecord(string(pubKeyPEM))

	return keyPath, dnsRecord, nil
}

// GetDNSRecord reads an existing DKIM key and returns its DNS record
func GetDNSRecord(domain, selector, pathTemplate string) (string, error) {
	keyPath := strings.ReplaceAll(pathTemplate, "{domain}", domain)
	keyPath = strings.ReplaceAll(keyPath, "{selector}", selector)

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse private key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Extract public key
	publicKey := &privateKey.PublicKey
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return formatDNSRecord(string(pubKeyPEM)), nil
}

// formatDNSRecord converts a PEM public key to DKIM DNS TXT record format
func formatDNSRecord(pemKey string) string {
	// Remove PEM headers and newlines
	lines := strings.Split(pemKey, "\n")
	var keyData []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "-----") && line != "" {
			keyData = append(keyData, line)
		}
	}

	base64Key := strings.Join(keyData, "")

	// Format as DKIM record
	return fmt.Sprintf("v=DKIM1; k=rsa; p=%s", base64Key)
}

// Verify checks if a DKIM key exists for a domain
func Verify(domain, selector, pathTemplate string) (bool, error) {
	keyPath := strings.ReplaceAll(pathTemplate, "{domain}", domain)
	keyPath = strings.ReplaceAll(keyPath, "{selector}", selector)

	_, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
