// Package hub is a tiny in-process pub/sub for live message state. Each
// in_transit message gets a channel; clients subscribe to receive
// position updates as the carrier "flies". For multi-instance deploys
// this would back onto Redis pub/sub, but the API surface is identical
// so the swap is one struct's worth of code.
package hub

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Type     string    `json:"type"` // "position", "arrived", "lost"
	MessageID string   `json:"message_id"`
	Lat       float64  `json:"lat"`
	Lng       float64  `json:"lng"`
	At        time.Time `json:"at"`
}

type subscriber struct {
	id     string
	out    chan Event
}

type Hub struct {
	mu     sync.RWMutex
	subs   map[string][]subscriber // messageID -> subscribers
}

func New() *Hub {
	return &Hub{subs: make(map[string][]subscriber)}
}

func (h *Hub) Subscribe(messageID string) (string, <-chan Event, func()) {
	id := uuid.NewString()
	out := make(chan Event, 16)
	h.mu.Lock()
	h.subs[messageID] = append(h.subs[messageID], subscriber{id: id, out: out})
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		list := h.subs[messageID]
		for i, s := range list {
			if s.id == id {
				h.subs[messageID] = append(list[:i], list[i+1:]...)
				close(out)
				return
			}
		}
	}
	return id, out, cancel
}

func (h *Hub) Publish(messageID string, e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, s := range h.subs[messageID] {
		select {
		case s.out <- e:
		default:
			// drop if subscriber is too slow; non-blocking
		}
	}
}
