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

## 0.1.9 - 2026-07-03

### Added

- Spotify-style Connect handoff for local music, podcasts, audiobooks, radio, and mixed playable queues across web and Android devices.
- Explicit playback ownership changes that stop the previous playback host when control moves to another WaveNode device.
- HLS internet-radio playback support in the Android app.

### Fixed

- Radio genre filters now return stations by normalising Radio Browser tags.
- Android radio playback failures no longer force-close the app.
- Android Home radio-card spacing now matches album cards.

### Security

- External handoff items are size-limited, type-restricted, sanitised, and required to use secure stream URLs.

## 0.1.8 - 2026-07-02

### Added

- Native internet radio discovery, search, genre browsing, secure playback, live metadata, and per-user favourites on web and Android.
- Favourite radio stations on the Home screen and in Subsonic internet-radio clients.
- Audio-file previews in the administration and setup folder pickers.

### Changed

- Library job history now displays only the ten most recent completed scans.
- Application backups now include per-user radio favourites.

### Security

- Radio metadata requests enforce HTTPS streams, public network destinations, and safe redirects.

## 0.1.7 - 2026-07-02

### Fixed

- Installed-version reporting now uses the running backend version instead of the independently versioned updater sidecar.
- Administration dashboard version labels no longer display duplicate `v` prefixes.
- System version details refresh after an in-app update completes.

## 0.1.6 - 2026-07-02

### Added

- Per-user music-source permissions managed from the administration dashboard.
- Server-side access enforcement across library browsing, search, playlists, history, streaming, downloads, casting, lyrics, discovery, and Subsonic clients.

### Security

- Restricted users can no longer retrieve or play tracks outside their assigned music folders through direct media endpoints or secondary playback features.

## 0.1.5 - 2026-06-30

### Added

- Online public-domain audiobook browsing and search through LibriVox on web and Android.
- Chapter playback from Internet Archive with covers, next/previous controls, seeking, speed, sleep timers, and per-user continue-listening progress.

### Fixed

- Android no longer crashes when opening the Google Cast route chooser from the Connect sheet.

## 0.1.4 - 2026-06-30

### Added

- Time-synced lyrics on web and Android with active-line scrolling and seekable lyric lines.
- Plain-text lyric fallback, local `.lrc` and `.txt` sidecar support, and keyless LRCLIB lookup.
- Lyrics responses for Subsonic and OpenSubsonic-compatible clients.

### Fixed

- Android Connect sheet crashes caused by MediaRouter theme and initialization failures.
- Misleading end-stop color on Android podcast progress cards.

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
