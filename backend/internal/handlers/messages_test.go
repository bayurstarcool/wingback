package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/bayurstarcool/wingback/backend/internal/config"
	"github.com/bayurstarcool/wingback/backend/internal/hub"
	"github.com/bayurstarcool/wingback/backend/internal/models"
)

// We don't have a real repo or hub wired here, so Compose will return
// 500 when it tries to use them. The test verifies the auth contract:
// unauthenticated requests are rejected with 401, not 500.
func TestCompose_RequiresAuth(t *testing.T) {
	e := echo.New()
	cfg := &config.Config{}
	h := NewMessageHandler(cfg, nil, nil) // nil repo/hub is fine — auth runs first

	reqBody := composeRequest{
		RecipientID: "r",
		Body:        "hi",
		SenderLat:   -6, SenderLng: 107,
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Compose(c)
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T (%v)", err, err)
	}
	if he.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", he.Code)
	}
}

func TestListInbox_RequiresAuth(t *testing.T) {
	e := echo.New()
	h := NewMessageHandler(&config.Config{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/messages/inbox", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListInbox(c)
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T (%v)", err, err)
	}
	if he.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", he.Code)
	}
}

func TestStreamPayload_HiddenDoesNotExposeCoordinates(t *testing.T) {
	now := time.Now()
	m := models.Message{ID: "hidden", LocationPrivacy: models.LocationPrivacyHidden, DepartsAt: now.Add(-30 * time.Minute), ArrivesAt: now.Add(30 * time.Minute)}
	payload := streamPayload(m, hub.Event{Type: "position", MessageID: m.ID, Lat: -7.9, Lng: 112.6, At: now})
	if payload.Lat != nil || payload.Lng != nil {
		t.Fatalf("hidden payload exposed coordinates: lat=%v lng=%v", payload.Lat, payload.Lng)
	}
	if payload.Type != "progress" || payload.Phase == "" || payload.Progress <= 0 || payload.Progress >= 1 {
		t.Fatalf("unexpected hidden progress payload: %+v", payload)
	}
}

func TestStreamPayload_AccurateKeepsCoordinates(t *testing.T) {
	m := models.Message{ID: "accurate", LocationPrivacy: models.LocationPrivacyAccurate}
	payload := streamPayload(m, hub.Event{Type: "position", MessageID: m.ID, Lat: -7.9, Lng: 112.6, At: time.Now()})
	if payload.Type != "position" || payload.Lat == nil || payload.Lng == nil || *payload.Lat != -7.9 || *payload.Lng != 112.6 {
		t.Fatalf("accurate payload lost coordinates: %+v", payload)
	}
}

func TestToDTO_PrivateUsesCityLabelsOnly(t *testing.T) {
	m := models.Message{
		ID:              "private-city",
		LocationPrivacy: models.LocationPrivacyHidden,
		SenderLat:       -7.9839,
		SenderLng:       112.6214,
		RecLat:          -7.2575,
		RecLng:          112.7521,
		SenderCity:      "Malang",
		RecipientCity:   "Surabaya",
	}
	dto := toDTO(m)
	if dto.SenderLat != nil || dto.SenderLng != nil || dto.RecipientLat != nil || dto.RecipientLng != nil {
		t.Fatal("private DTO exposed coordinates")
	}
	if dto.SenderCity != "Malang" || dto.RecipientCity != "Surabaya" || dto.SameCity {
		t.Fatalf("unexpected city labels: %+v", dto)
	}

	m.RecipientCity = "malang"
	dto = toDTO(m)
	if !dto.SameCity || dto.RecipientCity != "malang" {
		t.Fatalf("same-city flag incorrect: %+v", dto)
	}
}
