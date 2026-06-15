# Changelog

All notable changes to WaveNode are documented here.

The project follows [Semantic Versioning](https://semver.org/).

## Unreleased

### Added

- Downloadable application backups and in-app restore.
- Playback session recovery and operating-system media controls.
- Account password changes and an account screen.
- Library diagnostics, server health monitoring, and source availability checks.
- Installable PWA metadata, offline application shell, keyboard skip navigation, and reduced-motion support.
- Login rate limiting, browser security headers, and versioned container metadata.
- Dynamic smart playlists with live previews and read-only Subsonic snapshots.
- Nested smart-playlist groups and rolling relative-date conditions.
- ReplayGain track/album normalization and configurable MP3, Opus, or AAC transcoding profiles.
- Searchable listening history with CSV export and clear controls.
- Connected-device session management with individual and bulk revocation.
- M3U/M3U8 playlist import and export backed by the same playlists exposed through Subsonic.
- Disc-aware metadata, ordering, and album-page sections for multi-disc releases.
- PostgreSQL integration coverage for Subsonic compatibility and final-administrator protection.
- Correct source content types for MP3, FLAC, WAV, M4A, AAC, OGG, Opus, and AIFF streams.

## 0.1.0 - 2026-06-10

### Added

- Browser playback, queue controls, search, playlists, liked tracks, albums, and artists.
- First-run setup for the administrator account, music sources, and artwork storage.
- Embedded artwork extraction with optional MusicBrainz cover-art enrichment.
- Local artist-image discovery with background progress and error reporting.
- Library administration, user management, and scan history.
- Docker Compose deployment with PostgreSQL, a non-root backend, and an Nginx frontend.
- Production configuration validation, restricted registration, CORS controls, and protected administration routes.

### Fixed

- Artwork selection across the library, album pages, recently played, and player controls.
- Accurate scan and enrichment progress.
- Track playback and alignment in the library track view.
- Artist creation and recently played persistence on fresh PostgreSQL installations.
- Browser playback startup, single-click volume changes, and authenticated stream proxying.
