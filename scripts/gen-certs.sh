#!/bin/sh
# Generates a self-signed TLS certificate and key for local use.
# Usage: sh scripts/gen-certs.sh [cert_dir]
set -e

cert_dir="${1:-certs}"
mkdir -p "$cert_dir"

openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout "$cert_dir/tls.key" \
  -out "$cert_dir/tls.crt" \
  -days 365 \
  -subj "/O=Doppler/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
  -addext "basicConstraints=critical,CA:FALSE" \
  -addext "keyUsage=critical,digitalSignature,keyEncipherment" \
  -addext "extendedKeyUsage=serverAuth" 2>/dev/null

chmod 600 "$cert_dir/tls.key"
echo "Generated $cert_dir/tls.crt and $cert_dir/tls.key (valid 365 days)"
