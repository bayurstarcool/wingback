package handlers

import (
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/bayurstarcool/wingback/backend/internal/config"
	"github.com/bayurstarcool/wingback/backend/internal/delivery"
	"github.com/bayurstarcool/wingback/backend/internal/hub"
	"github.com/bayurstarcool/wingback/backend/internal/location"
	"github.com/bayurstarcool/wingback/backend/internal/middleware"
	"github.com/bayurstarcool/wingback/backend/internal/models"
	"github.com/bayurstarcool/wingback/backend/internal/repo"
	"github.com/bayurstarcool/wingback/backend/internal/tracker"
	"github.com/gorilla/websocket"
)

type MessageHandler struct {
	cfg          *config.Config
	repo         *repo.Repo
	hub          *hub.Hub
	rng          *rand.Rand
	cityResolver location.Resolver
}

func NewMessageHandler(cfg *config.Config, r *repo.Repo, h *hub.Hub) *MessageHandler {
	return &MessageHandler{
		cfg:          cfg,
		repo:         r,
		hub:          h,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
		cityResolver: location.NewNominatimResolver(cfg.GeocoderURL),
	}
}

func NewMessageHandlerWithResolver(cfg *config.Config, r *repo.Repo, h *hub.Hub, resolver location.Resolver) *MessageHandler {
	result := NewMessageHandler(cfg, r, h)
	result.cityResolver = resolver
	return result
}

type composeRequest struct {
	RecipientID     string  `json:"recipient_id" validate:"required"`
	Body            string  `json:"body" validate:"required,max=2000"`
	CarrierSlug     string  `json:"carrier_slug"`
	LocationPrivacy string  `json:"location_privacy"`
	SenderLat       float64 `json:"sender_lat" validate:"required"`
	SenderLng       float64 `json:"sender_lng" validate:"required"`
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
	if req.LocationPrivacy == "" {
		req.LocationPrivacy = models.LocationPrivacyAccurate
	}
	if req.LocationPrivacy != models.LocationPrivacyAccurate && req.LocationPrivacy != models.LocationPrivacyHidden {
		return echo.NewHTTPError(http.StatusBadRequest, "location_privacy must be accurate or hidden")
	}
	recipient, err := h.repo.GetUserByID(c.Request().Context(), req.RecipientID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "recipient not found")
	}
	if recipient.LastLat == nil || recipient.LastLng == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "recipient has not set a delivery location")
	}
	if recipient.LastLocationAt == nil || time.Since(*recipient.LastLocationAt) > 30*24*time.Hour {
		return echo.NewHTTPError(http.StatusBadRequest, "recipient delivery location is outdated")
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
	to := delivery.Coordinates{Lat: *recipient.LastLat, Lng: *recipient.LastLng}
	var senderCity, recipientCity location.City
	if req.LocationPrivacy == models.LocationPrivacyHidden {
		if h.cityResolver == nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "private city labels unavailable")
		}
		senderCity, err = h.cityResolver.ResolveCity(c.Request().Context(), req.SenderLat, req.SenderLng)
		if err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "private city label unavailable for sender location")
		}
		recipientCity, err = h.cityResolver.ResolveCity(c.Request().Context(), *recipient.LastLat, *recipient.LastLng)
		if err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "private city label unavailable for recipient location")
		}
	}

	plan := delivery.Compute(from, to, carrier.SpeedKMH, h.cfg.MessageLossProbability, time.Now(), h.rng)

	// Reject sending to self.
	if req.RecipientID == uid {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot send a message to yourself")
	}

	msg := &models.Message{
		ID:               uuid.NewString(),
		SenderID:         uid,
		RecipientID:      req.RecipientID,
		CarrierID:        carrier.ID,
		Body:             req.Body,
		SenderLat:        req.SenderLat,
		SenderLng:        req.SenderLng,
		RecLat:           *recipient.LastLat,
		RecLng:           *recipient.LastLng,
		DistanceKM:       plan.DistanceKM,
		SpeedKMH:         plan.SpeedKMH,
		DepartsAt:        plan.DepartsAt,
		ArrivesAt:        plan.ArrivesAt,
		LocationPrivacy:  req.LocationPrivacy,
		SenderCity:       senderCity.Name,
		RecipientCity:    recipientCity.Name,
		SenderCityLat:    cityFloat(coarseCityCoordinate(senderCity.Lat)),
		SenderCityLng:    cityFloat(coarseCityCoordinate(senderCity.Lng)),
		RecipientCityLat: cityFloat(coarseCityCoordinate(recipientCity.Lat)),
		RecipientCityLng: cityFloat(coarseCityCoordinate(recipientCity.Lng)),
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
	ID               string     `json:"id"`
	SenderID         string     `json:"sender_id"`
	RecipientID      string     `json:"recipient_id"`
	Body             string     `json:"body"`
	SenderLat        *float64   `json:"sender_lat,omitempty"`
	SenderLng        *float64   `json:"sender_lng,omitempty"`
	RecipientLat     *float64   `json:"recipient_lat,omitempty"`
	RecipientLng     *float64   `json:"recipient_lng,omitempty"`
	LocationPrivacy  string     `json:"location_privacy"`
	SenderCity       string     `json:"from_label,omitempty"`
	RecipientCity    string     `json:"to_label,omitempty"`
	SameCity         bool       `json:"same_city"`
	SenderCityLat    *float64   `json:"from_map_lat,omitempty"`
	SenderCityLng    *float64   `json:"from_map_lng,omitempty"`
	RecipientCityLat *float64   `json:"to_map_lat,omitempty"`
	RecipientCityLng *float64   `json:"to_map_lng,omitempty"`
	DistanceKM       float64    `json:"distance_km"`
	SpeedKMH         float64    `json:"speed_kmh"`
	Status           string     `json:"status"`
	DepartsAt        time.Time  `json:"departs_at"`
	ArrivesAt        time.Time  `json:"arrives_at"`
	DeliveredAt      *time.Time `json:"delivered_at,omitempty"`
}

func publicRoute(m models.Message) (*float64, *float64, *float64, *float64) {
	if m.LocationPrivacy == models.LocationPrivacyHidden {
		return nil, nil, nil, nil
	}
	return &m.SenderLat, &m.SenderLng, &m.RecLat, &m.RecLng
}

func cityFloat(value float64) *float64 {
	if value == 0 {
		return nil
	}
	return &value
}

// coarseCityCoordinate keeps private map anchors at city-area precision. It
// is deliberately not the sender or recipient GPS coordinate.
func coarseCityCoordinate(value float64) float64 {
	return math.Round(value*10) / 10
}

type streamEvent struct {
	Type      string    `json:"type"`
	MessageID string    `json:"message_id"`
	Lat       *float64  `json:"lat,omitempty"`
	Lng       *float64  `json:"lng,omitempty"`
	Progress  float64   `json:"progress,omitempty"`
	Phase     string    `json:"phase,omitempty"`
	At        time.Time `json:"at"`
}

func streamPayload(m models.Message, e hub.Event) streamEvent {
	if m.LocationPrivacy != models.LocationPrivacyHidden {
		return streamEvent{Type: e.Type, MessageID: e.MessageID, Lat: &e.Lat, Lng: &e.Lng, At: e.At}
	}
	progress := 0.0
	if !m.DepartsAt.IsZero() && m.ArrivesAt.After(m.DepartsAt) {
		progress = float64(e.At.Sub(m.DepartsAt)) / float64(m.ArrivesAt.Sub(m.DepartsAt))
	}
	if e.Type == "arrived" || e.Type == "delivered" {
		progress = 1
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	phase := "Berangkat"
	if progress >= 0.72 {
		phase = "Mendekati tujuan"
	} else if progress >= 0.18 {
		phase = "Sedang melintas"
	}
	if e.Type == "lost" {
		phase = "Perjalanan terhenti"
	}
	return streamEvent{Type: "progress", MessageID: e.MessageID, Progress: progress, Phase: phase, At: e.At}
}

func toDTO(m models.Message) messageDTO {
	senderLat, senderLng, recipientLat, recipientLng := publicRoute(m)
	return messageDTO{
		ID:               m.ID,
		SenderID:         m.SenderID,
		RecipientID:      m.RecipientID,
		Body:             m.Body,
		SenderLat:        senderLat,
		SenderLng:        senderLng,
		RecipientLat:     recipientLat,
		RecipientLng:     recipientLng,
		LocationPrivacy:  m.LocationPrivacy,
		SenderCity:       m.SenderCity,
		RecipientCity:    m.RecipientCity,
		SameCity:         location.SameCity(m.SenderCity, m.RecipientCity),
		SenderCityLat:    privateCityCoordinate(m, m.SenderCityLat),
		SenderCityLng:    privateCityCoordinate(m, m.SenderCityLng),
		RecipientCityLat: privateCityCoordinate(m, m.RecipientCityLat),
		RecipientCityLng: privateCityCoordinate(m, m.RecipientCityLng),
		DistanceKM:       m.DistanceKM,
		SpeedKMH:         m.SpeedKMH,
		Status:           string(m.Status),
		DepartsAt:        m.DepartsAt,
		ArrivesAt:        m.ArrivesAt,
		DeliveredAt:      m.DeliveredAt,
	}
}

func privateCityCoordinate(m models.Message, value *float64) *float64 {
	if m.LocationPrivacy != models.LocationPrivacyHidden {
		return nil
	}
	return value
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

func (h *MessageHandler) SearchUsers(c echo.Context) error {
	query := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(c.QueryParam("q")), "@"))
	if len(query) < 2 {
		return c.JSON(http.StatusOK, []any{})
	}
	users, err := h.repo.SearchUsers(c.Request().Context(), query, 8)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	type userDTO struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		DisplayName   string `json:"display_name"`
		Email         string `json:"email"`
		LocationReady bool   `json:"location_ready"`
	}
	out := make([]userDTO, 0, len(users))
	for _, u := range users {
		ready := u.LastLat != nil && u.LastLng != nil && u.LastLocationAt != nil && time.Since(*u.LastLocationAt) <= 30*24*time.Hour
		out = append(out, userDTO{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Email: maskEmail(u.Email), LocationReady: ready})
	}
	return c.JSON(http.StatusOK, out)
}

func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) < 2 {
		return email
	}
	return parts[0][:1] + "***@" + parts[1]
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
		Lat       float64  `json:"lat"`
		Lng       float64  `json:"lng"`
		AccuracyM *float64 `json:"accuracy_m"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		return echo.NewHTTPError(http.StatusBadRequest, "lat must be [-90,90], lng [-180,180]")
	}
	if err := h.repo.UpdateUserLocation(c.Request().Context(), uid, req.Lat, req.Lng, req.AccuracyM); err != nil {
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
			e := streamPayload(*m, hub.Event{Type: "position", MessageID: m.ID, Lat: lat, Lng: lng, At: now})
			payload, _ := json.Marshal(e)
			_ = conn.WriteMessage(websocket.TextMessage, payload)
		}
	} else {
		e := streamPayload(*m, hub.Event{Type: string(m.Status), MessageID: m.ID, Lat: m.RecLat, Lng: m.RecLng, At: time.Now()})
		payload, _ := json.Marshal(e)
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
		case e, ok := <-events:
			if !ok {
				return nil
			}
			payload, _ := json.Marshal(streamPayload(*m, e))
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
