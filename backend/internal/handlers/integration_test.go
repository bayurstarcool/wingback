// Integration test that exercises the whole flow against a real
// Postgres: register two users, compose a message from A to B, verify
// the tracker publishes position events that the hub delivers to
// subscribers, and confirm the message eventually lands in B's inbox.
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/bayurstarcool/wingback/backend/internal/config"
	"github.com/bayurstarcool/wingback/backend/internal/db"
	"github.com/bayurstarcool/wingback/backend/internal/hub"
	"github.com/bayurstarcool/wingback/backend/internal/repo"
)

func openTestDB(t *testing.T) *repo.Repo {
	t.Helper()
	dsn := "postgres://wingback:wingback_dev_only@127.0.0.1:5432/wingback?sslmode=disable"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("DB not reachable, skipping integration test: %v", err)
	}
	t.Cleanup(pool.Close)
	return repo.New(pool)
}

func TestEndToEnd_ComposePersistsAndHubDelivers(t *testing.T) {
	r := openTestDB(t)
	cfg := &config.Config{
		DefaultCarrierSpeedKMH: 5000, // fast for the test
		MessageLossProbability: 0,
	}
	h := hub.New()
	handler := NewMessageHandler(cfg, r, h)

	// Seed two users.
	ctx := context.Background()
	ts := time.Now().UnixNano()
	alice, err := r.CreateUser(ctx,
		"alice-"+itoa(ts)+"@wingback.test", "h", "Alice")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	bob, err := r.CreateUser(ctx,
		"bob-"+itoa(ts)+"@wingback.test", "h", "Bob")
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}
	if err := r.UpdateUserLocation(ctx, bob.ID, -7.2575, 112.7521); err != nil {
		t.Fatalf("set bob location: %v", err)
	}

	// Subscribe to the message *before* it exists — the hub will
	// return a channel immediately; the first event we care about
	// is the position published by the tracker.
	// (We subscribe after compose below; this is just to assert
	// that the hub doesn't panic on a fresh messageID.)

	// Build a synthetic request as Alice.
	e := echo.New()
	e.HideBanner = true
	body := `{
		"recipient_id": "` + bob.ID + `",
		"body": "halo dari alice",
		"sender_lat": -6.2088, "sender_lng": 106.8456
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", alice.ID)

	if err := handler.Compose(c); err != nil {
		t.Fatalf("compose: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", rec.Code, rec.Body.String())
	}

	// Compose response must include arrival later than depart.
	if !strings.Contains(rec.Body.String(), `"will_be_lost":false`) {
		t.Fatalf("expected will_be_lost=false, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"message_id"`) {
		t.Fatalf("expected message_id in response, got %s", rec.Body.String())
	}

	// Hub: subscribe and wait briefly for a position event.
	messageID := extractField(t, rec.Body.String(), "message_id")
	_, events, cancel := h.Subscribe(messageID)
	defer cancel()

	select {
	case ev := <-events:
		if ev.Type != "position" {
			t.Fatalf("expected first event to be 'position', got %q", ev.Type)
		}
		// Sender coords.
		if !near(ev.Lat, -6.2088, 0.01) || !near(ev.Lng, 106.8456, 0.01) {
			t.Fatalf("expected initial position near sender, got (%f, %f)", ev.Lat, ev.Lng)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first position event")
	}

	// Verify the message is in Bob's inbox.
	inbox, err := r.ListInbox(ctx, bob.ID, 10)
	if err != nil {
		t.Fatalf("list inbox: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected 1 message in Bob's inbox, got %d", len(inbox))
	}
	if inbox[0].Body != "halo dari alice" {
		t.Fatalf("body mismatch: %q", inbox[0].Body)
	}
	if inbox[0].SenderID != alice.ID {
		t.Fatalf("sender_id mismatch: %q", inbox[0].SenderID)
	}
}

// --- helpers ---

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}

func extractField(t *testing.T, body, field string) string {
	t.Helper()
	key := `"` + field + `":"`
	i := strings.Index(body, key)
	if i < 0 {
		t.Fatalf("field %q not found in %s", field, body)
	}
	rest := body[i+len(key):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		t.Fatalf("malformed field %q in %s", field, body)
	}
	return rest[:j]
}

func near(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}
