#!/bin/bash

# Create certificates directory
mkdir -p certs

# Generate self-signed certificate + private key
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt \
    -days 365 -nodes \
    -subj "/C=CA/ST=Ontario/L=Toronto/O=LoadBalancer/CN=localhost"
