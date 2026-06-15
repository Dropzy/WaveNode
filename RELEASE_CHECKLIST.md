# Release Checklist

## Required Before Tagging

- [x] Add an open-source license
- [x] Keep deployment secrets in ignored local environment files
- [x] Run backend tests
- [x] Scan reachable Go code for known vulnerabilities
- [x] Run frontend build and lint
- [x] Audit production frontend dependencies
- [x] Build the Docker Compose stack
- [x] Verify ReplayGain/transcoding against the rebuilt container
- [x] Verify scan-time technical metadata persistence for the complete live library
- [x] Run automated coverage for nested smart-playlist and relative-date rules
- [x] Complete first-run setup in a fresh database
- [x] Scan a representative MP3, FLAC, WAV, M4A, OGG, and Opus library
- [x] Test search, playlists, liked tracks, queue controls, seeking, volume, next, and previous
- [x] Test a regular user cannot access administration endpoints
- [x] Test the final administrator cannot be deleted or demoted
- [x] Test backup and restore on a disposable installation
- [x] Test authenticated search, playlist add/remove, liked add/remove, and HTTP range streaming
- [x] Test application backup download, restore, and sign-in recovery
- [x] Test a regular user cannot access administration endpoints or another account's playlists
- [x] Verify production PWA assets, security headers, version metadata, and non-root container execution
- [x] Test HTTPS and WebSocket connectivity through the intended reverse proxy
- [x] Review generated artwork and local environment files are absent from the release archive
- [x] Publish from a clean or squashed Git history; legacy diagnostic commits contained local paths and expired test tokens

Run the repeatable local checks with:

```powershell
./scripts/verify-release.ps1
```

## Release Artifacts

- Source archive
- Container images or reproducible Docker build instructions
- Versioned changelog
- Upgrade notes for schema or configuration changes
- Checksums for downloadable binaries, if binaries are published

## Known Scope

The web application and Docker deployment are the primary release targets. Treat the Android client as separate preview software until its build, authentication, playback, and release signing have their own completed checklist.
