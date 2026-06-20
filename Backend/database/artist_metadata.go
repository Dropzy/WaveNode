package database

import (
	"database/sql"
	"fmt"
	"time"
)

func (db *DB) UpsertArtistExternalID(artistID, provider, externalID, externalURL string) error {
	_, err := db.conn.Exec(`
		INSERT INTO artist_external_ids (artist_id, provider, external_id, external_url, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (artist_id, provider) DO UPDATE
		SET external_id = EXCLUDED.external_id,
		    external_url = EXCLUDED.external_url,
		    updated_at = CURRENT_TIMESTAMP
	`, artistID, provider, externalID, nullableString(externalURL))
	if err != nil {
		return fmt.Errorf("failed to save artist external id: %v", err)
	}
	return nil
}

func (db *DB) UpsertArtistImage(image *ArtistImage) error {
	if image == nil {
		return fmt.Errorf("artist image is required")
	}
	if image.AttributionText == "" && image.AuthorName != "" {
		image.AttributionText = image.AuthorName
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if image.IsPrimary {
		if _, err := tx.Exec(`UPDATE artist_images SET is_primary = FALSE WHERE artist_id = $1`, image.ArtistID); err != nil {
			return fmt.Errorf("failed to clear primary artist images: %v", err)
		}
	}

	err = tx.QueryRow(`
		INSERT INTO artist_images (
			artist_id, source, image_url, thumbnail_url, source_page_url,
			license_name, license_url, author_name, attribution_text,
			width, height, mime_type, confidence_score, is_primary,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
		ON CONFLICT (artist_id, source, image_url) DO UPDATE
		SET thumbnail_url = EXCLUDED.thumbnail_url,
		    source_page_url = EXCLUDED.source_page_url,
		    license_name = EXCLUDED.license_name,
		    license_url = EXCLUDED.license_url,
		    author_name = EXCLUDED.author_name,
		    attribution_text = EXCLUDED.attribution_text,
		    width = EXCLUDED.width,
		    height = EXCLUDED.height,
		    mime_type = EXCLUDED.mime_type,
		    confidence_score = EXCLUDED.confidence_score,
		    is_primary = EXCLUDED.is_primary,
		    updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`, image.ArtistID, image.Source, image.ImageURL, nullableString(image.ThumbnailURL), nullableString(image.SourcePageURL),
		nullableString(image.LicenseName), nullableString(image.LicenseURL), nullableString(image.AuthorName), nullableString(image.AttributionText),
		image.Width, image.Height, nullableString(image.MimeType), image.ConfidenceScore, image.IsPrimary).Scan(&image.ID, &image.CreatedAt, &image.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save artist image: %v", err)
	}

	if image.IsPrimary {
		if _, err := tx.Exec(`
			UPDATE artists
			SET image_url = $2,
			    image_small_url = COALESCE(NULLIF($3, ''), $2),
			    image_medium_url = COALESCE(NULLIF($3, ''), $2),
			    image_large_url = $2,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, image.ArtistID, image.ImageURL, image.ThumbnailURL); err != nil {
			return fmt.Errorf("failed to update artist primary image: %v", err)
		}
	}

	return tx.Commit()
}

func (db *DB) ListArtistImages(artistID string) ([]ArtistImage, error) {
	rows, err := db.conn.Query(`
		SELECT id, artist_id, source, image_url, thumbnail_url, source_page_url,
		       license_name, license_url, author_name, attribution_text,
		       width, height, mime_type, confidence_score, is_primary,
		       created_at, updated_at
		FROM artist_images
		WHERE artist_id = $1
		ORDER BY is_primary DESC, confidence_score DESC, updated_at DESC
	`, artistID)
	if err != nil {
		return nil, fmt.Errorf("failed to query artist images: %v", err)
	}
	defer rows.Close()

	images := make([]ArtistImage, 0)
	for rows.Next() {
		image, err := scanArtistImage(rows)
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, rows.Err()
}

func (db *DB) GetPrimaryArtistImage(artistID string) (*ArtistImage, error) {
	row := db.conn.QueryRow(`
		SELECT id, artist_id, source, image_url, thumbnail_url, source_page_url,
		       license_name, license_url, author_name, attribution_text,
		       width, height, mime_type, confidence_score, is_primary,
		       created_at, updated_at
		FROM artist_images
		WHERE artist_id = $1 AND is_primary = TRUE
		ORDER BY confidence_score DESC, updated_at DESC
		LIMIT 1
	`, artistID)
	image, err := scanArtistImage(row)
	if err != nil {
		return nil, err
	}
	return &image, nil
}

func (db *DB) SetPrimaryArtistImage(artistID string, imageID int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE artist_images SET is_primary = FALSE WHERE artist_id = $1`, artistID); err != nil {
		return err
	}
	var imageURL, thumbnailURL string
	if err := tx.QueryRow(`
		UPDATE artist_images
		SET is_primary = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE artist_id = $1 AND id = $2
		RETURNING image_url, COALESCE(thumbnail_url, '')
	`, artistID, imageID).Scan(&imageURL, &thumbnailURL); err != nil {
		return err
	}
	if thumbnailURL == "" {
		thumbnailURL = imageURL
	}
	if _, err := tx.Exec(`
		UPDATE artists
		SET image_url = $2, image_small_url = $3, image_medium_url = $3, image_large_url = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, artistID, imageURL, thumbnailURL); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) DeleteArtistImageRecord(artistID, imageURL string) error {
	if imageURL == "" {
		return nil
	}
	_, err := db.conn.Exec(`
		DELETE FROM artist_images
		WHERE artist_id = $1 AND image_url = $2
	`, artistID, imageURL)
	if err != nil {
		return fmt.Errorf("failed to delete artist image record: %v", err)
	}
	return nil
}

type artistImageScanner interface {
	Scan(dest ...interface{}) error
}

func scanArtistImage(scanner artistImageScanner) (ArtistImage, error) {
	var image ArtistImage
	var thumbnailURL, sourcePageURL, licenseName, licenseURL, authorName, attributionText, mimeType sql.NullString
	err := scanner.Scan(
		&image.ID, &image.ArtistID, &image.Source, &image.ImageURL, &thumbnailURL, &sourcePageURL,
		&licenseName, &licenseURL, &authorName, &attributionText,
		&image.Width, &image.Height, &mimeType, &image.ConfidenceScore, &image.IsPrimary,
		&image.CreatedAt, &image.UpdatedAt,
	)
	if err != nil {
		return image, err
	}
	image.ThumbnailURL = thumbnailURL.String
	image.SourcePageURL = sourcePageURL.String
	image.LicenseName = licenseName.String
	image.LicenseURL = licenseURL.String
	image.AuthorName = authorName.String
	image.AttributionText = attributionText.String
	image.MimeType = mimeType.String
	return image, nil
}

func (db *DB) GetSourceAPICache(cacheKey string) ([]byte, bool, error) {
	var payload []byte
	err := db.conn.QueryRow(`
		SELECT response_body
		FROM source_api_cache
		WHERE cache_key = $1 AND expires_at > CURRENT_TIMESTAMP
	`, cacheKey).Scan(&payload)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to read source api cache: %v", err)
	}
	return payload, true, nil
}

func (db *DB) SetSourceAPICache(cacheKey string, provider string, payload []byte, expiresAt time.Time) error {
	_, err := db.conn.Exec(`
		INSERT INTO source_api_cache (cache_key, provider, response_body, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (cache_key) DO UPDATE
		SET provider = EXCLUDED.provider,
		    response_body = EXCLUDED.response_body,
		    expires_at = EXCLUDED.expires_at,
		    updated_at = CURRENT_TIMESTAMP
	`, cacheKey, provider, payload, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to write source api cache: %v", err)
	}
	return nil
}

func (db *DB) GetArtistsNeedingMetadataRefresh(limit int, staleAfter time.Duration) ([]Artist, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := db.conn.Query(`
		SELECT id
		FROM artists
		WHERE last_enriched_at IS NULL OR last_enriched_at < $1
		ORDER BY COALESCE(last_enriched_at, TIMESTAMP '1970-01-01') ASC, name ASC
		LIMIT $2
	`, time.Now().Add(-staleAfter), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find stale artist metadata: %v", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	artists := make([]Artist, 0, len(ids))
	for _, id := range ids {
		artist, err := db.GetArtistByID(id)
		if err != nil {
			return nil, err
		}
		artists = append(artists, *artist)
	}
	return artists, nil
}
