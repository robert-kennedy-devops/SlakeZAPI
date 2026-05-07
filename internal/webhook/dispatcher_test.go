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

type stubWebhookRepo struct {
	deliveries map[string]*domain.WebhookDelivery
}

func (r *stubWebhookRepo) Create(ctx context.Context, wh *domain.Webhook) error { return nil }
func (r *stubWebhookRepo) GetByTenant(ctx context.Context, tenantID, instanceID string) ([]domain.Webhook, error) {
	return nil, nil
}
func (r *stubWebhookRepo) GetByID(ctx context.Context, id string) (*domain.Webhook, error) {
	return nil, domain.ErrWebhookNotFound
}
func (r *stubWebhookRepo) Delete(ctx context.Context, id string) error { return nil }
func (r *stubWebhookRepo) CreateDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error {
	if r.deliveries == nil {
		r.deliveries = map[string]*domain.WebhookDelivery{}
	}
	copyItem := *delivery
	r.deliveries[delivery.ID] = &copyItem
	return nil
}
func (r *stubWebhookRepo) UpdateDeliveryAttempt(ctx context.Context, id string, status domain.WebhookDeliveryStatus, responseStatus int, responseBody, lastError string, deliveredAt, attemptedAt *time.Time) error {
	item, ok := r.deliveries[id]
	if !ok {
		return domain.ErrWebhookDeliveryNotFound
	}
	item.Status = status
	item.Attempts++
	item.ResponseStatus = responseStatus
	item.ResponseBody = responseBody
	item.LastError = lastError
	item.DeliveredAt = deliveredAt
	item.LastAttemptAt = attemptedAt
	return nil
}
func (r *stubWebhookRepo) ListDeliveries(ctx context.Context, tenantID, instanceID, webhookID string, limit int) ([]domain.WebhookDelivery, error) {
	return nil, nil
}
func (r *stubWebhookRepo) GetDeliveryByID(ctx context.Context, id string) (*domain.WebhookDelivery, error) {
	item, ok := r.deliveries[id]
	if !ok {
		return nil, domain.ErrWebhookDeliveryNotFound
	}
	copyItem := *item
	return &copyItem, nil
}

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

	repo := &stubWebhookRepo{deliveries: map[string]*domain.WebhookDelivery{}}
	dispatcher := NewDispatcher(repo, events.NewBus(), queue.NewPool(1, 1, 1, logger.New()), time.Second, 1, logger.New())
	dispatcher.httpClient = srv.Client()

	envelope := domain.WebhookEnvelope{
		ID:        "delivery-1",
		Version:   "v1",
		Type:      domain.EventMessageReceived,
		TenantID:  "tenant-1",
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"message_id": "msg-1",
			"type":       "image",
		},
	}
	payloadJSON, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	wh := domain.Webhook{
		ID:     "wh-1",
		URL:    srv.URL,
		Secret: "super-secret",
		Active: true,
	}
	delivery := &domain.WebhookDelivery{
		ID:          envelope.ID,
		WebhookID:   wh.ID,
		TenantID:    envelope.TenantID,
		EventType:   envelope.Type,
		WebhookURL:  wh.URL,
		Status:      domain.WebhookDeliveryQueued,
		PayloadJSON: payloadJSON,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := repo.CreateDelivery(context.Background(), delivery); err != nil {
		t.Fatalf("create delivery: %v", err)
	}

	if err := dispatcher.Deliver(context.Background(), wh, delivery, 1); err != nil {
		t.Fatalf("deliver webhook: %v", err)
	}

	if gotEvent != string(envelope.Type) {
		t.Fatalf("unexpected event header: %s", gotEvent)
	}
	if gotWebhookID == "" {
		t.Fatal("expected webhook id header")
	}
	if gotSignature != sign(gotBody, wh.Secret) {
		t.Fatalf("unexpected signature: %s", gotSignature)
	}

	var receivedEnvelope domain.WebhookEnvelope
	if err := json.Unmarshal(gotBody, &receivedEnvelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if receivedEnvelope.ID != gotWebhookID {
		t.Fatalf("header/body id mismatch: %s vs %s", gotWebhookID, receivedEnvelope.ID)
	}
	if receivedEnvelope.Version != "v1" || receivedEnvelope.Type != domain.EventMessageReceived || receivedEnvelope.TenantID != "tenant-1" {
		t.Fatalf("unexpected envelope metadata: %+v", receivedEnvelope)
	}
	if receivedEnvelope.Timestamp.IsZero() {
		t.Fatal("expected timestamp in envelope")
	}
	stored, err := repo.GetDeliveryByID(context.Background(), delivery.ID)
	if err != nil {
		t.Fatalf("get stored delivery: %v", err)
	}
	if stored.Status != domain.WebhookDeliveryDelivered || stored.Attempts != 1 {
		t.Fatalf("unexpected stored delivery: %+v", stored)
	}
}
