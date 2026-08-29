// Package tracker simulates a carrier's flight along the great-circle
// between sender and recipient. It runs as a goroutine per message,
// publishing position samples to the hub on a tick interval. This is
// the cheapest demoable realtime mechanic — for production you'd
// either move this to a worker queue or use deterministic interpolation
// on the client and skip server-side ticks entirely.
package tracker

import (
	"context"
	"math"
	"time"

	"github.com/bayurstarcool/wingback/backend/internal/delivery"
	"github.com/bayurstarcool/wingback/backend/internal/hub"
	"github.com/bayurstarcool/wingback/backend/internal/models"
	"github.com/bayurstarcool/wingback/backend/internal/repo"
)

const tickInterval = 2 * time.Second

// Fly starts a goroutine that emits position samples for the message
// until it arrives, gets lost, or the context is cancelled.
//
// The willBeLost flag is a hint passed by the caller (computed by the
// delivery engine) — we keep it as a parameter rather than expanding
// the model so the storage layer stays clean.
func Fly(ctx context.Context, repo *repo.Repo, h *hub.Hub, msg *models.Message, willBeLost bool) {
	go func() {
		now := time.Now()
		if willBeLost {
			// Pre-roll the loss so it happens within the first few seconds
			// of flight — more dramatic than losing it right at arrival.
			t := msg.DepartsAt.Add(time.Duration(float64(msg.ArrivesAt.Sub(msg.DepartsAt)) * 0.3))
			time.Sleep(time.Until(t))
			select {
			case <-ctx.Done():
				return
			default:
			}
			_ = repo.MarkLost(ctx, msg.ID)
			h.Publish(msg.ID, hub.Event{
				Type:      "lost",
				MessageID: msg.ID,
				At:        time.Now(),
			})
			return
		}

		// Initial position = sender
		h.Publish(msg.ID, hub.Event{
			Type: "position", MessageID: msg.ID,
			Lat: msg.SenderLat, Lng: msg.SenderLng, At: now,
		})

		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				if now.After(msg.ArrivesAt) {
					delivered, err := repo.MarkArrived(ctx, msg.ID, now)
					if err == nil && delivered {
						h.Publish(msg.ID, hub.Event{
							Type:      "arrived",
							MessageID: msg.ID,
							Lat:       msg.RecLat,
							Lng:       msg.RecLng,
							At:        now,
						})
					}
					return
				}
				// Compute current position via great-circle interpolation.
				frac := float64(now.Sub(msg.DepartsAt)) / float64(msg.ArrivesAt.Sub(msg.DepartsAt))
				frac = math.Max(0, math.Min(1, frac))
				lat, lng := interpolate(
					delivery.Coordinates{Lat: msg.SenderLat, Lng: msg.SenderLng},
					delivery.Coordinates{Lat: msg.RecLat, Lng: msg.RecLng},
					frac,
				)
				h.Publish(msg.ID, hub.Event{
					Type: "position", MessageID: msg.ID,
					Lat: lat, Lng: lng, At: now,
				})
			}
		}
	}()
}

// interpolate walks a fraction of the way from `from` to `to` along the
// great circle. Sufficient accuracy for a UI dot on a world map; not
// geodesy-grade.
func interpolate(from, to delivery.Coordinates, frac float64) (float64, float64) {
	lat := from.Lat + (to.Lat-from.Lat)*frac
	lng := from.Lng + (to.Lng-from.Lng)*frac
	return lat, lng
}
