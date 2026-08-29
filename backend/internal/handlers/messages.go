package handlers

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/bayurstarcool/wingback/backend/internal/config"
	"github.com/bayurstarcool/wingback/backend/internal/delivery"
)

// MessageHandler wires HTTP routes to the delivery engine.
// A real implementation persists to Postgres and pushes state over
// WebSocket; this scaffold focuses on the delivery-computation contract
// so the frontend can integrate against a stable response shape early.
type MessageHandler struct {
	cfg *config.Config
	rng *rand.Rand
}

func NewMessageHandler(cfg *config.Config) *MessageHandler {
	return &MessageHandler{
		cfg: cfg,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

type composeRequest struct {
	RecipientID  string  `json:"recipient_id" validate:"required"`
	Body         string  `json:"body" validate:"required,max=2000"`
	CarrierSlug  string  `json:"carrier_slug"`
	SenderLat    float64 `json:"sender_lat" validate:"required"`
	SenderLng    float64 `json:"sender_lng" validate:"required"`
	RecipientLat float64 `json:"recipient_lat" validate:"required"`
	RecipientLng float64 `json:"recipient_lng" validate:"required"`
}

type composeResponse struct {
	MessageID  string    `json:"message_id"`
	DistanceKM float64   `json:"distance_km"`
	SpeedKMH   float64   `json:"speed_kmh"`
	DepartsAt  time.Time `json:"departs_at"`
	ArrivesAt  time.Time `json:"arrives_at"`
	WillBeLost bool      `json:"will_be_lost"`
}

// Compose handles POST /api/messages — creates a message and computes
// its delivery plan immediately so the client can render a live ETA/map.
func (h *MessageHandler) Compose(c echo.Context) error {
	var req composeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Body == "" || req.RecipientID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "recipient_id and body are required")
	}

	speed := h.cfg.DefaultCarrierSpeedKMH // TODO: look up carrier by slug from DB

	from := delivery.Coordinates{Lat: req.SenderLat, Lng: req.SenderLng}
	to := delivery.Coordinates{Lat: req.RecipientLat, Lng: req.RecipientLng}

	plan := delivery.Compute(from, to, speed, h.cfg.MessageLossProbability, time.Now(), h.rng)

	// TODO: persist message + plan to Postgres, publish initial state to Redis/WebSocket hub.

	return c.JSON(http.StatusCreated, composeResponse{
		MessageID:  uuid.NewString(),
		DistanceKM: plan.DistanceKM,
		SpeedKMH:   plan.SpeedKMH,
		DepartsAt:  plan.DepartsAt,
		ArrivesAt:  plan.ArrivesAt,
		WillBeLost: plan.WillBeLost,
	})
}
