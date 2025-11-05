#!/bin/bash
# Quick test script for MailStack development

set -e

echo "=== MailStack Quick Test ==="
echo ""

# Check if running on Linux
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "❌ This script must be run on Linux"
    exit 1
fi

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "❌ Please run as root (use sudo)"
    exit 1
fi

# Build the binary
echo "📦 Building MailStack..."
cd "$(dirname "$0")"
go build -o mailstack ./cmd/mailstack

echo "✅ Build successful"
echo ""

# Test OS detection
echo "🔍 Testing OS detection..."
./mailstack config validate --config=configs/example.json || true

echo ""
echo "=== Test Complete ==="
echo ""
echo "To install MailStack:"
echo "  1. Create your config: cp configs/example.json mailstack.json"
echo "  2. Edit mailstack.json with your settings"
echo "  3. Run: sudo ./mailstack install --config=mailstack.json"
