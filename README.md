# WaveNode

WaveNode is a self-hosted music server for streaming a personal music collection through a web interface. Music remains on your server, and album artwork is read from audio metadata before optional MusicBrainz enrichment is used.

> WaveNode does not provide music. You are responsible for ensuring you have the right to store and stream media added to your server.

## Features

- Dynamic smart playlists with nested match-all/match-any groups, relative dates, sorting, limits, and live previews
- Smart playlists remain fully playable in Subsonic clients as read-only snapshots
- Browser-based playback, queue, search, playlists, liked tracks, albums, and artists
- Online public-domain audiobooks from LibriVox with chapter playback and synchronized reading progress
- Subsonic/OpenSubsonic-compatible API for third-party music clients
- ReplayGain track/album normalization and configurable per-user transcoding profiles
- Searchable listening history with CSV export and privacy controls
- Connected-device session management and remote session revocation
- M3U/M3U8 playlist import and export
- Multi-disc album ordering and disc-aware album displays
- First-run administrator and storage setup
- Multiple server-side music folders
- Embedded album artwork extraction during library scans
- Local artist-image discovery
- Background scan and enrichment progress
- Automatic library updates with file-change detection and scheduled rescans
- Downloadable backups, restore, library diagnostics, and server monitoring
- Installable web app with playback session recovery and media controls
- Standalone Electron desktop client with server URL entry and LAN server discovery
- Native Android client for phone playback and remote device control
- Account management with administrator-created users
- Declarative plugins with administrator-managed home-page radio rows
- PostgreSQL persistence
- Docker Compose deployment

## Quick Start

Requirements:

- Docker Engine with Docker Compose
- A folder containing supported audio files

```bash
cp .env.example .env
```

Edit `.env` and set:

- `POSTGRES_PASSWORD`
- `JWT_SECRET` to a unique random value of at least 32 characters
- `MUSIC_PATH` to the absolute host path containing music

Start WaveNode:

```bash
docker compose up -d
```

Open `http://localhost:8080`. The first-run wizard creates the administrator and asks for:

- Music folder: choose `/music`
- Artwork folder: choose `/data/artwork`

See [DEPLOYMENT.md](DEPLOYMENT.md) for HTTPS, one-click Docker updates, backup, and restore instructions.
See [SUBSONIC.md](SUBSONIC.md) for compatible client setup and supported endpoints.
See [PLUGINS.md](PLUGINS.md) for the plugin manifest format and extension points.
See [SMART_PLAYLISTS.md](SMART_PLAYLISTS.md) for rule groups and relative-date examples.

## Client Apps

The Docker deployment serves the WaveNode web app at `http://localhost:8080`.

The Electron desktop app is a standalone client for any WaveNode server. It opens with the same login flow as the Android app, lets the user enter a server address, and can discover WaveNode servers on the local network.

```bash
cd Frontend
npm ci
npm run desktop:build
```

The packaged desktop app is written to `Frontend/release/`.

The Android app lives in `Android/` and connects to an existing WaveNode server by URL or local network discovery.

## Development

Backend:

```bash
cd Backend
go run ./cmd/server
```

Frontend:

```bash
cd Frontend
npm ci
npm run dev
```

The Vite development server proxies API and WebSocket traffic to `127.0.0.1:8080`.

## Supported Formats

MP3, FLAC, WAV, M4A, AAC, OGG, WMA, and Opus. FFmpeg and FFprobe are included in the backend container for duration detection.

## Security

Public registration is disabled by default in production. Administrators can create accounts under **Admin Dashboard > Users**.

Read [SECURITY.md](SECURITY.md) before exposing WaveNode outside a trusted network.

For secure internet access, WaveNode includes an optional Caddy gateway that obtains and renews HTTPS certificates automatically. Follow the **Internet Deployment** section in [DEPLOYMENT.md](DEPLOYMENT.md); do not forward port `8080`.

## Project Status

WaveNode `v0.1.6` is the current stable release. Docker/web playback is the primary server release target. The Electron desktop client and Android client are included as companion clients for connecting to an existing WaveNode server.

## Community

- Read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting changes.
- Use [SUPPORT.md](SUPPORT.md) for support guidance.
- Follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) in project spaces.
- Report vulnerabilities according to [SECURITY.md](SECURITY.md).

WaveNode is available under the [MIT License](LICENSE).
