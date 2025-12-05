#!/bin/bash

# Create certificates directory
mkdir -p certs

# Generate private key and self-signed certificate
# For development/testing purposes
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt \
    -days 365 -nodes \
    -subj "/C=CA/ST=Ontario/L=Toronto/O=LoadBalancer/CN=localhost"

echo "✅ Generated self-signed certificate:"
echo "   - Certificate: certs/server.crt"
echo "   - Private Key: certs/server.key"
echo ""
echo "⚠️  Note: This is a self-signed certificate for development only."
echo "   For production, use certificates from a trusted CA (Let's Encrypt, etc.)"
