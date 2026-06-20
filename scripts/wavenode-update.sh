#!/usr/bin/env sh
set -eu

INSTALL_DIR="${WAVENODE_INSTALL_DIR:-/srv/wavenode}"
COMPOSE_FILES="${WAVENODE_COMPOSE_FILES:-docker-compose.yml}"

cd "$INSTALL_DIR"

git fetch --tags origin
git pull --ff-only

set -- docker compose
OLD_IFS="$IFS"
IFS=,
for compose_file in $COMPOSE_FILES; do
  set -- "$@" -f "$compose_file"
done
IFS="$OLD_IFS"

"$@" pull
"$@" up -d --build
docker image prune -f
