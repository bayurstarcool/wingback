package handlers

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/bayurstarcool/wingback/backend/internal/config"
	"github.com/bayurstarcool/wingback/backend/internal/delivery"
	"github.com/bayurstarcool/wingback/backend/internal/hub"
	"github.com/bayurstarcool/wingback/backend/internal/middleware"
	"github.com/bayurstarcool/wingback/backend/internal/models"
	"github.com/bayurstarcool/wingback/backend/internal/repo"
	"github.com/bayurstarcool/wingback/backend/internal/tracker"
	"github.com/gorilla/websocket"
)

type MessageHandler struct {
	cfg  *config.Config
	repo *repo.Repo
	hub  *hub.Hub
	rng  *rand.Rand
}

func NewMessageHandler(cfg *config.Config, r *repo.Repo, h *hub.Hub) *MessageHandler {
	return &MessageHandler{
		cfg:  cfg,
		repo: r,
		hub:  h,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
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
	Carrier    string    `json:"carrier"`
}

func (h *MessageHandler) Compose(c echo.Context) error {
	uid := middleware.UserID(c)
	if uid == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing auth")
	}

	var req composeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Body == "" || req.RecipientID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "recipient_id and body are required")
	}

	// Look up carrier (default if not specified).
	carrier, err := h.repo.GetCarrierBySlug(c.Request().Context(), strings.TrimSpace(req.CarrierSlug))
	if err != nil {
		carrier, err = h.repo.GetDefaultCarrier(c.Request().Context())
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "no carrier available")
		}
	}

	from := delivery.Coordinates{Lat: req.SenderLat, Lng: req.SenderLng}
	to := delivery.Coordinates{Lat: req.RecipientLat, Lng: req.RecipientLng}

	plan := delivery.Compute(from, to, carrier.SpeedKMH, h.cfg.MessageLossProbability, time.Now(), h.rng)

	// Reject sending to self.
	if req.RecipientID == uid {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot send a message to yourself")
	}

	// Verify recipient exists.
	if _, err := h.repo.GetUserByID(c.Request().Context(), req.RecipientID); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "recipient not found")
	}

	msg := &models.Message{
		ID:          uuid.NewString(),
		SenderID:    uid,
		RecipientID: req.RecipientID,
		CarrierID:   carrier.ID,
		Body:        req.Body,
		SenderLat:   req.SenderLat,
		SenderLng:   req.SenderLng,
		RecLat:      req.RecipientLat,
		RecLng:      req.RecipientLng,
		DistanceKM:  plan.DistanceKM,
		SpeedKMH:    plan.SpeedKMH,
		DepartsAt:   plan.DepartsAt,
		ArrivesAt:   plan.ArrivesAt,
	}
	if plan.WillBeLost {
		msg.Status = models.StatusInTransit // will be marked lost by tracker
	} else {
		msg.Status = models.StatusInTransit
	}

	if err := h.repo.CreateMessage(c.Request().Context(), msg); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Kick off live tracking for this message.
	tracker.Fly(c.Request().Context(), h.repo, h.hub, msg, plan.WillBeLost)

	return c.JSON(http.StatusCreated, composeResponse{
		MessageID:  msg.ID,
		DistanceKM: msg.DistanceKM,
		SpeedKMH:   msg.SpeedKMH,
		DepartsAt:  msg.DepartsAt,
		ArrivesAt:  msg.ArrivesAt,
		WillBeLost: plan.WillBeLost,
		Carrier:    carrier.Slug,
	})
}

type messageDTO struct {
	ID          string     `json:"id"`
	SenderID    string     `json:"sender_id"`
	RecipientID string     `json:"recipient_id"`
	Body        string     `json:"body"`
	DistanceKM  float64    `json:"distance_km"`
	SpeedKMH    float64    `json:"speed_kmh"`
	Status      string     `json:"status"`
	DepartsAt   time.Time  `json:"departs_at"`
	ArrivesAt   time.Time  `json:"arrives_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

func toDTO(m models.Message) messageDTO {
	return messageDTO{
		ID:          m.ID,
		SenderID:    m.SenderID,
		RecipientID: m.RecipientID,
		Body:        m.Body,
		DistanceKM:  m.DistanceKM,
		SpeedKMH:    m.SpeedKMH,
		Status:      string(m.Status),
		DepartsAt:   m.DepartsAt,
		ArrivesAt:   m.ArrivesAt,
		DeliveredAt: m.DeliveredAt,
	}
}

func (h *MessageHandler) ListInbox(c echo.Context) error {
	uid := middleware.UserID(c)
	if uid == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing auth")
	}
	msgs, err := h.repo.ListInbox(c.Request().Context(), uid, 50)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	out := make([]messageDTO, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, toDTO(m))
	}
	return c.JSON(http.StatusOK, out)
}

func (h *MessageHandler) ListSent(c echo.Context) error {
	uid := middleware.UserID(c)
	if uid == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing auth")
	}
	msgs, err := h.repo.ListSent(c.Request().Context(), uid, 50)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	out := make([]messageDTO, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, toDTO(m))
	}
	return c.JSON(http.StatusOK, out)
}

func (h *MessageHandler) GetMessage(c echo.Context) error {
	uid := middleware.UserID(c)
	if uid == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing auth")
	}
	id := c.Param("id")
	m, err := h.repo.GetMessage(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "message not found")
	}
	if m.SenderID != uid && m.RecipientID != uid {
		return echo.NewHTTPError(http.StatusForbidden, "not your message")
	}
	return c.JSON(http.StatusOK, toDTO(*m))
}

func (h *MessageHandler) ListCarriers(c echo.Context) error {
	carriers, err := h.repo.ListCarriers(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	type carrierDTO struct {
		Slug     string  `json:"slug"`
		Name     string  `json:"name"`
		SpeedKMH float64 `json:"speed_kmh"`
		Price    int     `json:"price"`
		Rarity   string  `json:"rarity"`
	}
	out := make([]carrierDTO, 0, len(carriers))
	for _, c := range carriers {
		out = append(out, carrierDTO{
			Slug:     c.Slug,
			Name:     c.Name,
			SpeedKMH: c.SpeedKMH,
			Price:    c.Price,
			Rarity:   c.Rarity,
		})
	}
	return c.JSON(http.StatusOK, out)
}

func (h *MessageHandler) UpdateLocation(c echo.Context) error {
	uid := middleware.UserID(c)
	if uid == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing auth")
	}
	var req struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		return echo.NewHTTPError(http.StatusBadRequest, "lat must be [-90,90], lng [-180,180]")
	}
	if err := h.repo.UpdateUserLocation(c.Request().Context(), uid, req.Lat, req.Lng); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// --- WebSocket streaming of live position updates ---

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // dev: allow any origin
}

func (h *MessageHandler) Stream(c echo.Context) error {
	uid := middleware.UserID(c)
	if uid == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing auth")
	}
	id := c.Param("id")
	m, err := h.repo.GetMessage(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "message not found")
	}
	if m.SenderID != uid && m.RecipientID != uid {
		return echo.NewHTTPError(http.StatusForbidden, "not your message")
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, events, cancel := h.hub.Subscribe(id)
	defer cancel()

	// Emit current known position immediately so the client doesn't
	// have to wait up to tickInterval to see *something*.
	if m.Status == models.StatusInTransit {
		now := time.Now()
		if now.Before(m.ArrivesAt) {
			frac := float64(now.Sub(m.DepartsAt)) / float64(m.ArrivesAt.Sub(m.DepartsAt))
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			lat, lng := interpolate(
				delivery.Coordinates{Lat: m.SenderLat, Lng: m.SenderLng},
				delivery.Coordinates{Lat: m.RecLat, Lng: m.RecLng},
				frac,
			)
			payload, _ := json.Marshal(hub.Event{
				Type: "position", MessageID: m.ID, Lat: lat, Lng: lng, At: now,
			})
			_ = conn.WriteMessage(websocket.TextMessage, payload)
		}
	} else {
		payload, _ := json.Marshal(hub.Event{
			Type: string(m.Status), MessageID: m.ID,
			Lat: m.RecLat, Lng: m.RecLng, At: time.Now(),
		})
		_ = conn.WriteMessage(websocket.TextMessage, payload)
	}

	// Read loop just to detect client disconnect.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return nil
		case e := <-events:
			payload, _ := json.Marshal(e)
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return nil
			}
		}
	}
}

func interpolate(from, to delivery.Coordinates, frac float64) (float64, float64) {
	lat := from.Lat + (to.Lat-from.Lat)*frac
	lng := from.Lng + (to.Lng-from.Lng)*frac
	return lat, lng
}
