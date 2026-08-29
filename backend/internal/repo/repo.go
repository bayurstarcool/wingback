// Package repo holds the data-access layer. The current implementation
// is hand-written SQL via pgx; once it grows past a handful of queries
// it would be reasonable to swap to sqlc or similar codegen, but for a
// 4-week MVP this keeps the dependency footprint small and the code
// trivial to follow.
package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bayurstarcool/wingback/backend/internal/models"
)

var ErrNotFound = errors.New("not found")

type Repo struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{Pool: pool}
}

// --- users ---

func usernameFromDisplayName(displayName string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(displayName) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '-' {
			b.WriteByte('_')
		}
	}
	name := strings.Trim(b.String(), "_")
	if len(name) < 3 {
		name = "user_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	}
	if len(name) > 32 {
		name = name[:32]
	}
	return name
}

func (r *Repo) CreateUser(ctx context.Context, email, passwordHash, displayName string) (*models.User, error) {
	base := usernameFromDisplayName(displayName)
	if len(base) > 25 {
		base = base[:25]
	}
	return r.CreateUserWithUsername(ctx, email, passwordHash, displayName, base+"_"+strings.ReplaceAll(uuid.NewString()[:6], "-", ""))
}

func (r *Repo) CreateUserWithUsername(ctx context.Context, email, passwordHash, displayName, username string) (*models.User, error) {
	row := r.Pool.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, username, password_hash, display_name, COALESCE(avatar_url, ''), currency,
		          last_lat, last_lng, last_location_at, last_location_accuracy_m, created_at, updated_at
	`, email, username, passwordHash, displayName)

	u := &models.User{}
	var lastLat, lastLng, accuracy *float64
	var locationAt *time.Time
	if err := row.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.Currency, &lastLat, &lastLng, &locationAt, &accuracy, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	u.LastLat = lastLat
	u.LastLng = lastLng
	u.LastLocationAt = locationAt
	u.LastLocationAccuracyM = accuracy
	return u, nil
}

func (r *Repo) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	row := r.Pool.QueryRow(ctx, `
		SELECT id, email, username, password_hash, display_name, COALESCE(avatar_url, ''),
		       currency, last_lat, last_lng, last_location_at, last_location_accuracy_m, created_at, updated_at
		FROM users WHERE username = $1
	`, username)

	u := &models.User{}
	var lastLat, lastLng, accuracy *float64
	var locationAt *time.Time
	if err := row.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.Currency, &lastLat, &lastLng, &locationAt, &accuracy, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select user: %w", err)
	}
	u.LastLat = lastLat
	u.LastLng = lastLng
	u.LastLocationAt = locationAt
	u.LastLocationAccuracyM = accuracy
	return u, nil
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.Pool.QueryRow(ctx, `
		SELECT id, email, username, password_hash, display_name, COALESCE(avatar_url, ''), currency,
		       last_lat, last_lng, last_location_at, last_location_accuracy_m, created_at, updated_at
		FROM users WHERE email = $1
	`, email)

	u := &models.User{}
	var lastLat, lastLng, accuracy *float64
	var locationAt *time.Time
	if err := row.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.Currency, &lastLat, &lastLng, &locationAt, &accuracy, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select user: %w", err)
	}
	u.LastLat = lastLat
	u.LastLng = lastLng
	u.LastLocationAt = locationAt
	u.LastLocationAccuracyM = accuracy
	return u, nil
}

func (r *Repo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	row := r.Pool.QueryRow(ctx, `
		SELECT id, email, username, password_hash, display_name, COALESCE(avatar_url, ''), currency,
		       last_lat, last_lng, last_location_at, last_location_accuracy_m, created_at, updated_at
		FROM users WHERE id = $1
	`, id)

	u := &models.User{}
	var lastLat, lastLng, accuracy *float64
	var locationAt *time.Time
	if err := row.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.Currency, &lastLat, &lastLng, &locationAt, &accuracy, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select user: %w", err)
	}
	u.LastLat = lastLat
	u.LastLng = lastLng
	u.LastLocationAt = locationAt
	u.LastLocationAccuracyM = accuracy
	return u, nil
}

func (r *Repo) UpdateUserLocation(ctx context.Context, id string, lat, lng float64, accuracy ...*float64) error {
	var accuracyM *float64
	if len(accuracy) > 0 {
		accuracyM = accuracy[0]
	}
	_, err := r.Pool.Exec(ctx, `
		UPDATE users SET last_lat = $1, last_lng = $2, last_location_at = now(),
		last_location_accuracy_m = $4, updated_at = now() WHERE id = $3
	`, lat, lng, id, accuracyM)
	if err != nil {
		return fmt.Errorf("update location: %w", err)
	}
	return nil
}

// --- user discovery ---

func (r *Repo) SearchUsers(ctx context.Context, query string, limit int) ([]models.User, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, email, username, password_hash, display_name, COALESCE(avatar_url, ''), currency,
		       last_lat, last_lng, last_location_at, last_location_accuracy_m, created_at, updated_at
		FROM users
		WHERE (username ILIKE $1 OR display_name ILIKE $1)
		ORDER BY display_name ASC
		LIMIT $2
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	out := make([]models.User, 0)
	for rows.Next() {
		u := models.User{}
		var lastLat, lastLng, accuracy *float64
		var locationAt *time.Time
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
			&u.Currency, &lastLat, &lastLng, &locationAt, &accuracy, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.LastLat = lastLat
		u.LastLng = lastLng
		u.LastLocationAt = locationAt
		u.LastLocationAccuracyM = accuracy
		out = append(out, u)
	}
	return out, rows.Err()
}

// --- carriers ---

func (r *Repo) GetCarrierBySlug(ctx context.Context, slug string) (*models.Carrier, error) {
	row := r.Pool.QueryRow(ctx, `
		SELECT id, slug, name, speed_kmh, is_default, price, rarity, COALESCE(asset_url, '')
		FROM carriers WHERE slug = $1
	`, slug)

	c := &models.Carrier{}
	if err := row.Scan(&c.ID, &c.Slug, &c.Name, &c.SpeedKMH, &c.IsDefault, &c.Price,
		&c.Rarity, &c.AssetURL); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select carrier: %w", err)
	}
	return c, nil
}

func (r *Repo) GetDefaultCarrier(ctx context.Context) (*models.Carrier, error) {
	row := r.Pool.QueryRow(ctx, `
		SELECT id, slug, name, speed_kmh, is_default, price, rarity, COALESCE(asset_url, '')
		FROM carriers WHERE is_default = true LIMIT 1
	`)

	c := &models.Carrier{}
	if err := row.Scan(&c.ID, &c.Slug, &c.Name, &c.SpeedKMH, &c.IsDefault, &c.Price,
		&c.Rarity, &c.AssetURL); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select default carrier: %w", err)
	}
	return c, nil
}

func (r *Repo) ListCarriers(ctx context.Context) ([]models.Carrier, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, slug, name, speed_kmh, is_default, price, rarity, COALESCE(asset_url, '')
		FROM carriers ORDER BY price ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list carriers: %w", err)
	}
	defer rows.Close()

	out := make([]models.Carrier, 0)
	for rows.Next() {
		c := models.Carrier{}
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.SpeedKMH, &c.IsDefault, &c.Price,
			&c.Rarity, &c.AssetURL); err != nil {
			return nil, fmt.Errorf("scan carrier: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// --- messages ---

func (r *Repo) CreateMessage(ctx context.Context, m *models.Message) error {
	row := r.Pool.QueryRow(ctx, `
		INSERT INTO messages (
			sender_id, recipient_id, carrier_id, body,
			sender_lat, sender_lng, recipient_lat, recipient_lng, sender_city, recipient_city,
			sender_city_lat, sender_city_lng, recipient_city_lat, recipient_city_lng,
			distance_km, speed_kmh, status, departs_at, arrives_at, location_privacy
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		RETURNING id, created_at
	`,
		m.SenderID, m.RecipientID, m.CarrierID, m.Body,
		m.SenderLat, m.SenderLng, m.RecLat, m.RecLng, m.SenderCity, m.RecipientCity,
		m.SenderCityLat, m.SenderCityLng, m.RecipientCityLat, m.RecipientCityLng,
		m.DistanceKM, m.SpeedKMH, m.Status, m.DepartsAt, m.ArrivesAt, m.LocationPrivacy,
	)
	if err := row.Scan(&m.ID, &m.CreatedAt); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

func (r *Repo) GetMessage(ctx context.Context, id string) (*models.Message, error) {
	row := r.Pool.QueryRow(ctx, `
		SELECT id, sender_id, recipient_id, carrier_id, body,
		       sender_lat, sender_lng, recipient_lat, recipient_lng, COALESCE(sender_city, ''), COALESCE(recipient_city, ''),
		       sender_city_lat, sender_city_lng, recipient_city_lat, recipient_city_lng,
		       distance_km, speed_kmh, status, departs_at, arrives_at, delivered_at, speedups_used, location_privacy, created_at
		FROM messages WHERE id = $1
	`, id)

	m := &models.Message{}
	var deliveredAt *time.Time
	if err := row.Scan(&m.ID, &m.SenderID, &m.RecipientID, &m.CarrierID, &m.Body,
		&m.SenderLat, &m.SenderLng, &m.RecLat, &m.RecLng, &m.SenderCity, &m.RecipientCity,
		&m.SenderCityLat, &m.SenderCityLng, &m.RecipientCityLat, &m.RecipientCityLng,
		&m.DistanceKM, &m.SpeedKMH, &m.Status, &m.DepartsAt, &m.ArrivesAt,
		&deliveredAt, &m.SpeedupsUsed, &m.LocationPrivacy, &m.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select message: %w", err)
	}
	m.DeliveredAt = deliveredAt
	return m, nil
}

func (r *Repo) ListInbox(ctx context.Context, userID string, limit int) ([]models.Message, error) {
	return r.listMessages(ctx, `
		SELECT id, sender_id, recipient_id, carrier_id, body,
		       sender_lat, sender_lng, recipient_lat, recipient_lng, COALESCE(sender_city, ''), COALESCE(recipient_city, ''),
		       sender_city_lat, sender_city_lng, recipient_city_lat, recipient_city_lng,
		       distance_km, speed_kmh, status, departs_at, arrives_at, delivered_at, speedups_used, location_privacy, created_at
		FROM messages
		WHERE recipient_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
}

func (r *Repo) ListSent(ctx context.Context, userID string, limit int) ([]models.Message, error) {
	return r.listMessages(ctx, `
		SELECT id, sender_id, recipient_id, carrier_id, body,
		       sender_lat, sender_lng, recipient_lat, recipient_lng, COALESCE(sender_city, ''), COALESCE(recipient_city, ''),
		       sender_city_lat, sender_city_lng, recipient_city_lat, recipient_city_lng,
		       distance_km, speed_kmh, status, departs_at, arrives_at, delivered_at, speedups_used, location_privacy, created_at
		FROM messages
		WHERE sender_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
}

func (r *Repo) listMessages(ctx context.Context, sql string, args ...any) ([]models.Message, error) {
	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	out := make([]models.Message, 0)
	for rows.Next() {
		m := models.Message{}
		var deliveredAt *time.Time
		if err := rows.Scan(&m.ID, &m.SenderID, &m.RecipientID, &m.CarrierID, &m.Body,
			&m.SenderLat, &m.SenderLng, &m.RecLat, &m.RecLng, &m.SenderCity, &m.RecipientCity,
			&m.SenderCityLat, &m.SenderCityLng, &m.RecipientCityLat, &m.RecipientCityLng,
			&m.DistanceKM, &m.SpeedKMH, &m.Status, &m.DepartsAt, &m.ArrivesAt,
			&deliveredAt, &m.SpeedupsUsed, &m.LocationPrivacy, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.DeliveredAt = deliveredAt
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarkArrived flips an in_transit message to delivered if the ETA has
// passed. Returns true if the row was actually updated (a useful signal
// to drive a "just arrived" push notification or websocket event).
func (r *Repo) MarkArrived(ctx context.Context, id string, now time.Time) (bool, error) {
	tag, err := r.Pool.Exec(ctx, `
		UPDATE messages
		SET status = 'delivered', delivered_at = $2
		WHERE id = $1 AND status = 'in_transit' AND arrives_at <= $2
	`, id, now)
	if err != nil {
		return false, fmt.Errorf("mark arrived: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repo) MarkLost(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE messages SET status = 'lost', delivered_at = now()
		WHERE id = $1 AND status = 'in_transit'
	`, id)
	if err != nil {
		return fmt.Errorf("mark lost: %w", err)
	}
	return nil
}

func (r *Repo) SpeedUpMessage(ctx context.Context, id string, newArrivesAt time.Time) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE messages
		SET arrives_at = $2, speedups_used = speedups_used + 1
		WHERE id = $1 AND status = 'in_transit'
	`, id, newArrivesAt)
	if err != nil {
		return fmt.Errorf("speed up: %w", err)
	}
	return nil
}
