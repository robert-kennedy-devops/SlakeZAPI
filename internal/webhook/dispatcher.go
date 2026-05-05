package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	hooks, err := d.webhookRepo.GetByTenant(ctx, evt.TenantID)
	if err != nil || len(hooks) == 0 {
		return
	}

	for _, wh := range hooks {
		wh := wh // capture for closure
		if !wh.Active || !containsEvent(wh.Events, string(evt.Type)) {
			continue
		}

		d.pool.Enqueue(queue.Job{
			ID:      fmt.Sprintf("%s-%s", wh.ID, evt.Type),
			Payload: deliveryPayload{webhook: wh, event: evt},
			Handler: d.deliverHandler,
		})
	}
}

type deliveryPayload struct {
	webhook domain.Webhook
	event   domain.Event
}

func (d *Dispatcher) deliverHandler(ctx context.Context, raw interface{}) error {
	p := raw.(deliveryPayload)
	return d.Deliver(ctx, p.webhook, p.event)
}

// Deliver performs a single HTTP POST attempt to the webhook URL.
func (d *Dispatcher) Deliver(ctx context.Context, wh domain.Webhook, evt domain.Event) error {
	envelope := domain.WebhookEnvelope{
		ID:        uuid.NewString(),
		Version:   "v1",
		Type:      evt.Type,
		TenantID:  evt.TenantID,
		Timestamp: time.Now().UTC(),
		Payload:   evt.Payload,
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", string(evt.Type))
	req.Header.Set("X-Webhook-Id", envelope.ID)
	req.Header.Set("X-Webhook-Signature", sign(body, wh.Secret))

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		d.log.Info("webhook delivered", map[string]interface{}{
			"webhook_id": wh.ID,
			"event":      evt.Type,
			"status":     resp.StatusCode,
		})
		return nil
	}

	return fmt.Errorf("webhook returned status %d", resp.StatusCode)
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
