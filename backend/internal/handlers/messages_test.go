package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/bayurstarcool/wingback/backend/internal/config"
)

func newTestEcho() (*echo.Echo, *MessageHandler) {
	cfg := &config.Config{
		DefaultCarrierSpeedKMH: 177,
		MessageLossProbability: 0, // deterministic for tests
	}
	h := NewMessageHandler(cfg)
	e := echo.New()
	return e, h
}

func TestCompose_ValidRequest_Returns201(t *testing.T) {
	e, h := newTestEcho()

	reqBody := composeRequest{
		RecipientID:  "recipient-123",
		Body:         "Halo dari Jakarta!",
		SenderLat:    -6.2088,
		SenderLng:    106.8456,
		RecipientLat: -7.2575,
		RecipientLng: 112.7521,
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Compose(c); err != nil {
		t.Fatalf("Compose returned error: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var resp composeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v, body=%s", err, rec.Body.String())
	}

	if resp.MessageID == "" {
		t.Error("expected non-empty message_id")
	}
	// Jakarta -> Surabaya is roughly 600-700km great circle.
	if resp.DistanceKM < 600 || resp.DistanceKM > 700 {
		t.Errorf("expected distance ~600-700km, got %f", resp.DistanceKM)
	}
	if resp.SpeedKMH != 177 {
		t.Errorf("expected default speed 177, got %f", resp.SpeedKMH)
	}
	if !resp.ArrivesAt.After(resp.DepartsAt) {
		t.Error("expected ArrivesAt to be after DepartsAt")
	}
	if resp.WillBeLost {
		t.Error("expected no loss with probability 0")
	}
}

func TestCompose_MissingBody_Returns400(t *testing.T) {
	e, h := newTestEcho()

	reqBody := composeRequest{
		RecipientID: "recipient-123",
		// Body intentionally omitted
		SenderLat: -6.2088, SenderLng: 106.8456,
		RecipientLat: -7.2575, RecipientLng: 112.7521,
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
	if he.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", he.Code)
	}
}

func TestCompose_MissingRecipient_Returns400(t *testing.T) {
	e, h := newTestEcho()

	reqBody := composeRequest{
		Body:      "Halo!",
		SenderLat: -6.2088, SenderLng: 106.8456,
		RecipientLat: -7.2575, RecipientLng: 112.7521,
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
	if he.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", he.Code)
	}
}

func TestCompose_InvalidJSON_Returns400(t *testing.T) {
	e, h := newTestEcho()

	req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader([]byte("not-json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Compose(c)
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T (%v)", err, err)
	}
	if he.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", he.Code)
	}
}
