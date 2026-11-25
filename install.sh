#!/bin/sh

# Variables
app_name=doppler
bin_path=/tmp/$app_name-6415415

# Check for root privileges
echo "Starting $app_name installer..."
if [ $USER != "root" ]; then
  echo "This installer must be run as root."
  exit 1
fi

# Build application
echo "Building application..."
mkdir -p bin
rm -rf bin/*
GOOS=linux GOARCH=amd64 go build -o $bin_path

# Install application
echo "Installing application..."
systemctl stop $app_name > /dev/null
cp $bin_path /usr/bin/$app_name
mkdir -p /etc/$app_name

# Configure application
echo "Configuring application..."
if [ ! -f /etc/$app_name/config.yml ]; then
  if [ ! -f config.yml ]; then
    cp config.yml.example config.yml
  fi
  
  cp config.yml /etc/$app_name
fi
cp config.yml /etc/$app_name &> /dev/null
cp config.yml.example /etc/$app_name &> /dev/null
cp proxy.pac /etc/$app_name
cp $app_name.service /etc/systemd/system

# TLS certificate: generated once, kept on reinstall.
cert_dir=/etc/$app_name/certs
if [ ! -f $cert_dir/tls.crt ] || [ ! -f $cert_dir/tls.key ]; then
  echo "Generating TLS certificate..."
  mkdir -p $cert_dir
  sh scripts/gen-certs.sh $cert_dir
fi

# Enable the TLS listener in the installed config, unless it is already set.
if ! grep -q '^tls_addr:' /etc/$app_name/config.yml; then
  printf '\n' >> /etc/$app_name/config.yml
  cat >> /etc/$app_name/config.yml <<EOF
tls_addr: ":5443"
tls_cert: "$cert_dir/tls.crt"
tls_key: "$cert_dir/tls.key"
EOF
fi

# Create system user
echo "Creating system user..."
adduser $app_name \
    --gecos "" \
    --system \
    --no-create-home \
    --disabled-password \
    --disabled-login > /dev/null

# Set permissions
chown -R $app_name: /etc/$app_name

# Start application
echo "Starting $app_name service..."
systemctl daemon-reload
systemctl start $app_name
systemctl status $app_name

