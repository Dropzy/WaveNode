# Artist metadata and images

WaveNode sources artist metadata from MusicBrainz, then follows MusicBrainz URL
relationships to Wikidata and Wikimedia Commons. Artist photos are only selected
when the Commons metadata reports a reusable license such as CC BY, CC BY-SA,
CC0, or Public Domain.

WaveNode stores the original image URL, thumbnail URL, source page URL, license
name and URL, author, attribution text, dimensions, MIME type, confidence score,
and source. The UI should show the source and license anywhere externally
sourced images are displayed.

The pipeline deliberately does not scrape Google Images, Spotify, Apple Music,
Instagram, Facebook, or other commercial image sources. Cover Art Archive is
reserved for album artwork. Fanart.tv can be added only when an administrator
provides an API key, and any use must still respect the upstream terms.

Administrators can upload custom artist images as a fallback. In that case the
administrator is responsible for having permission to use the image and should
fill in attribution details when needed.

Relevant environment variables:

- `FANART_TV_API_KEY`: optional, enables a future Fanart.tv provider.
- `ARTIST_METADATA_REFRESH_ENABLED`: optional feature flag for scheduled
  refresh jobs.
