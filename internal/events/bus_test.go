package events

import (
	"testing"

	"github.com/whatsapp-saas/api/internal/domain"
)

func TestBusSubscribeAllReceivesTenantEvents(t *testing.T) {
	bus := NewBus()

	tenantCh, tenantUnsub := bus.Subscribe("tenant-1")
	defer tenantUnsub()

	allCh, allUnsub := bus.SubscribeAll()
	defer allUnsub()

	event := domain.Event{
		Type:     domain.EventMessageSent,
		TenantID: "tenant-1",
		Payload:  "hello",
	}
	bus.Publish(event)

	select {
	case got := <-tenantCh:
		if got.TenantID != event.TenantID {
			t.Fatalf("unexpected tenant event: %+v", got)
		}
	default:
		t.Fatal("expected tenant subscriber to receive event")
	}

	select {
	case got := <-allCh:
		if got.Type != event.Type {
			t.Fatalf("unexpected global event: %+v", got)
		}
	default:
		t.Fatal("expected global subscriber to receive event")
	}
}
