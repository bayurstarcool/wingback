-- Per-message location privacy and location freshness metadata.
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_location_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_location_accuracy_m DOUBLE PRECISION;

-- Existing coordinates were already stored by the previous MVP. Treat their
-- last profile update as the best available freshness signal.
UPDATE users
SET last_location_at = COALESCE(last_location_at, updated_at)
WHERE last_lat IS NOT NULL AND last_lng IS NOT NULL;

ALTER TABLE messages ADD COLUMN IF NOT EXISTS location_privacy TEXT NOT NULL DEFAULT 'accurate';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_city TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS recipient_city TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_city_lat DOUBLE PRECISION;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_city_lng DOUBLE PRECISION;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS recipient_city_lat DOUBLE PRECISION;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS recipient_city_lng DOUBLE PRECISION;
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_location_privacy_check;
ALTER TABLE messages ADD CONSTRAINT messages_location_privacy_check
    CHECK (location_privacy IN ('accurate', 'hidden'));