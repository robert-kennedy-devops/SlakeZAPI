package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/events"
	"github.com/whatsapp-saas/api/internal/queue"
	"github.com/whatsapp-saas/api/pkg/logger"
)

func TestDispatcherDeliverWrapsEnvelopeAndSignsPayload(t *testing.T) {
	var (
		gotBody      []byte
		gotEvent     string
		gotWebhookID string
		gotSignature string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		gotBody = body
		gotEvent = r.Header.Get("X-Webhook-Event")
		gotWebhookID = r.Header.Get("X-Webhook-Id")
		gotSignature = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dispatcher := NewDispatcher(nil, events.NewBus(), queue.NewPool(1, 1, 1, logger.New()), time.Second, 0, logger.New())
	dispatcher.httpClient = srv.Client()

	evt := domain.Event{
		Type:     domain.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]any{
			"message_id": "msg-1",
			"type":       "image",
		},
	}
	wh := domain.Webhook{
		ID:     "wh-1",
		URL:    srv.URL,
		Secret: "super-secret",
		Active: true,
	}

	if err := dispatcher.Deliver(context.Background(), wh, evt); err != nil {
		t.Fatalf("deliver webhook: %v", err)
	}

	if gotEvent != string(evt.Type) {
		t.Fatalf("unexpected event header: %s", gotEvent)
	}
	if gotWebhookID == "" {
		t.Fatal("expected webhook id header")
	}
	if gotSignature != sign(gotBody, wh.Secret) {
		t.Fatalf("unexpected signature: %s", gotSignature)
	}

	var envelope domain.WebhookEnvelope
	if err := json.Unmarshal(gotBody, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if envelope.ID != gotWebhookID {
		t.Fatalf("header/body id mismatch: %s vs %s", gotWebhookID, envelope.ID)
	}
	if envelope.Version != "v1" || envelope.Type != evt.Type || envelope.TenantID != evt.TenantID {
		t.Fatalf("unexpected envelope metadata: %+v", envelope)
	}
	if envelope.Timestamp.IsZero() {
		t.Fatal("expected timestamp in envelope")
	}
}
