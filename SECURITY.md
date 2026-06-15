# Security

## Supported Releases

Security fixes are applied to the latest release.

## Deployment Requirements

- Use a unique `JWT_SECRET` of at least 32 characters.
- Use a strong PostgreSQL password.
- Keep public registration disabled unless it is intentionally needed.
- Use HTTPS before exposing WaveNode outside a trusted network.
- Mount music folders read-only.
- Do not publish PostgreSQL directly to the internet.
- Keep the container images and host operating system updated.
- Keep backups outside the WaveNode host and encrypt them when stored on shared or remote systems.
- Review **Admin Dashboard > System** after changing mounts or permissions.
- Disable query-string access logging for `/api/music/*/stream` and `/ws` in custom reverse proxies.

WaveNode refuses to start in production with the bundled development JWT secret.
Login failures are rate limited, and the bundled frontend adds restrictive browser security headers.
Changing a password or account role invalidates previously issued session tokens.
The bundled Nginx configuration suppresses access logs for authenticated browser audio and WebSocket URLs.

## Reporting

Use the repository's **Security** tab to submit a private vulnerability report. Do not publish suspected vulnerabilities in a public issue containing credentials, tokens, private paths, or personal library information.

Include reproduction steps, affected versions, expected impact, and any known mitigations. You should receive an acknowledgement within seven days.

## Sensitive Data

Backups contain account records, playlists, listening history, library paths, and artwork. Store backups with access controls appropriate for private data.
