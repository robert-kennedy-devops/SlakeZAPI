package events

import (
	"sync"

	"github.com/whatsapp-saas/api/internal/domain"
)

// Bus is a thread-safe in-process pub/sub event bus.
// It fans out published events to all subscribers of the same tenant.
type Bus struct {
	mu      sync.RWMutex
	subs    map[string][]chan domain.Event // tenantID → list of subscriber channels
	allSubs []chan domain.Event
}

// NewBus creates a ready-to-use event Bus.
func NewBus() *Bus {
	return &Bus{
		subs: make(map[string][]chan domain.Event),
	}
}

// Ensure Bus satisfies domain.EventBus.
var _ domain.EventBus = (*Bus)(nil)

// Publish sends an event to every subscriber registered for the event's tenant.
// It is non-blocking: slow subscribers are skipped.
func (b *Bus) Publish(event domain.Event) {
	b.mu.RLock()
	chans := b.subs[event.TenantID]
	allChans := append([]chan domain.Event(nil), b.allSubs...)
	b.mu.RUnlock()

	for _, ch := range chans {
		select {
		case ch <- event:
		default:
			// subscriber is not reading fast enough — drop the event
		}
	}
	for _, ch := range allChans {
		select {
		case ch <- event:
		default:
		}
	}
}

// Subscribe returns a channel that receives events for the given tenant and
// an unsubscribe function the caller must invoke when done.
func (b *Bus) Subscribe(tenantID string) (<-chan domain.Event, func()) {
	ch := make(chan domain.Event, 32)

	b.mu.Lock()
	b.subs[tenantID] = append(b.subs[tenantID], ch)
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		chans := b.subs[tenantID]
		for i, c := range chans {
			if c == ch {
				b.subs[tenantID] = append(chans[:i], chans[i+1:]...)
				close(ch)
				break
			}
		}
		if len(b.subs[tenantID]) == 0 {
			delete(b.subs, tenantID)
		}
	}

	return ch, unsubscribe
}

func (b *Bus) SubscribeAll() (<-chan domain.Event, func()) {
	ch := make(chan domain.Event, 64)

	b.mu.Lock()
	b.allSubs = append(b.allSubs, ch)
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		for i, candidate := range b.allSubs {
			if candidate == ch {
				b.allSubs = append(b.allSubs[:i], b.allSubs[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, unsubscribe
}
