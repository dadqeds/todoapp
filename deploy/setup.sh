#!/usr/bin/env bash
# Первичная настройка чистого сервера Ubuntu под todoapp. Запускать от root:
#   curl -fsSL https://raw.githubusercontent.com/dadqeds/todoapp/main/deploy/setup.sh | bash
# Скрипт можно запускать повторно: уже сделанные шаги пропускаются.
set -euo pipefail

APP_DIR=/opt/todoapp
REPO_URL=https://github.com/dadqeds/todoapp.git

step() { printf '\n==> %s\n' "$1"; }

step "Файл подкачки 2 ГБ (сборке Go не хватает памяти без него)"
if ! swapon --show | grep -q .; then
	fallocate -l 2G /swapfile
	chmod 600 /swapfile
	mkswap /swapfile
	swapon /swapfile
	echo '/swapfile none swap sw 0 0' >> /etc/fstab
else
	echo "уже есть"
fi

step "Пакеты"
export DEBIAN_FRONTEND=noninteractive
apt-get update -q
apt-get install -y -q git make ufw cron unattended-upgrades
if ! command -v docker >/dev/null; then
	apt-get install -y -q docker.io docker-compose-v2
fi
docker compose version

step "Ротация логов Docker"
if [ ! -f /etc/docker/daemon.json ]; then
	cat > /etc/docker/daemon.json <<'EOF'
{ "log-driver": "json-file", "log-opts": { "max-size": "10m", "max-file": "3" } }
EOF
	systemctl restart docker
else
	echo "/etc/docker/daemon.json уже есть, не трогаю"
fi
systemctl enable --now docker cron

step "Файрвол: открыты только SSH, 80 и 443"
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443
ufw --force enable

step "Вход по SSH только по ключу"
if [ -s /root/.ssh/authorized_keys ]; then
	echo 'PasswordAuthentication no' > /etc/ssh/sshd_config.d/10-todoapp.conf
	systemctl reload ssh
	echo "пароль отключён"
else
	echo "в /root/.ssh/authorized_keys нет ключей — вход по паролю оставлен"
fi

step "Код приложения в $APP_DIR"
if [ ! -d "$APP_DIR/.git" ]; then
	git clone "$REPO_URL" "$APP_DIR"
else
	git -C "$APP_DIR" pull --ff-only
fi

step "Ежедневная копия базы (03:00) и чистка логов старше 30 дней"
cat > /etc/cron.d/todoapp <<EOF
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
0 3 * * * root cd $APP_DIR && mkdir -p out/backups && make db-backup >> out/backups/cron.log 2>&1
30 3 * * * root find $APP_DIR/out/logs -name '*.log' -mtime +30 -delete
EOF

step "Готово"
if [ ! -f "$APP_DIR/.env" ]; then
	echo "Дальше: cp $APP_DIR/.env.example $APP_DIR/.env, заполнить его и выполнить make server-up (см. README, раздел «Сервер»)."
fi
