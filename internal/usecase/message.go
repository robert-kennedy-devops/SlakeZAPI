package usecase

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

var phoneRegex = regexp.MustCompile(`^\d{7,15}$`)

const maxMediaDownloadSize = 32 << 20

type MessageUsecase struct {
	msgRepo  domain.MessageRepository
	whatsapp domain.WhatsAppService
	billing  domain.BillingService
	eventBus domain.EventBus
	log      *logger.Logger
}

func (u *MessageUsecase) ListMessages(ctx context.Context, tenantID string, rawQuery url.Values) ([]domain.Message, error) {
	limit := 50
	offset := 0

	if v := rawQuery.Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 || parsed > 200 {
			return nil, fmt.Errorf("%w: invalid limit", domain.ErrBadRequest)
		}
		limit = parsed
	}
	if v := rawQuery.Get("offset"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 0 {
			return nil, fmt.Errorf("%w: invalid offset", domain.ErrBadRequest)
		}
		offset = parsed
	}

	return u.msgRepo.ListByTenant(ctx, tenantID, limit, offset)
}

func (u *MessageUsecase) GetMessage(ctx context.Context, tenantID, messageID string) (*domain.Message, error) {
	msg, err := u.msgRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg.TenantID != tenantID {
		return nil, domain.ErrMessageNotFound
	}
	return msg, nil
}

func (u *MessageUsecase) GetMessageMedia(ctx context.Context, tenantID, messageID string) (*domain.MediaDownload, error) {
	msg, err := u.GetMessage(ctx, tenantID, messageID)
	if err != nil {
		return nil, err
	}
	if msg.DirectPath == "" || len(msg.MediaKey) == 0 {
		return nil, domain.ErrMessageMediaAbsent
	}

	download, err := u.whatsapp.DownloadMedia(ctx, tenantID, msg)
	if err != nil {
		return nil, err
	}
	if download.FileName == "" {
		download.FileName = fallbackMediaFileName(msg)
	}
	if download.MimeType == "" {
		download.MimeType = msg.MimeType
	}
	return download, nil
}

func NewMessageUsecase(
	msgRepo domain.MessageRepository,
	whatsapp domain.WhatsAppService,
	billing domain.BillingService,
	eventBus domain.EventBus,
	log *logger.Logger,
) *MessageUsecase {
	return &MessageUsecase{
		msgRepo:  msgRepo,
		whatsapp: whatsapp,
		billing:  billing,
		eventBus: eventBus,
		log:      log,
	}
}

// SendMessage validates, checks billing, sends via WhatsApp, and persists the message.
func (u *MessageUsecase) SendMessage(ctx context.Context, tenantID string, req domain.SendMessageRequest) (*domain.SendMessageResponse, error) {
	// 1. Validate phone
	if !phoneRegex.MatchString(req.Phone) {
		return nil, domain.ErrInvalidPhone
	}
	if req.Message == "" {
		return nil, fmt.Errorf("%w: message body is required", domain.ErrBadRequest)
	}

	// 2. Check plan limit
	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}

	// 3. Send via WhatsApp
	waID, err := u.whatsapp.SendMessage(ctx, tenantID, req.Phone, req.Message)
	if err != nil {
		return nil, fmt.Errorf("whatsapp send: %w", err)
	}

	// 4. Persist message record
	now := time.Now().UTC()
	msg := &domain.Message{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		WhatsAppID: waID,
		Phone:      req.Phone,
		Body:       req.Message,
		Type:       "text",
		Direction:  "outbound",
		Status:     domain.MessageStatusSent,
		SentAt:     now,
		CreatedAt:  now,
	}

	if err := u.msgRepo.Create(ctx, msg); err != nil {
		u.log.WithContext(ctx).Error("failed to persist message", map[string]interface{}{"err": err.Error()})
		// non-fatal: message was sent, log and continue
	}

	// 5. Track usage
	go func() {
		if err := u.billing.TrackSent(context.Background(), tenantID); err != nil {
			u.log.WithContext(ctx).Error("usage tracking failed", map[string]interface{}{"err": err.Error()})
		}
	}()

	// 6. Publish event to internal bus
	u.eventBus.Publish(domain.Event{
		Type:     domain.EventMessageSent,
		TenantID: tenantID,
		Payload:  msg,
	})

	u.log.WithContext(ctx).Info("message sent", map[string]interface{}{
		"message_id": msg.ID, "phone": req.Phone,
	})

	return &domain.SendMessageResponse{
		MessageID: msg.ID,
		Status:    msg.Status,
	}, nil
}

func (u *MessageUsecase) SendMediaMessage(ctx context.Context, tenantID string, req domain.SendMediaMessageRequest) (*domain.SendMessageResponse, error) {
	if !phoneRegex.MatchString(req.Phone) {
		return nil, domain.ErrInvalidPhone
	}
	mediaType := strings.ToLower(strings.TrimSpace(req.Type))
	if mediaType != "image" && mediaType != "video" && mediaType != "audio" && mediaType != "document" {
		return nil, fmt.Errorf("%w: invalid media type", domain.ErrBadRequest)
	}
	if req.URL == "" {
		return nil, fmt.Errorf("%w: url is required", domain.ErrBadRequest)
	}
	if _, err := url.ParseRequestURI(req.URL); err != nil {
		return nil, fmt.Errorf("%w: invalid url", domain.ErrBadRequest)
	}

	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}

	enriched, err := enrichMediaRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	waID, err := u.whatsapp.SendMediaMessage(ctx, tenantID, enriched)
	if err != nil {
		return nil, fmt.Errorf("whatsapp media send: %w", err)
	}

	now := time.Now().UTC()
	msg := &domain.Message{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		WhatsAppID: waID,
		Phone:      enriched.Phone,
		Body:       enriched.Caption,
		Type:       enriched.Type,
		MimeType:   enriched.MimeType,
		FileName:   enriched.FileName,
		Direction:  "outbound",
		Status:     domain.MessageStatusSent,
		SentAt:     now,
		CreatedAt:  now,
	}

	if err := u.msgRepo.Create(ctx, msg); err != nil {
		u.log.WithContext(ctx).Error("failed to persist media message", map[string]interface{}{"err": err.Error()})
	}

	go func() {
		if err := u.billing.TrackSent(context.Background(), tenantID); err != nil {
			u.log.WithContext(ctx).Error("usage tracking failed", map[string]interface{}{"err": err.Error()})
		}
	}()

	u.eventBus.Publish(domain.Event{
		Type:     domain.EventMessageSent,
		TenantID: tenantID,
		Payload:  msg,
	})

	return &domain.SendMessageResponse{
		MessageID: msg.ID,
		Status:    msg.Status,
	}, nil
}

func enrichMediaRequest(ctx context.Context, req domain.SendMediaMessageRequest) (domain.SendMediaMessageRequest, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return req, fmt.Errorf("%w: invalid media request", domain.ErrBadRequest)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return req, fmt.Errorf("%w: failed to fetch media", domain.ErrBadRequest)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return req, fmt.Errorf("%w: media url returned status %d", domain.ErrBadRequest, resp.StatusCode)
	}

	if resp.ContentLength > maxMediaDownloadSize {
		return req, fmt.Errorf("%w: media file too large", domain.ErrBadRequest)
	}

	reader := io.LimitReader(resp.Body, maxMediaDownloadSize+1)
	body, err := io.ReadAll(reader)
	if err != nil {
		return req, fmt.Errorf("%w: failed to read media", domain.ErrBadRequest)
	}
	if len(body) > maxMediaDownloadSize {
		return req, fmt.Errorf("%w: media file too large", domain.ErrBadRequest)
	}

	req.Data = body
	if req.MimeType == "" {
		req.MimeType = resp.Header.Get("Content-Type")
	}
	if req.MimeType == "" {
		req.MimeType = http.DetectContentType(body)
	}
	if req.FileName == "" {
		req.FileName = inferFileName(req.URL, req.Type, req.MimeType)
	}

	return req, nil
}

func inferFileName(rawURL, mediaType, mimeType string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		name := path.Base(parsed.Path)
		if name != "" && name != "." && name != "/" {
			return name
		}
	}

	ext := inferExtension(mimeType)
	if mediaType == "audio" {
		return "audio" + ext
	}
	return mediaType + ext
}

func fallbackMediaFileName(msg *domain.Message) string {
	if msg.FileName != "" {
		return msg.FileName
	}
	name := msg.Type
	if name == "" {
		name = "media"
	}
	return name + inferExtension(msg.MimeType)
}

func inferExtension(mimeType string) string {
	ext := ""
	if parts := strings.Split(mimeType, "/"); len(parts) == 2 {
		ext = "." + strings.TrimSpace(parts[1])
		if ext == ".jpeg" {
			ext = ".jpg"
		}
	}
	if ext == "." || ext == "" {
		return ".bin"
	}
	return ext
}
