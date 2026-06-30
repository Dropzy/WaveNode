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
- Optional internet deployment with automatic HTTPS certificates through Caddy.
- First-run setup access codes for preventing remote administrator-account claims.

## 0.1.3 - 2026-06-30

### Added

- Chromecast output on web and Android, AirPlay output in supported Apple browsers, and DLNA/UPnP renderer discovery and playback.
- Secure, expiring receiver stream URLs for playing authenticated library tracks on household devices.
- Android podcast subscriptions, offline downloads, sleep timers, playback speed, configurable skip intervals, sharing, notes, and queue controls.

### Fixed

- Android landscape layouts and podcast-specific transport controls.
- Web podcast controls, sleep-timer option contrast, and containment within the desktop player bar.
- Cast handoff now pauses local playback only after the receiver accepts the media.

## 0.1.2 - 2026-06-28

### Fixed

- Updates now target the existing WaveNode Compose project instead of creating a duplicate `compose` stack.
- Updater version reporting now uses the version compiled into the released image.

## 0.1.1 - 2026-06-28

### Added

- Podcast discovery, playback, progress synchronization, Continue Listening, and top-podcast rows on web and Android.
- Automated release containers and the optional updater sidecar.

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
- WebSocket origin validation and private-by-default host port binding.
