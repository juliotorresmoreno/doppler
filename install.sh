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
if [ ! -f config.yml ]; then
  cp config.yml.example config.yml
fi
cp config.yml /etc/$app_name &> /dev/null
cp config.yml.example /etc/$app_name &> /dev/null
cp $app_name.service /etc/systemd/system

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

