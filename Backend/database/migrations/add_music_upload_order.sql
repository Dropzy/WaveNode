CREATE SEQUENCE IF NOT EXISTS music_upload_order_seq;
ALTER TABLE music ADD COLUMN IF NOT EXISTS upload_order BIGINT;

WITH existing_max AS (
    SELECT COALESCE(MAX(upload_order), 0) AS max_order FROM music
),
missing_order AS (
    SELECT id,
           ROW_NUMBER() OVER (ORDER BY created_at ASC, ctid ASC)
           + (SELECT max_order FROM existing_max) AS new_order
    FROM music
    WHERE upload_order IS NULL
)
UPDATE music
SET upload_order = missing_order.new_order
FROM missing_order
WHERE music.id = missing_order.id;

SELECT setval(
    'music_upload_order_seq',
    GREATEST(COALESCE((SELECT MAX(upload_order) FROM music), 0), 1),
    COALESCE((SELECT MAX(upload_order) FROM music), 0) > 0
);

ALTER TABLE music
    ALTER COLUMN upload_order SET DEFAULT nextval('music_upload_order_seq');
ALTER TABLE music ALTER COLUMN upload_order SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_music_upload_order ON music(upload_order DESC);
