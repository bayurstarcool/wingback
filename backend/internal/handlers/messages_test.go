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

// We don't have a real repo or hub wired here, so Compose will return
// 500 when it tries to use them. The test verifies the auth contract:
// unauthenticated requests are rejected with 401, not 500.
func TestCompose_RequiresAuth(t *testing.T) {
	e := echo.New()
	cfg := &config.Config{}
	h := NewMessageHandler(cfg, nil, nil) // nil repo/hub is fine — auth runs first

	reqBody := composeRequest{
		RecipientID:  "r",
		Body:         "hi",
		SenderLat:    -6, SenderLng: 107,
		RecipientLat: -7, RecipientLng: 112,
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
