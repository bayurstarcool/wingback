-- Wingback initial schema
-- Run with: psql $DATABASE_URL -f migrations/001_init.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    avatar_url    TEXT,
    currency      INTEGER NOT NULL DEFAULT 0,
    last_lat      DOUBLE PRECISION,
    last_lng      DOUBLE PRECISION,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE carriers (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug         TEXT UNIQUE NOT NULL,       -- 'pigeon', 'eagle', 'drone', 'paper_plane'
    name         TEXT NOT NULL,
    speed_kmh    DOUBLE PRECISION NOT NULL,
    is_default   BOOLEAN NOT NULL DEFAULT false,
    price        INTEGER NOT NULL DEFAULT 0, -- in-app currency cost, 0 = free
    rarity       TEXT NOT NULL DEFAULT 'common', -- common, rare, epic, legendary
    asset_url    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_inventory (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    carrier_id  UUID NOT NULL REFERENCES carriers(id) ON DELETE CASCADE,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, carrier_id)
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sender_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    carrier_id      UUID NOT NULL REFERENCES carriers(id),
    body            TEXT NOT NULL,
    sender_lat      DOUBLE PRECISION NOT NULL,
    sender_lng      DOUBLE PRECISION NOT NULL,
    recipient_lat   DOUBLE PRECISION NOT NULL,
    recipient_lng   DOUBLE PRECISION NOT NULL,
    distance_km     DOUBLE PRECISION NOT NULL,
    speed_kmh       DOUBLE PRECISION NOT NULL,
    status          TEXT NOT NULL DEFAULT 'in_transit', -- in_transit, delivered, lost
    departs_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    arrives_at      TIMESTAMPTZ NOT NULL,
    delivered_at    TIMESTAMPTZ,
    speedups_used   INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_recipient_status ON messages(recipient_id, status);
CREATE INDEX idx_messages_sender ON messages(sender_id);
CREATE INDEX idx_messages_arrives_at ON messages(arrives_at) WHERE status = 'in_transit';

CREATE TABLE ad_rewards (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_id  UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    watched_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ad_rewards_user_day ON ad_rewards(user_id, watched_at);

-- Seed default carriers
INSERT INTO carriers (slug, name, speed_kmh, is_default, price, rarity) VALUES
    ('pigeon', 'Carrier Pigeon', 177, true, 0, 'common'),
    ('paper_plane', 'Paper Plane', 250, false, 100, 'rare'),
    ('falcon', 'Peregrine Falcon', 320, false, 300, 'epic'),
    ('drone', 'Mini Drone', 500, false, 800, 'legendary');
