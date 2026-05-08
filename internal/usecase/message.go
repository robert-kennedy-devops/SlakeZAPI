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

var (
	phoneRegex      = regexp.MustCompile(`^\d{8,15}$`)
	nonDigitPattern = regexp.MustCompile(`\D`)
)

const maxMediaDownloadSize = 32 << 20

type MessageUsecase struct {
	msgRepo      domain.MessageRepository
	instanceRepo domain.InstanceRepository
	campaignRepo domain.CampaignRepository
	whatsapp     domain.WhatsAppService
	billing      domain.BillingService
	eventBus     domain.EventBus
	log          *logger.Logger
}

func (u *MessageUsecase) ListMessages(ctx context.Context, tenantID, requestedInstanceID string, rawQuery url.Values) ([]domain.Message, error) {
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

	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.msgRepo.ListByTenant(ctx, tenantID, instanceID, limit, offset)
}

func (u *MessageUsecase) ListConversations(ctx context.Context, tenantID, requestedInstanceID string, rawQuery url.Values) ([]domain.Conversation, error) {
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

	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.msgRepo.ListConversations(ctx, tenantID, instanceID, limit, offset)
}

func (u *MessageUsecase) UpdateConversation(ctx context.Context, tenantID, requestedInstanceID, phone string, req domain.UpdateConversationRequest) (*domain.Conversation, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	convo, err := u.msgRepo.GetConversation(ctx, tenantID, instanceID, phone)
	if err != nil {
		return nil, err
	}
	if req.State != "" {
		convo.State = req.State
	}
	if req.AssignedUserID != "" {
		convo.AssignedUserID = req.AssignedUserID
	}
	convo.Note = req.Note
	convo.UpdatedAt = time.Now().UTC()
	if err := u.msgRepo.UpdateConversation(ctx, convo); err != nil {
		return nil, err
	}
	return u.msgRepo.GetConversation(ctx, tenantID, instanceID, phone)
}

func (u *MessageUsecase) GetMessage(ctx context.Context, tenantID, requestedInstanceID, messageID string) (*domain.Message, error) {
	msg, err := u.msgRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg.TenantID != tenantID {
		return nil, domain.ErrMessageNotFound
	}
	if requestedInstanceID != "" && msg.InstanceID != requestedInstanceID {
		return nil, domain.ErrMessageNotFound
	}
	return msg, nil
}

func (u *MessageUsecase) GetMessageMedia(ctx context.Context, tenantID, requestedInstanceID, messageID string) (*domain.MediaDownload, error) {
	msg, err := u.GetMessage(ctx, tenantID, requestedInstanceID, messageID)
	if err != nil {
		return nil, err
	}
	if msg.DirectPath == "" || len(msg.MediaKey) == 0 {
		return nil, domain.ErrMessageMediaAbsent
	}

	download, err := u.whatsapp.DownloadMedia(ctx, tenantID, msg.InstanceID, msg)
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

func (u *MessageUsecase) ResolveContacts(ctx context.Context, tenantID string, req domain.ResolveContactsRequest) ([]domain.ResolvedContact, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	normalized, originalByNormalized, err := normalizePhoneInputs(req.Phones)
	if err != nil {
		return nil, err
	}
	recognized, err := u.whatsapp.ResolveContacts(ctx, tenantID, instanceID, normalized)
	if err != nil {
		return nil, err
	}

	byLookup := make(map[string]domain.ResolvedContact, len(recognized))
	for _, contact := range recognized {
		byLookup[strings.TrimPrefix(contact.LookupPhone, "+")] = contact
	}

	results := make([]domain.ResolvedContact, 0, len(normalized))
	for _, phone := range normalized {
		contact, ok := byLookup[phone]
		if !ok {
			results = append(results, domain.ResolvedContact{
				InputPhone:  originalByNormalized[phone],
				LookupPhone: "+" + phone,
				Phone:       phone,
				IsWhatsApp:  false,
				Error:       "contact lookup returned no result",
			})
			continue
		}
		contact.InputPhone = originalByNormalized[phone]
		if contact.Phone == "" {
			contact.Phone = phone
		}
		results = append(results, contact)
	}

	return results, nil
}

func NewMessageUsecase(
	msgRepo domain.MessageRepository,
	instanceRepo domain.InstanceRepository,
	campaignRepo domain.CampaignRepository,
	whatsapp domain.WhatsAppService,
	billing domain.BillingService,
	eventBus domain.EventBus,
	log *logger.Logger,
) *MessageUsecase {
	return &MessageUsecase{
		msgRepo:      msgRepo,
		instanceRepo: instanceRepo,
		campaignRepo: campaignRepo,
		whatsapp:     whatsapp,
		billing:      billing,
		eventBus:     eventBus,
		log:          log,
	}
}

// SendMessage validates, checks billing, sends via WhatsApp, and persists the message.
func (u *MessageUsecase) SendMessage(ctx context.Context, tenantID string, req domain.SendMessageRequest) (*domain.SendMessageResponse, error) {
	if req.Message == "" {
		return nil, fmt.Errorf("%w: message body is required", domain.ErrBadRequest)
	}

	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}

	resolvedPhone, err := u.resolveSingleRecipient(ctx, tenantID, instanceID, req.Phone)
	if err != nil {
		return nil, err
	}

	return u.sendTextToResolvedPhone(ctx, tenantID, instanceID, resolvedPhone, req.Message)
}

func (u *MessageUsecase) sendTextToResolvedPhone(ctx context.Context, tenantID, instanceID, phone, message string) (*domain.SendMessageResponse, error) {
	if message == "" {
		return nil, fmt.Errorf("%w: message body is required", domain.ErrBadRequest)
	}

	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}

	waID, err := u.whatsapp.SendMessage(ctx, tenantID, instanceID, phone, message)
	if err != nil {
		return nil, fmt.Errorf("whatsapp send: %w", err)
	}

	now := time.Now().UTC()
	msg := &domain.Message{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		InstanceID: instanceID,
		WhatsAppID: waID,
		Phone:      phone,
		Body:       message,
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

	go func() {
		if err := u.billing.TrackSent(context.Background(), tenantID); err != nil {
			u.log.WithContext(ctx).Error("usage tracking failed", map[string]interface{}{"err": err.Error()})
		}
	}()

	u.eventBus.Publish(domain.Event{
		Type:       domain.EventMessageSent,
		TenantID:   tenantID,
		InstanceID: instanceID,
		Payload:    msg,
	})

	u.log.WithContext(ctx).Info("message sent", map[string]interface{}{
		"message_id": msg.ID, "phone": phone,
	})

	return &domain.SendMessageResponse{
		MessageID: msg.ID,
		Status:    msg.Status,
	}, nil
}

func (u *MessageUsecase) SendBulkMessage(ctx context.Context, tenantID string, req domain.BulkSendMessageRequest) (*domain.BulkSendMessageResponse, error) {
	if strings.TrimSpace(req.Message) == "" {
		return nil, fmt.Errorf("%w: message body is required", domain.ErrBadRequest)
	}
	if len(req.Phones) == 0 {
		return nil, fmt.Errorf("%w: at least one phone is required", domain.ErrBadRequest)
	}

	contacts, err := u.ResolveContacts(ctx, tenantID, domain.ResolveContactsRequest{Phones: req.Phones})
	if err != nil {
		return nil, err
	}
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}

	response := &domain.BulkSendMessageResponse{
		Total:   len(req.Phones),
		Results: make([]domain.BulkSendMessageItem, 0, len(contacts)),
	}
	for _, contact := range contacts {
		item := domain.BulkSendMessageItem{
			InputPhone: contact.InputPhone,
			Phone:      contact.Phone,
			IsWhatsApp: contact.IsWhatsApp,
		}
		if !contact.IsWhatsApp {
			item.Error = contact.Error
			response.Failed++
			response.Results = append(response.Results, item)
			continue
		}

		response.Accepted++
		singleResp, sendErr := u.sendTextToResolvedPhone(ctx, tenantID, instanceID, contact.Phone, req.Message)
		if sendErr != nil {
			item.Error = sendErr.Error()
			response.Failed++
		} else {
			item.MessageID = singleResp.MessageID
			item.Status = singleResp.Status
			response.Sent++
		}
		response.Results = append(response.Results, item)
	}

	return response, nil
}

func (u *MessageUsecase) SendMediaMessage(ctx context.Context, tenantID string, req domain.SendMediaMessageRequest) (*domain.SendMessageResponse, error) {
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

	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}

	resolvedPhone, err := u.resolveSingleRecipient(ctx, tenantID, instanceID, req.Phone)
	if err != nil {
		return nil, err
	}
	req.Phone = resolvedPhone

	enriched, err := enrichMediaRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	waID, err := u.whatsapp.SendMediaMessage(ctx, tenantID, instanceID, enriched)
	if err != nil {
		return nil, fmt.Errorf("whatsapp media send: %w", err)
	}

	now := time.Now().UTC()
	msg := &domain.Message{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		InstanceID: instanceID,
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
		Type:       domain.EventMessageSent,
		TenantID:   tenantID,
		InstanceID: instanceID,
		Payload:    msg,
	})

	return &domain.SendMessageResponse{
		MessageID: msg.ID,
		Status:    msg.Status,
	}, nil
}

func (u *MessageUsecase) SendInteractiveMessage(ctx context.Context, tenantID string, req domain.InteractiveMessageRequest) (*domain.SendMessageResponse, error) {
	if strings.TrimSpace(req.Body) == "" {
		return nil, fmt.Errorf("%w: interactive body is required", domain.ErrBadRequest)
	}
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	resolvedPhone, err := u.resolveSingleRecipient(ctx, tenantID, instanceID, req.Phone)
	if err != nil {
		return nil, err
	}
	req.Phone = resolvedPhone
	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}
	waID, err := u.whatsapp.SendInteractiveMessage(ctx, tenantID, instanceID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := &domain.Message{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		InstanceID: instanceID,
		WhatsAppID: waID,
		Phone:      req.Phone,
		Body:       req.Body,
		Type:       "interactive_" + strings.ToLower(req.Type),
		Direction:  "outbound",
		Status:     domain.MessageStatusSent,
		SentAt:     now,
		CreatedAt:  now,
	}
	if err := u.msgRepo.Create(ctx, msg); err != nil {
		u.log.WithContext(ctx).Error("failed to persist interactive message", map[string]interface{}{"err": err.Error()})
	}
	go func() { _ = u.billing.TrackSent(context.Background(), tenantID) }()
	u.eventBus.Publish(domain.Event{Type: domain.EventMessageSent, TenantID: tenantID, InstanceID: instanceID, Payload: msg})
	return &domain.SendMessageResponse{MessageID: msg.ID, Status: msg.Status}, nil
}

func (u *MessageUsecase) SendGroupMessage(ctx context.Context, tenantID string, req domain.GroupMessageRequest) (*domain.SendMessageResponse, error) {
	if strings.TrimSpace(req.Message) == "" || strings.TrimSpace(req.GroupJID) == "" {
		return nil, fmt.Errorf("%w: group_jid and message are required", domain.ErrBadRequest)
	}
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}
	waID, err := u.whatsapp.SendGroupMessage(ctx, tenantID, instanceID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := &domain.Message{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		InstanceID: instanceID,
		WhatsAppID: waID,
		Phone:      req.GroupJID,
		Body:       req.Message,
		Type:       "group_text",
		Direction:  "outbound",
		Status:     domain.MessageStatusSent,
		SentAt:     now,
		CreatedAt:  now,
	}
	if err := u.msgRepo.Create(ctx, msg); err != nil {
		u.log.WithContext(ctx).Error("failed to persist group message", map[string]interface{}{"err": err.Error()})
	}
	go func() { _ = u.billing.TrackSent(context.Background(), tenantID) }()
	return &domain.SendMessageResponse{MessageID: msg.ID, Status: msg.Status}, nil
}

func (u *MessageUsecase) PostStatus(ctx context.Context, tenantID string, req domain.StatusMessageRequest) (*domain.SendMessageResponse, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	if req.Type == "" {
		req.Type = "text"
	}
	if req.Type != "text" && req.URL != "" {
		mediaReq, enrichErr := enrichMediaRequest(ctx, domain.SendMediaMessageRequest{
			Type:       req.Type,
			URL:        req.URL,
			Caption:    req.Caption,
			FileName:   req.FileName,
			MimeType:   req.MimeType,
			InstanceID: req.InstanceID,
		})
		if enrichErr != nil {
			return nil, enrichErr
		}
		req.FileName = mediaReq.FileName
		req.MimeType = mediaReq.MimeType
		req.Data = mediaReq.Data
	}
	waID, err := u.whatsapp.PostStatus(ctx, tenantID, instanceID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := &domain.Message{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		InstanceID: instanceID,
		WhatsAppID: waID,
		Phone:      "status@broadcast",
		Body:       req.Message,
		Type:       "status_" + req.Type,
		Direction:  "outbound",
		Status:     domain.MessageStatusSent,
		SentAt:     now,
		CreatedAt:  now,
	}
	if err := u.msgRepo.Create(ctx, msg); err != nil {
		u.log.WithContext(ctx).Error("failed to persist status message", map[string]interface{}{"err": err.Error()})
	}
	return &domain.SendMessageResponse{MessageID: msg.ID, Status: msg.Status}, nil
}

func (u *MessageUsecase) SendLocationMessage(ctx context.Context, tenantID string, req domain.LocationMessageRequest) (*domain.SendMessageResponse, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	resolvedPhone, err := u.resolveSingleRecipient(ctx, tenantID, instanceID, req.Phone)
	if err != nil {
		return nil, err
	}
	req.Phone = resolvedPhone
	req.InstanceID = instanceID
	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}
	waID, err := u.whatsapp.SendLocationMessage(ctx, tenantID, instanceID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := &domain.Message{
		ID: uuid.NewString(), TenantID: tenantID, InstanceID: instanceID,
		WhatsAppID: waID, Phone: req.Phone, Body: req.Name,
		Type: "location", Direction: "outbound", Status: domain.MessageStatusSent,
		SentAt: now, CreatedAt: now,
	}
	_ = u.msgRepo.Create(ctx, msg)
	go func() { _ = u.billing.TrackSent(context.Background(), tenantID) }()
	u.eventBus.Publish(domain.Event{Type: domain.EventMessageSent, TenantID: tenantID, InstanceID: instanceID, Payload: msg})
	return &domain.SendMessageResponse{MessageID: msg.ID, Status: msg.Status}, nil
}

func (u *MessageUsecase) SendContactCard(ctx context.Context, tenantID string, req domain.ContactCardRequest) (*domain.SendMessageResponse, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	resolvedPhone, err := u.resolveSingleRecipient(ctx, tenantID, instanceID, req.Phone)
	if err != nil {
		return nil, err
	}
	req.Phone = resolvedPhone
	req.InstanceID = instanceID
	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}
	waID, err := u.whatsapp.SendContactCard(ctx, tenantID, instanceID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := &domain.Message{
		ID: uuid.NewString(), TenantID: tenantID, InstanceID: instanceID,
		WhatsAppID: waID, Phone: req.Phone, Body: strings.Join(req.Contacts, ","),
		Type: "contact", Direction: "outbound", Status: domain.MessageStatusSent,
		SentAt: now, CreatedAt: now,
	}
	_ = u.msgRepo.Create(ctx, msg)
	go func() { _ = u.billing.TrackSent(context.Background(), tenantID) }()
	u.eventBus.Publish(domain.Event{Type: domain.EventMessageSent, TenantID: tenantID, InstanceID: instanceID, Payload: msg})
	return &domain.SendMessageResponse{MessageID: msg.ID, Status: msg.Status}, nil
}

func (u *MessageUsecase) SendSticker(ctx context.Context, tenantID string, req domain.SendMediaMessageRequest) (*domain.SendMessageResponse, error) {
	if req.URL == "" {
		return nil, fmt.Errorf("%w: url is required", domain.ErrBadRequest)
	}
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	resolvedPhone, err := u.resolveSingleRecipient(ctx, tenantID, instanceID, req.Phone)
	if err != nil {
		return nil, err
	}
	req.Phone = resolvedPhone
	req.Type = "sticker"
	enriched, err := enrichMediaRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}
	waID, err := u.whatsapp.SendSticker(ctx, tenantID, instanceID, enriched)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := &domain.Message{
		ID: uuid.NewString(), TenantID: tenantID, InstanceID: instanceID,
		WhatsAppID: waID, Phone: enriched.Phone,
		Type: "sticker", Direction: "outbound", Status: domain.MessageStatusSent,
		SentAt: now, CreatedAt: now,
	}
	_ = u.msgRepo.Create(ctx, msg)
	go func() { _ = u.billing.TrackSent(context.Background(), tenantID) }()
	u.eventBus.Publish(domain.Event{Type: domain.EventMessageSent, TenantID: tenantID, InstanceID: instanceID, Payload: msg})
	return &domain.SendMessageResponse{MessageID: msg.ID, Status: msg.Status}, nil
}

func (u *MessageUsecase) ReactToMessage(ctx context.Context, tenantID string, req domain.ReactMessageRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.ReactToMessage(ctx, tenantID, instanceID, req)
}

func (u *MessageUsecase) DeleteMessage(ctx context.Context, tenantID string, req domain.DeleteMessageRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.DeleteMessage(ctx, tenantID, instanceID, req)
}

func (u *MessageUsecase) SendQuotedMessage(ctx context.Context, tenantID string, req domain.QuotedSendRequest) (*domain.SendMessageResponse, error) {
	if strings.TrimSpace(req.Message) == "" || strings.TrimSpace(req.QuotedMessageID) == "" {
		return nil, fmt.Errorf("%w: message and quoted_message_id are required", domain.ErrBadRequest)
	}
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	resolvedPhone, err := u.resolveSingleRecipient(ctx, tenantID, instanceID, req.Phone)
	if err != nil {
		return nil, err
	}
	req.Phone = resolvedPhone
	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}
	waID, err := u.whatsapp.SendQuotedMessage(ctx, tenantID, instanceID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := &domain.Message{
		ID: uuid.NewString(), TenantID: tenantID, InstanceID: instanceID,
		WhatsAppID: waID, Phone: req.Phone, Body: req.Message,
		Type: "text", Direction: "outbound", Status: domain.MessageStatusSent,
		SentAt: now, CreatedAt: now,
	}
	_ = u.msgRepo.Create(ctx, msg)
	go func() { _ = u.billing.TrackSent(context.Background(), tenantID) }()
	u.eventBus.Publish(domain.Event{Type: domain.EventMessageSent, TenantID: tenantID, InstanceID: instanceID, Payload: msg})
	return &domain.SendMessageResponse{MessageID: msg.ID, Status: msg.Status}, nil
}

func (u *MessageUsecase) EditMessage(ctx context.Context, tenantID string, req domain.EditMessageRequest) error {
	if strings.TrimSpace(req.MessageID) == "" || strings.TrimSpace(req.NewMessage) == "" {
		return fmt.Errorf("%w: message_id and new_message are required", domain.ErrBadRequest)
	}
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.EditMessage(ctx, tenantID, instanceID, req)
}

func (u *MessageUsecase) ForwardMessage(ctx context.Context, tenantID string, req domain.ForwardMessageRequest) (*domain.SendMessageResponse, error) {
	if strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.Message) == "" {
		return nil, fmt.Errorf("%w: phone and message are required", domain.ErrBadRequest)
	}
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	resolvedPhone, err := u.resolveSingleRecipient(ctx, tenantID, instanceID, req.Phone)
	if err != nil {
		return nil, err
	}
	req.Phone = resolvedPhone
	if err := u.billing.CheckLimit(ctx, tenantID); err != nil {
		return nil, err
	}
	waID, err := u.whatsapp.ForwardMessage(ctx, tenantID, instanceID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := &domain.Message{
		ID: uuid.NewString(), TenantID: tenantID, InstanceID: instanceID,
		WhatsAppID: waID, Phone: req.Phone, Body: req.Message,
		Type: "text", Direction: "outbound", Status: domain.MessageStatusSent,
		SentAt: now, CreatedAt: now,
	}
	_ = u.msgRepo.Create(ctx, msg)
	go func() { _ = u.billing.TrackSent(context.Background(), tenantID) }()
	u.eventBus.Publish(domain.Event{Type: domain.EventMessageSent, TenantID: tenantID, InstanceID: instanceID, Payload: msg})
	return &domain.SendMessageResponse{MessageID: msg.ID, Status: msg.Status}, nil
}

func (u *MessageUsecase) StarMessage(ctx context.Context, tenantID string, req domain.StarMessageRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.StarMessage(ctx, tenantID, instanceID, req)
}

func (u *MessageUsecase) ListGroups(ctx context.Context, tenantID, requestedInstanceID string) ([]domain.Group, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.whatsapp.ListGroups(ctx, tenantID, instanceID)
}

func (u *MessageUsecase) ListContacts(ctx context.Context, tenantID, requestedInstanceID string) ([]domain.WAContact, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.whatsapp.ListContacts(ctx, tenantID, instanceID)
}

func (u *MessageUsecase) resolveSingleRecipient(ctx context.Context, tenantID, instanceID, rawPhone string) (string, error) {
	normalized, err := normalizePhone(rawPhone)
	if err != nil {
		return "", err
	}

	contacts, err := u.whatsapp.ResolveContacts(ctx, tenantID, instanceID, []string{normalized})
	if err != nil {
		return "", err
	}
	if len(contacts) == 0 {
		return "", fmt.Errorf("%w: phone could not be resolved on WhatsApp", domain.ErrInvalidPhone)
	}
	resolved := contacts[0]
	if !resolved.IsWhatsApp {
		return "", fmt.Errorf("%w: phone is not registered on WhatsApp", domain.ErrInvalidPhone)
	}
	if resolved.Phone == "" {
		return normalized, nil
	}
	return resolved.Phone, nil
}

func (u *MessageUsecase) resolveInstanceID(ctx context.Context, tenantID, requestedInstanceID string) (string, error) {
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

func normalizePhoneInputs(inputs []string) ([]string, map[string]string, error) {
	if len(inputs) == 0 {
		return nil, nil, fmt.Errorf("%w: at least one phone is required", domain.ErrBadRequest)
	}
	normalized := make([]string, 0, len(inputs))
	originalByNormalized := make(map[string]string, len(inputs))
	for _, raw := range inputs {
		phone, err := normalizePhone(raw)
		if err != nil {
			return nil, nil, err
		}
		if _, ok := originalByNormalized[phone]; ok {
			continue
		}
		normalized = append(normalized, phone)
		originalByNormalized[phone] = strings.TrimSpace(raw)
	}
	return normalized, originalByNormalized, nil
}

func normalizePhone(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", domain.ErrInvalidPhone
	}
	digits := nonDigitPattern.ReplaceAllString(trimmed, "")
	if !phoneRegex.MatchString(digits) {
		return "", domain.ErrInvalidPhone
	}
	return digits, nil
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
