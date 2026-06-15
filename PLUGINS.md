# WaveNode plugins

WaveNode plugins are declarative JSON manifests installed by an administrator.
They can contribute approved data to extension points, but they cannot execute
JavaScript, load native code, access the database, or register arbitrary HTTP
handlers.

## Install a plugin

1. Open **Admin Dashboard**.
2. Select **Plugins**.
3. Choose **Install plugin**.
4. Paste the manifest and select **Validate and install**.

Installing an existing plugin ID updates it and enables the new version.
Plugins can be disabled or removed from the same screen.

## Manifest version 1

```json
{
  "schema_version": 1,
  "id": "community.radio",
  "name": "Community Radio",
  "version": "1.0.0",
  "description": "Adds radio stations to the home page.",
  "homepage": "https://example.org",
  "permissions": ["network", "playback"],
  "contributes": {
    "home_rows": [
      {
        "id": "radio-stations",
        "title": "Radio stations",
        "subtitle": "Community-curated live streams",
        "type": "radio",
        "items": [
          {
            "id": "station-one",
            "title": "Station One",
            "subtitle": "Live radio",
            "description": "Optional longer description",
            "stream_url": "https://radio.example.org/live.mp3",
            "homepage_url": "https://radio.example.org",
            "image_url": "https://radio.example.org/logo.jpg"
          }
        ]
      }
    ]
  }
}
```

IDs must contain lowercase letters, numbers, dots, or hyphens. Media and
homepage URLs must be absolute HTTP or HTTPS URLs. Radio contributions require
the `network` and `playback` permissions.

## Current extension points

- `home_rows` with type `radio`

Enabled radio stations also appear through Subsonic's
`getInternetRadioStations` endpoint.

## Security limits

- Maximum manifest size: 256 KB
- Maximum home rows per plugin: 10
- Maximum items per row: 100
- Unknown manifest properties are rejected
- No executable plugin code is accepted
