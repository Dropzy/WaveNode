# Smart Playlists

WaveNode smart playlists are saved rules rather than fixed track lists. Create one from the
sidebar or the Playlists tab in Your Library.

Rules can match track title, artist, album, genre, year, duration, play count, rating, date
added, liked status, and artwork availability. A playlist can match every condition or any
condition, then sort and limit the results.

Rule groups can be nested up to three levels. Each group independently chooses whether every
rule or any rule must match. For example:

- Match `genre is Drum & Bass`
- AND a nested group where `rating is at least 4` OR `liked is true`
- AND `date added is within the last 30 days`

Relative-date rules are evaluated whenever the playlist is opened, so a rolling "recently
added" playlist stays current without being edited. The editor limits a smart playlist to 50
conditions to keep previews and Subsonic snapshots responsive.

Membership is recalculated when the playlist is opened or requested through the API. Adding,
removing, rating, liking, or rescanning tracks can therefore change the result automatically.

## Subsonic compatibility

Subsonic and OpenSubsonic clients see a smart playlist as a normal playlist snapshot and can
browse or play its tracks. The snapshot is read-only through Subsonic: rename, add/remove track,
and delete requests return error code `50`. Manage the name, rules, or deletion in WaveNode.

This keeps smart-playlist behavior available without changing the Subsonic playlist contract or
allowing a third-party client to replace the saved rules with a static list.
