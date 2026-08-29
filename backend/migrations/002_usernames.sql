-- Public Telegram-style usernames.
-- Additive migration: preserves existing email, messages, and user IDs.

ALTER TABLE users ADD COLUMN IF NOT EXISTS username TEXT;

-- Backfill old accounts from display_name. Duplicate names receive _2, _3, ...
WITH normalized AS (
    SELECT id,
           COALESCE(NULLIF(trim(both '_' FROM lower(regexp_replace(display_name, '[^a-zA-Z0-9_]+', '_', 'g'))), ''), 'user') AS base
    FROM users
), numbered AS (
    SELECT id, base,
           row_number() OVER (PARTITION BY base ORDER BY id) AS position
    FROM normalized
)
UPDATE users u
SET username = CASE
    WHEN n.position = 1 THEN left(n.base, 32)
    ELSE left(n.base, 27) || '_' || n.position::text
END
FROM numbered n
WHERE u.id = n.id AND u.username IS NULL;

-- Extremely unlikely UUID collision guard for truncated backfills.
WITH duplicates AS (
    SELECT id, username,
           row_number() OVER (PARTITION BY username ORDER BY id) AS position
    FROM users
)
UPDATE users u
SET username = left(u.username, 26) || '_' || substring(replace(u.id::text, '-', '') from 1 for 5)
FROM duplicates d
WHERE u.id = d.id AND d.position > 1;

ALTER TABLE users ALTER COLUMN username SET NOT NULL;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_format;
ALTER TABLE users ADD CONSTRAINT users_username_format CHECK (username ~ '^[a-z0-9][a-z0-9_]{2,31}$');
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);
