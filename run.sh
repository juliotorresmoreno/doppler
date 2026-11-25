#!/bin/sh
# Builds Doppler and runs it with the TLS listener enabled.
#
#   ./run.sh
#
# - Generates a self-signed certificate in certs/ if it does not exist.
# - Creates certs/config.yml from config.yml.example if it does not exist,
#   with tls_addr/tls_cert/tls_key pointing at the generated files.
# - Plain HTTP listens on :5080 and TLS on :5443 (edit certs/config.yml to change).
set -e

cd "$(dirname "$0")"
root="$(pwd)"
cert_dir="certs"
conf="$cert_dir/config.yml"

if [ ! -f "$cert_dir/tls.crt" ] || [ ! -f "$cert_dir/tls.key" ]; then
  sh scripts/gen-certs.sh "$cert_dir"
fi

# proxy.pac is served by Doppler at /proxy.pac, from the config directory.
cp proxy.pac "$cert_dir/proxy.pac"

if [ ! -f "$conf" ]; then
  mkdir -p "$cert_dir"
  sed \
    -e "s|^addr:.*|addr: \":5080\"|" \
    -e "s|^tls_addr:.*|tls_addr: \":5443\"|" \
    -e "s|^tls_cert:.*|tls_cert: \"$root/$cert_dir/tls.crt\"|" \
    -e "s|^tls_key:.*|tls_key: \"$root/$cert_dir/tls.key\"|" \
    config.yml.example > "$conf"
  echo "Created $conf"
fi

echo "Building doppler..."
mkdir -p dist
go build -o dist/doppler .

echo "Starting doppler (HTTP :5080, TLS :5443)..."
exec ./dist/doppler -c "$conf"
