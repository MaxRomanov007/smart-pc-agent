#!/bin/sh
set -e

# определяем текущего пользователя (не root)
REAL_USER=${SUDO_USER:-$USER}
REAL_HOME=$(getent passwd "$REAL_USER" | cut -d: -f6)

# создаём директорию для user-сервисов
mkdir -p "$REAL_HOME/.config/systemd/user"

# копируем unit файл
cp /usr/share/smart-pc/smart-pc.service \
   "$REAL_HOME/.config/systemd/user/smart-pc.service"

# включаем и запускаем от имени пользователя
sudo -u "$REAL_USER" XDG_RUNTIME_DIR="/run/user/$(id -u $REAL_USER)" \
  systemctl --user enable --now smart-pc.service

echo "Smart PC Agent установлен и запущен"