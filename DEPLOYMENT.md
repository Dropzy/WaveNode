# Deployment

## Docker Compose

1. Copy `.env.example` to `.env`.
2. Set strong values for `POSTGRES_PASSWORD` and `JWT_SECRET`.
3. Set `MUSIC_PATH` to an absolute folder on the Docker host.
4. Run `docker compose up -d --build`.
5. Open `http://server-address:8080`.
6. In first-run setup, select `/music` and `/data/artwork`.

The host music folder is mounted read-only. Database and extracted artwork use Docker volumes.

## HTTPS

Do not expose the plain HTTP port directly to the internet. Put WaveNode behind a reverse proxy that:

- Terminates TLS
- Proxies normal HTTP traffic to port 8080
- Supports WebSocket upgrades on `/ws`
- Allows long responses for audio streaming

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

## Upgrade

Download a backup from **Admin Dashboard > Library jobs > Backup and restore** before upgrading. It contains WaveNode accounts, playlists, settings, indexed data, listening history, and stored artwork. Original audio files remain separate.

Then run:

```bash
docker compose pull
docker compose up -d --build
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
- Missing or unplayable tracks: open **Admin Dashboard > System** and review missing files and unavailable sources.
- Backup upload is rejected by a reverse proxy: allow request bodies up to the size of the backup archive; the bundled proxy permits 1 GB.
