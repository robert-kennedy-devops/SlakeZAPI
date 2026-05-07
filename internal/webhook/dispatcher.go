package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/queue"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// Dispatcher listens on the event bus and delivers events to registered webhooks.
type Dispatcher struct {
	webhookRepo domain.WebhookRepository
	eventBus    domain.EventBus
	pool        *queue.Pool
	httpClient  *http.Client
	retries     int
	log         *logger.Logger
}

func NewDispatcher(
	webhookRepo domain.WebhookRepository,
	eventBus domain.EventBus,
	pool *queue.Pool,
	timeout time.Duration,
	retries int,
	log *logger.Logger,
) *Dispatcher {
	return &Dispatcher{
		webhookRepo: webhookRepo,
		eventBus:    eventBus,
		pool:        pool,
		httpClient:  &http.Client{Timeout: timeout},
		retries:     retries,
		log:         log,
	}
}

var _ domain.WebhookReplayService = (*Dispatcher)(nil)

// Start subscribes to all events on the bus and dispatches them to registered webhooks.
func (d *Dispatcher) Start(ctx context.Context) {
	ch, unsub := d.eventBus.SubscribeAll()
	defer unsub()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			d.dispatch(ctx, evt)
		}
	}
}

func (d *Dispatcher) dispatch(ctx context.Context, evt domain.Event) {
	hooks, err := d.webhookRepo.GetByTenant(ctx, evt.TenantID, evt.InstanceID)
	if err != nil || len(hooks) == 0 {
		return
	}

	for _, wh := range hooks {
		wh := wh // capture for closure
		if !wh.Active || !containsEvent(wh.Events, string(evt.Type)) {
			continue
		}

		envelope := domain.WebhookEnvelope{
			ID:         uuid.NewString(),
			Version:    "v1",
			Type:       evt.Type,
			TenantID:   evt.TenantID,
			InstanceID: evt.InstanceID,
			Timestamp:  time.Now().UTC(),
			Payload:    evt.Payload,
		}
		payloadJSON, err := json.Marshal(envelope)
		if err != nil {
			d.log.Error("failed to marshal webhook envelope", map[string]interface{}{"webhook_id": wh.ID, "err": err.Error()})
			continue
		}
		delivery := &domain.WebhookDelivery{
			ID:          envelope.ID,
			WebhookID:   wh.ID,
			TenantID:    evt.TenantID,
			InstanceID:  evt.InstanceID,
			EventType:   evt.Type,
			WebhookURL:  wh.URL,
			Status:      domain.WebhookDeliveryQueued,
			PayloadJSON: payloadJSON,
			CreatedAt:   envelope.Timestamp,
			UpdatedAt:   envelope.Timestamp,
		}
		if err := d.webhookRepo.CreateDelivery(ctx, delivery); err != nil {
			d.log.Error("failed to persist webhook delivery", map[string]interface{}{"webhook_id": wh.ID, "err": err.Error()})
			continue
		}

		d.pool.Enqueue(queue.Job{
			ID:      envelope.ID,
			Kind:    "webhook",
			Payload: deliveryPayload{webhook: wh, delivery: delivery},
			Handler: d.deliverHandler,
		})
	}
}

type deliveryPayload struct {
	webhook  domain.Webhook
	delivery *domain.WebhookDelivery
}

func (d *Dispatcher) deliverHandler(ctx context.Context, raw interface{}, attempt int) error {
	p := raw.(deliveryPayload)
	return d.Deliver(ctx, p.webhook, p.delivery, attempt)
}

// Deliver performs a single HTTP POST attempt to the webhook URL.
func (d *Dispatcher) Deliver(ctx context.Context, wh domain.Webhook, delivery *domain.WebhookDelivery, attempt int) error {
	req, _, err := d.buildRequest(ctx, wh, delivery)
	if err != nil {
		return err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		now := time.Now().UTC()
		status := domain.WebhookDeliveryFailed
		if attempt < d.retries {
			status = domain.WebhookDeliveryRetrying
		}
		_ = d.webhookRepo.UpdateDeliveryAttempt(ctx, delivery.ID, status, 0, "", err.Error(), nil, &now)
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	now := time.Now().UTC()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = d.webhookRepo.UpdateDeliveryAttempt(ctx, delivery.ID, domain.WebhookDeliveryDelivered, resp.StatusCode, string(respBody), "", &now, &now)
		d.log.Info("webhook delivered", map[string]interface{}{
			"webhook_id": wh.ID,
			"event":      delivery.EventType,
			"status":     resp.StatusCode,
		})
		return nil
	}

	status := domain.WebhookDeliveryFailed
	if attempt < d.retries {
		status = domain.WebhookDeliveryRetrying
	}
	_ = d.webhookRepo.UpdateDeliveryAttempt(ctx, delivery.ID, status, resp.StatusCode, string(respBody), fmt.Sprintf("webhook returned status %d", resp.StatusCode), nil, &now)
	return fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

func (d *Dispatcher) ReplayDelivery(ctx context.Context, wh domain.Webhook, source *domain.WebhookDelivery) (*domain.WebhookDelivery, error) {
	if source == nil {
		return nil, domain.ErrWebhookDeliveryNotFound
	}
	now := time.Now().UTC()
	replay := &domain.WebhookDelivery{
		ID:          uuid.NewString(),
		WebhookID:   wh.ID,
		TenantID:    source.TenantID,
		InstanceID:  source.InstanceID,
		EventType:   source.EventType,
		WebhookURL:  wh.URL,
		Status:      domain.WebhookDeliveryReplayed,
		PayloadJSON: append([]byte(nil), source.PayloadJSON...),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := d.webhookRepo.CreateDelivery(ctx, replay); err != nil {
		return nil, err
	}
	d.pool.Enqueue(queue.Job{
		ID:      replay.ID,
		Kind:    "webhook_replay",
		Payload: deliveryPayload{webhook: wh, delivery: replay},
		Handler: d.deliverHandler,
	})
	return replay, nil
}

func (d *Dispatcher) buildRequest(ctx context.Context, wh domain.Webhook, delivery *domain.WebhookDelivery) (*http.Request, []byte, error) {
	if delivery == nil {
		return nil, nil, domain.ErrWebhookDeliveryNotFound
	}
	body := delivery.PayloadJSON
	if len(body) == 0 {
		return nil, nil, fmt.Errorf("%w: empty webhook payload", domain.ErrBadRequest)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", string(delivery.EventType))
	req.Header.Set("X-Webhook-Id", delivery.ID)
	req.Header.Set("X-Webhook-Signature", sign(body, wh.Secret))
	return req, body, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func containsEvent(list []string, target string) bool {
	for _, e := range list {
		if e == target {
			return true
		}
	}
	return false
}
