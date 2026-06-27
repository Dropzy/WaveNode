#!/usr/bin/env sh
set -eu

APP_DIR="$HOME/wavenode"
REPO="https://github.com/Dropzy/WaveNode.git"
DOMAIN="wavenode.dropzy.co.uk"
ACME_EMAIL="admin@dropzy.co.uk"
MUSIC_PATH="/mnt/storage/media/music"

if [ ! -d "$APP_DIR/.git" ]; then
  git clone "$REPO" "$APP_DIR"
else
  git -C "$APP_DIR" fetch origin main
  git -C "$APP_DIR" checkout main
  git -C "$APP_DIR" pull --ff-only origin main
fi

cd "$APP_DIR"

random_base64() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 "$1" | tr -d '\n'
  else
    head -c "$1" /dev/urandom | base64 | tr -d '\n'
  fi
}

random_hex() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex "$1"
  else
    od -An -N "$1" -tx1 /dev/urandom | tr -d ' \n'
  fi
}

set_env_value() {
  key="$1"
  value="$2"
  if grep -q "^${key}=" .env; then
    sed -i "s|^${key}=.*|${key}=${value}|" .env
  else
    printf '%s=%s\n' "$key" "$value" >> .env
  fi
}

if [ ! -f .env ]; then
  umask 077
  {
    printf 'POSTGRES_PASSWORD=%s\n' "$(random_base64 36)"
    printf 'JWT_SECRET=%s\n' "$(random_base64 48)"
    setup_token="$(random_hex 16)"
    printf 'SETUP_TOKEN=%s\n' "$setup_token"
    printf 'MUSIC_PATH=%s\n' "$MUSIC_PATH"
    printf 'WAVENODE_BIND_ADDRESS=127.0.0.1\n'
    printf 'WAVENODE_PORT=8080\n'
    printf 'WAVENODE_VERSION=0.1.0\n'
    printf 'ALLOW_REGISTRATION=false\n'
    printf 'CORS_ALLOWED_ORIGINS=\n'
    printf 'FANART_TV_API_KEY=\n'
    printf 'ARTIST_METADATA_REFRESH_ENABLED=false\n'
    printf 'WAVENODE_DOMAIN=%s\n' "$DOMAIN"
    printf 'ACME_EMAIL=%s\n' "$ACME_EMAIL"
  } > .env
  printf '%s\n' "$setup_token" > SETUP_TOKEN.txt
  chmod 600 .env SETUP_TOKEN.txt
else
  cp .env ".env.backup.$(date +%Y%m%d%H%M%S)"
  chmod 600 .env
  set_env_value MUSIC_PATH "$MUSIC_PATH"
  set_env_value WAVENODE_BIND_ADDRESS "127.0.0.1"
  set_env_value WAVENODE_PORT "8080"
  set_env_value WAVENODE_VERSION "0.1.0"
  set_env_value ALLOW_REGISTRATION "false"
  set_env_value WAVENODE_DOMAIN "$DOMAIN"
  set_env_value ACME_EMAIL "$ACME_EMAIL"
  if ! grep -q '^SETUP_TOKEN=.' .env; then
    setup_token="$(random_hex 16)"
    set_env_value SETUP_TOKEN "$setup_token"
    printf '%s\n' "$setup_token" > SETUP_TOKEN.txt
    chmod 600 SETUP_TOKEN.txt
  fi
fi

docker compose -f docker-compose.yml -f docker-compose.internet.yml up -d --build

if command -v systemctl >/dev/null 2>&1; then
  sudo systemctl enable docker
fi

docker compose -f docker-compose.yml -f docker-compose.internet.yml ps
