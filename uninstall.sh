#!/bin/sh

# Removes the doppler service, binary and systemd unit installed by install.sh.
# /etc/doppler (config, certificates, proxy.pac) is kept unless --purge is given.
#
#   sudo ./uninstall.sh          # keep /etc/doppler
#   sudo ./uninstall.sh --purge  # also remove /etc/doppler and the doppler user

app_name=doppler
purge=0
[ "$1" = "--purge" ] && purge=1

echo "Starting $app_name uninstaller..."
if [ "$(id -u)" -ne 0 ]; then
  echo "This uninstaller must be run as root."
  exit 1
fi

echo "Stopping $app_name service..."
systemctl stop $app_name > /dev/null 2>&1
systemctl disable $app_name > /dev/null 2>&1
rm -f /etc/systemd/system/$app_name.service
systemctl daemon-reload
systemctl reset-failed $app_name > /dev/null 2>&1

echo "Removing binary..."
rm -f /usr/bin/$app_name

if [ $purge -eq 1 ]; then
  echo "Removing /etc/$app_name and system user..."
  rm -rf /etc/$app_name
  deluser --system $app_name > /dev/null 2>&1 || userdel $app_name > /dev/null 2>&1
else
  echo "Kept /etc/$app_name (config, certificates). Run with --purge to remove it."
fi

echo "$app_name uninstalled."
