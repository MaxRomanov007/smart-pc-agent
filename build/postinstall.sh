#!/bin/sh
set -e

REAL_USER=${SUDO_USER:-$USER}
REAL_HOME=$(getent passwd "$REAL_USER" | cut -d: -f6)

mkdir -p "$REAL_HOME/.config/systemd/user"

cp /usr/share/smart-pc/smart-pc.service \
   "$REAL_HOME/.config/systemd/user/smart-pc.service"

sudo -u "$REAL_USER" XDG_RUNTIME_DIR="/run/user/$(id -u $REAL_USER)" \
  systemctl --user enable --now smart-pc.service

echo "Smart PC Agent installed and ran"