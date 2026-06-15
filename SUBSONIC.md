# Subsonic Compatibility

WaveNode exposes a Subsonic/OpenSubsonic-compatible API at:

```text
https://your-wavenode-server/rest
```

Use the same username and password as the WaveNode web application. In the
client, enable password or legacy authentication. WaveNode accepts clear
passwords and Subsonic's `enc:` hexadecimal password format.

Always use HTTPS outside a trusted local network.

## Authentication

WaveNode stores account passwords with bcrypt and does not retain a reversible
password. Subsonic's token scheme requires the server to calculate
`md5(password + salt)`, which is impossible without weakening that storage.
Token/salt authentication therefore returns Subsonic error `41`.

## Supported API

WaveNode currently reports Subsonic API version `1.16.1`.

- System: `ping`, `getLicense`, `getOpenSubsonicExtensions`
- Browsing: `getMusicFolders`, `getIndexes`, `getMusicDirectory`, `getGenres`,
  `getArtists`, `getArtist`, `getAlbum`, `getSong`, `getArtistInfo`,
  `getArtistInfo2`, `getAlbumInfo`, `getAlbumInfo2`, `getTopSongs`,
  `getSimilarSongs`, `getSimilarSongs2`
- Lists: `getAlbumList`, `getAlbumList2`, `getRandomSongs`,
  `getSongsByGenre`, `getNowPlaying`, `getStarred`, `getStarred2`
- Search: `search`, `search2`, `search3`
- Media: `stream`, `download`, `getCoverArt`, `getLyrics`,
  `getLyricsBySongId` (empty until lyrics are stored)
- Playlists: `getPlaylists`, `getPlaylist`, `createPlaylist`,
  `updatePlaylist`, `deletePlaylist`
- Activity: `star`, `unstar`, `setRating`, `scrobble`
- Playback state: `getBookmarks`, `createBookmark`, `deleteBookmark`,
  `getPlayQueue`, `savePlayQueue`
- Scanning: `getScanStatus`, `startScan` (administrator only)
- Users: `getUser`, `getUsers`, `createUser`, `updateUser`, `deleteUser`,
  `changePassword`

Ratings are stored per user for songs, albums, and artists. A rating of `0`
removes the existing rating. Rated media includes the standard `userRating`
field in Subsonic responses. WaveNode's web player uses the same rating records.

The `stream` endpoint honors the client's `maxBitRate`, `format`, and
`timeOffset` parameters. It also applies the signed-in user's ReplayGain mode
and preamp. Supported transcoding outputs are MP3, Opus, and AAC; `download`
always returns the original file.

Manual WaveNode playlists use the same records exposed by Subsonic
`getPlaylists`, `createPlaylist`, `updatePlaylist`, and `deletePlaylist`.
Changes made in either interface are therefore visible to the other on its
next refresh. WaveNode can also import and export these playlists as M3U/M3U8.

Playlist metadata responses are marked as non-cacheable and preserve
sub-second `changed` timestamps. Subsonic does not define a server-push
playlist notification, so third-party clients refresh according to their own
polling or navigation behavior. Reopening or manually refreshing the playlist
view requests the latest server state.

JSON and XML responses are available. Most modern clients request JSON.

## Compatibility Responses

The following read methods return valid empty collections so clients can safely
detect that the corresponding optional library is empty:

- Video: `getVideos`
- Sharing: `getShares`
- Podcasts: `getPodcasts`, `getNewestPodcasts`
- Internet radio: `getInternetRadioStations`
- Chat: `getChatMessages`

Mutating methods for shares, podcasts, internet radio, and chat return a
permission error because those optional server features are not enabled.
Jukebox, HLS, captions, avatars, and video details are also reported as
unavailable rather than silently pretending to succeed.

## Connection Test

```text
/rest/ping.view?u=USERNAME&p=PASSWORD&v=1.16.1&c=CLIENT&f=json
```

The server URL entered in a client should normally be the WaveNode origin,
such as `https://music.example.com`, without adding `/rest`.

## Integration Tests

The integration suite starts an isolated temporary PostgreSQL instance and
tests the real HTTP router, migrations, authentication, endpoint responses,
streaming, seeking, playlists, ratings, bookmarks, and saved queues.

On Windows:

```powershell
.\scripts\test-subsonic.ps1
```

The test database uses temporary Docker storage and is removed when the suite
finishes. It does not read or modify the configured WaveNode music library.
