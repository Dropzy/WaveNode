# Deployment

## Docker Compose

1. Copy `.env.example` to `.env`.
2. Set strong values for `POSTGRES_PASSWORD` and `JWT_SECRET`.
3. Set `MUSIC_PATH` to an absolute folder on the Docker host.
4. Run `docker compose up -d`.
5. Open `http://server-address:8080`.
6. In first-run setup, select `/music` and `/data/artwork`.

The host music folder is mounted read-only. Database and extracted artwork use Docker volumes.

The standard deployment binds port `8080` to `127.0.0.1`, so it is available only on the Docker host. Use an SSH tunnel, a private VPN, or the internet deployment below for remote access.

## Internet Deployment

Requirements:

- A domain or subdomain you control, such as `music.example.com`
- Public DNS `A` and/or `AAAA` records pointing to the server
- TCP ports `80` and `443` forwarded to the server
- UDP port `443` forwarded if you want HTTP/3
- A public IP address; users behind carrier-grade NAT need a VPN or tunnel instead

Add these values to `.env`:

```env
WAVENODE_DOMAIN=music.example.com
ACME_EMAIL=admin@example.com
SETUP_TOKEN=replace-with-a-long-random-first-run-code
```

Use a randomly generated setup code of at least 16 characters. This prevents an outside visitor from claiming the first administrator account.

Start the internet deployment:

```bash
docker compose -f docker-compose.yml -f docker-compose.internet.yml up -d
```

Open `https://music.example.com`. Caddy obtains and renews the certificate automatically. Enter `SETUP_TOKEN` when the first-run wizard requests the setup access code.

Only ports `80` and `443` should be publicly reachable. Do not forward `8080` or PostgreSQL port `5432`. WaveNode keeps the backend and database on the private Docker network, redirects HTTP to HTTPS, enables HSTS, supports WebSockets and streaming, and keeps authenticated media URLs out of the bundled proxy access logs.

To stop this deployment:

```bash
docker compose -f docker-compose.yml -f docker-compose.internet.yml down
```

For a custom reverse proxy:

- Terminate TLS and redirect HTTP to HTTPS
- Proxy normal traffic to `127.0.0.1:8080`
- Support WebSocket upgrades on `/ws`
- Disable query-string access logging for `/api/music/*/stream` and `/ws`
- Allow long-lived and range-based audio responses

When the included frontend and API share one hostname, leave `CORS_ALLOWED_ORIGINS` blank. Add explicit HTTPS origins only when a separate frontend hostname calls the API.

## Accounts

Public registration defaults to disabled. Create accounts from **Admin Dashboard > Users**.

To temporarily enable public registration:

```env
ALLOW_REGISTRATION=true
```

Restart the backend after changing it:

```bash
docker compose up -d
```

## Development Builds

The default Compose file uses published GHCR images. Contributors who want to build from the local source tree should use the development override:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build
```

## Upgrade

Download a backup from **Admin Dashboard > Library jobs > Backup and restore** before upgrading. It contains WaveNode accounts, playlists, settings, indexed data, listening history, and stored artwork. Original audio files remain separate.

WaveNode can check GitHub releases from **Admin Dashboard > System > WaveNode updates**. The **Update now** button is disabled until the server owner configures a trusted server-side update command.

The default Docker stack keeps the main app container restricted. That is deliberate. One-click Docker updates are handled by an optional updater sidecar. Only the updater sidecar receives Docker socket access; the web app and API do not.

To enable one-click Docker updates, add these values to `.env`:

```env
WAVENODE_UPDATE_REPOSITORY=Dropzy/WaveNode
WAVENODE_UPDATER_URL=http://updater:8090
WAVENODE_UPDATER_TOKEN=replace-with-a-long-random-token
WAVENODE_UPDATER_COMPOSE_FILES=/compose/docker-compose.yml
WAVENODE_UPDATE_TIMEOUT_SECONDS=900
COMPOSE_PROFILES=updater
```

Then recreate the stack:

```bash
docker compose up -d
```

For the internet deployment, use the same `.env` values and run:

```bash
docker compose -f docker-compose.yml -f docker-compose.internet.yml up -d
```

The updater sidecar runs `docker compose pull` and `docker compose up -d` for the WaveNode backend and frontend services. It uses the same `.env` and Compose file mounted read-only from the host. Only administrators can trigger updates from the dashboard.

Do not expose the updater port publicly. It should only be reachable on the private Docker network.

Native/source-checkout deployments can still use `WAVENODE_UPDATE_COMMAND` instead of the updater sidecar, but Docker installs should prefer the sidecar.

Manual upgrade remains:

```bash
docker compose pull
docker compose up -d
```

Database migrations run when the backend starts. Check status with:

```bash
docker compose ps
docker compose logs --tail=200 backend
```

Open **Admin Dashboard > System** after the upgrade and confirm the expected version, available source folders, and zero missing files.

## Backup

The recommended method is **Admin Dashboard > Library jobs > Backup and restore > Download backup**. Store the downloaded zip outside the WaveNode host.

The commands below are an alternative full PostgreSQL backup for administrators:
Create a backup directory:

```bash
mkdir -p backup
```

Database:

```bash
docker compose exec -T database pg_dump -U wavenode -d wavenode > backup/wavenode.sql
```

Artwork:

```bash
docker compose cp backend:/data/artwork backup/artwork
```

Also back up the original music collection separately. WaveNode never treats extracted artwork as a replacement for the source audio files.

Test restores regularly. An untested backup should not be treated as recoverable.

## Restore

For a WaveNode zip backup, open **Admin Dashboard > Library jobs > Backup and restore**, choose **Restore backup**, and select the zip. Restore replaces WaveNode database records and stored artwork. It never changes original audio files. Sign in again after it finishes.

For a manual PostgreSQL backup:
Stop the application while keeping PostgreSQL available:

```bash
docker compose stop frontend backend
```

Restore the database:

```bash
docker compose exec -T database psql -U wavenode -d postgres -c "DROP DATABASE IF EXISTS wavenode;"
docker compose exec -T database psql -U wavenode -d postgres -c "CREATE DATABASE wavenode OWNER wavenode;"
docker compose exec -T database psql -U wavenode -d wavenode < backup/wavenode.sql
```

Restore artwork:

```bash
docker compose cp backup/artwork/. backend:/data/artwork
```

Start WaveNode:

```bash
docker compose up -d
```

## Troubleshooting

- Setup cannot see a host path: only paths mounted into the backend container are visible. The supplied stack mounts the configured host folder at `/music`.
- Artwork cannot be saved: select `/data/artwork` and confirm the `artwork_data` volume is mounted.
- Tracks have zero duration: confirm FFmpeg is installed when running outside Docker.
- Browser cannot connect: check `docker compose logs backend frontend` and verify the reverse proxy supports `/ws`.
- HTTPS certificate cannot be issued: confirm public DNS points to this server and inbound TCP ports `80` and `443` are reachable.
- The domain works inside the network but not outside: check router forwarding, host firewall rules, and whether the ISP uses carrier-grade NAT.
- Setup access code is rejected: confirm the running backend received the same `SETUP_TOKEN` value entered in the wizard, then recreate the containers.
- Missing or unplayable tracks: open **Admin Dashboard > System** and review missing files and unavailable sources.
- Backup upload is rejected by a reverse proxy: allow request bodies up to the size of the backup archive; the bundled proxy permits 1 GB.
