package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type WebhookUsecase struct {
	webhookRepo  domain.WebhookRepository
	instanceRepo domain.InstanceRepository
	subRepo      domain.SubscriptionRepository
	deliverySvc  domain.WebhookReplayService
	log          *logger.Logger
}

func NewWebhookUsecase(
	webhookRepo domain.WebhookRepository,
	instanceRepo domain.InstanceRepository,
	subRepo domain.SubscriptionRepository,
	deliverySvc domain.WebhookReplayService,
	log *logger.Logger,
) *WebhookUsecase {
	return &WebhookUsecase{
		webhookRepo:  webhookRepo,
		instanceRepo: instanceRepo,
		subRepo:      subRepo,
		deliverySvc:  deliverySvc,
		log:          log,
	}
}

// Register creates a new webhook for the tenant.
func (u *WebhookUsecase) Register(ctx context.Context, tenantID string, req domain.RegisterWebhookRequest) (*domain.Webhook, error) {
	// Check if tenant plan allows webhooks
	sub, err := u.subRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !sub.Plan.WebhookEnabled {
		return nil, domain.ErrBadRequest
	}

	if err := validateWebhookURL(req.URL); err != nil {
		return nil, err
	}

	secret, _ := generateSecret()

	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}

	wh := &domain.Webhook{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		InstanceID: instanceID,
		URL:        req.URL,
		Events:     req.Events,
		Secret:     secret,
		Active:     true,
		CreatedAt:  time.Now().UTC(),
	}

	if err := u.webhookRepo.Create(ctx, wh); err != nil {
		return nil, err
	}

	u.log.WithContext(ctx).Info("webhook registered", map[string]interface{}{
		"webhook_id": wh.ID, "url": wh.URL,
	})

	return wh, nil
}

// ListByTenant returns active webhooks for a tenant.
func (u *WebhookUsecase) ListByTenant(ctx context.Context, tenantID, requestedInstanceID string) ([]domain.Webhook, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.webhookRepo.GetByTenant(ctx, tenantID, instanceID)
}

func (u *WebhookUsecase) Delete(ctx context.Context, tenantID, requestedInstanceID, webhookID string) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return err
	}
	hooks, err := u.webhookRepo.GetByTenant(ctx, tenantID, instanceID)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		if hook.ID == webhookID {
			return u.webhookRepo.Delete(ctx, webhookID)
		}
	}
	return domain.ErrWebhookNotFound
}

func (u *WebhookUsecase) ListDeliveries(ctx context.Context, tenantID, requestedInstanceID, webhookID string, limit int) ([]domain.WebhookDelivery, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.webhookRepo.ListDeliveries(ctx, tenantID, instanceID, webhookID, limit)
}

func (u *WebhookUsecase) ReplayDelivery(ctx context.Context, tenantID, requestedInstanceID, deliveryID string) (*domain.ReplayWebhookDeliveryResponse, error) {
	if u.deliverySvc == nil {
		return nil, domain.ErrBadRequest
	}
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	delivery, err := u.webhookRepo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return nil, err
	}
	if delivery.TenantID != tenantID || delivery.InstanceID != instanceID {
		return nil, domain.ErrWebhookDeliveryNotFound
	}
	wh, err := u.webhookRepo.GetByID(ctx, delivery.WebhookID)
	if err != nil {
		return nil, err
	}
	if wh.TenantID != tenantID {
		return nil, domain.ErrWebhookNotFound
	}
	replay, err := u.deliverySvc.ReplayDelivery(ctx, *wh, delivery)
	if err != nil {
		return nil, err
	}
	return &domain.ReplayWebhookDeliveryResponse{
		DeliveryID: replay.ID,
		Status:     replay.Status,
	}, nil
}

func (u *WebhookUsecase) resolveInstanceID(ctx context.Context, tenantID, requestedInstanceID string) (string, error) {
	if requestedInstanceID != "" {
		instance, err := u.instanceRepo.GetByID(ctx, requestedInstanceID)
		if err != nil {
			return "", err
		}
		if instance.TenantID != tenantID {
			return "", domain.ErrInstanceNotFound
		}
		return instance.ID, nil
	}
	instance, err := u.instanceRepo.GetDefaultByTenant(ctx, tenantID)
	if err != nil {
		return "", err
	}
	return instance.ID, nil
}

func generateSecret() (string, error) {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

// validateWebhookURL rejects URLs that would allow SSRF attacks.
// Only public HTTPS (or HTTP in dev) targets are permitted; loopback,
// private ranges, link-local, and non-HTTP schemes are all blocked.
func validateWebhookURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("%w: webhook url is required", domain.ErrBadRequest)
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("%w: invalid webhook url", domain.ErrBadRequest)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("%w: webhook url must use http or https", domain.ErrBadRequest)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("%w: webhook url missing host", domain.ErrBadRequest)
	}

	// Resolve the hostname to catch DNS rebinding to private IPs.
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("%w: webhook url host could not be resolved", domain.ErrBadRequest)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		if isPrivateIP(ip) {
			return fmt.Errorf("%w: webhook url must not target private or reserved addresses", domain.ErrBadRequest)
		}
	}
	return nil
}

// privateRanges lists all IP ranges that must not be reachable via webhook.
var privateRanges = func() []*net.IPNet {
	blocks := []string{
		"127.0.0.0/8",    // loopback
		"::1/128",        // IPv6 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // link-local (AWS metadata, etc.)
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique-local
		"100.64.0.0/10",  // shared address space (RFC6598)
		"0.0.0.0/8",      // "this" network
		"192.0.2.0/24",   // TEST-NET-1
		"198.51.100.0/24", // TEST-NET-2
		"203.0.113.0/24", // TEST-NET-3
		"240.0.0.0/4",    // reserved
	}
	nets := make([]*net.IPNet, 0, len(blocks))
	for _, b := range blocks {
		_, ipNet, _ := net.ParseCIDR(b)
		nets = append(nets, ipNet)
	}
	return nets
}()

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, block := range privateRanges {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
