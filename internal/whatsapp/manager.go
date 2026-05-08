package whatsapp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"google.golang.org/protobuf/proto"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	wmstore "go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	wmtypes "go.mau.fi/whatsmeow/types"
	wmevents "go.mau.fi/whatsmeow/types/events"
)

type session struct {
	tenantID   string
	instanceID string
	client     *whatsmeow.Client
	deviceJID  string
	phone      string
	status     domain.SessionStatus
	updatedAt  time.Time
	lastEvent  string
	lastError  string
	qrCode     string
	qrCancel   context.CancelFunc
}

type inboundMediaMeta struct {
	msgType       string
	mimeType      string
	fileName      string
	mediaURL      string
	directPath    string
	fileLength    int64
	mediaKey      []byte
	fileSHA256    []byte
	fileEncSHA256 []byte
}

type Manager struct {
	mu           sync.RWMutex
	sessions     map[string]*session
	eventBus     domain.EventBus
	instanceRepo domain.InstanceRepository
	sessionRepo  domain.SessionRepository
	msgRepo      domain.MessageRepository
	billing      domain.BillingService
	container    *sqlstore.Container
	log          *logger.Logger
}

func NewManager(
	db *sql.DB,
	eventBus domain.EventBus,
	instanceRepo domain.InstanceRepository,
	sessionRepo domain.SessionRepository,
	msgRepo domain.MessageRepository,
	billing domain.BillingService,
	log *logger.Logger,
) (*Manager, error) {
	sqlstore.PostgresArrayWrapper = pq.Array
	container := sqlstore.NewWithDB(db, "postgres", newWhatsmeowLogger(log, "whatsmeow/store"))
	if err := container.Upgrade(context.Background()); err != nil {
		return nil, fmt.Errorf("upgrade whatsmeow store: %w", err)
	}

	return &Manager{
		sessions:     make(map[string]*session),
		eventBus:     eventBus,
		instanceRepo: instanceRepo,
		sessionRepo:  sessionRepo,
		msgRepo:      msgRepo,
		billing:      billing,
		container:    container,
		log:          log,
	}, nil
}

var _ domain.WhatsAppService = (*Manager)(nil)

func (m *Manager) Connect(ctx context.Context, tenantID, instanceID string) (string, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return "", err
	}

	if sess.client.IsConnected() {
		return "", nil
	}

	if sess.client.Store.ID != nil {
		if err := sess.client.Connect(); err != nil {
			m.setSessionDiagnostic(ctx, sess, "reconnect_error", err.Error(), "")
			return "", fmt.Errorf("reconnect session: %w", err)
		}
		m.setSessionState(ctx, sess, domain.SessionStatusConnecting, sess.phone, "reconnect_requested", "")
		return "", nil
	}

	if sess.qrCancel != nil {
		sess.qrCancel()
		sess.qrCancel = nil
	}
	qrCtx, cancel := context.WithCancel(context.Background())
	sess.qrCancel = cancel

	qrChan, err := sess.client.GetQRChannel(qrCtx)
	if err != nil {
		sess.qrCancel = nil
		cancel()
		m.setSessionDiagnostic(ctx, sess, "qr_channel_error", err.Error(), "")
		return "", fmt.Errorf("get qr channel: %w", err)
	}
	if err := sess.client.Connect(); err != nil {
		sess.qrCancel = nil
		cancel()
		m.setSessionDiagnostic(ctx, sess, "connect_error", err.Error(), "")
		return "", fmt.Errorf("connect session: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case item, ok := <-qrChan:
			if !ok {
				return "", domain.ErrQRTimeout
			}
			switch item.Event {
			case whatsmeow.QRChannelEventCode:
				m.setSessionState(ctx, sess, domain.SessionStatusConnecting, sess.phone, "qr_code", item.Code)
				return item.Code, nil
			case whatsmeow.QRChannelTimeout.Event:
				m.setSessionDiagnostic(ctx, sess, "qr_timeout", domain.ErrQRTimeout.Error(), "")
				return "", domain.ErrQRTimeout
			case whatsmeow.QRChannelEventError:
				if item.Error != nil {
					m.setSessionDiagnostic(ctx, sess, "qr_error", item.Error.Error(), "")
				}
				return "", item.Error
			}
		}
	}
}

func (m *Manager) Disconnect(ctx context.Context, tenantID, instanceID string) error {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return err
	}

	sess.client.Disconnect()
	if sess.qrCancel != nil {
		sess.qrCancel()
		sess.qrCancel = nil
	}
	m.setSessionState(ctx, sess, domain.SessionStatusDisconnected, sess.phone, "manual_disconnect", "")
	return nil
}

func (m *Manager) Logout(ctx context.Context, tenantID, instanceID string) error {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return err
	}
	if sess.qrCancel != nil {
		sess.qrCancel()
		sess.qrCancel = nil
	}
	logoutErr := sess.client.Logout(ctx)
	if logoutErr != nil {
		m.log.WithContext(ctx).Warn("whatsapp logout failed, forcing local cleanup", map[string]interface{}{
			"tenant_id":    tenantID,
			"instance_id":  instanceID,
			"logout_error": logoutErr.Error(),
		})
		sess.client.Disconnect()
		if sess.client.Store != nil && sess.client.Store.ID != nil {
			if err := sess.client.Store.Delete(ctx); err != nil {
				m.log.WithContext(ctx).Warn("whatsapp local store delete failed", map[string]interface{}{
					"tenant_id":   tenantID,
					"instance_id": instanceID,
					"err":         err.Error(),
				})
			}
		}
	}
	return m.forgetSession(ctx, tenantID, instanceID, sess)
}

func (m *Manager) forgetSession(ctx context.Context, tenantID, instanceID string, sess *session) error {
	if err := m.sessionRepo.Delete(ctx, instanceID); err != nil && !errors.Is(err, domain.ErrSessionMetadataNotFound) {
		return err
	}

	m.mu.Lock()
	delete(m.sessions, instanceID)
	m.mu.Unlock()

	now := time.Now().UTC()
	_ = m.instanceRepo.UpdateStatus(ctx, instanceID, domain.SessionStatusDisconnected, "", now)
	m.eventBus.Publish(domain.Event{
		Type:       domain.EventConnectionUpdate,
		TenantID:   tenantID,
		InstanceID: instanceID,
		Payload:    map[string]string{"status": string(domain.SessionStatusDisconnected)},
	})
	return nil
}

func (m *Manager) SendMessage(ctx context.Context, tenantID, instanceID, phone, message string) (string, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return "", err
	}
	if !sess.client.IsConnected() {
		return "", domain.ErrSessionNotConnected
	}

	jid := wmtypes.NewJID(phone, wmtypes.DefaultUserServer)
	resp, err := sess.client.SendMessage(ctx, jid, &waProto.Message{
		Conversation: proto.String(message),
	})
	if err != nil {
		return "", fmt.Errorf("whatsapp send: %w", err)
	}

	return resp.ID, nil
}

func (m *Manager) SendMediaMessage(ctx context.Context, tenantID, instanceID string, req domain.SendMediaMessageRequest) (string, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return "", err
	}
	if !sess.client.IsConnected() {
		return "", domain.ErrSessionNotConnected
	}
	if len(req.Data) == 0 {
		return "", fmt.Errorf("%w: media payload is empty", domain.ErrBadRequest)
	}

	jid := wmtypes.NewJID(req.Phone, wmtypes.DefaultUserServer)
	appInfo, err := mediaAppInfo(req.Type)
	if err != nil {
		return "", err
	}
	upload, err := sess.client.Upload(ctx, req.Data, appInfo)
	if err != nil {
		return "", fmt.Errorf("upload media: %w", err)
	}

	msg := buildMediaMessage(req, upload)
	resp, err := sess.client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("send media: %w", err)
	}
	return resp.ID, nil
}

func (m *Manager) SendInteractiveMessage(ctx context.Context, tenantID, instanceID string, req domain.InteractiveMessageRequest) (string, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return "", err
	}
	if !sess.client.IsConnected() {
		return "", domain.ErrSessionNotConnected
	}
	jid := wmtypes.NewJID(req.Phone, wmtypes.DefaultUserServer)
	msg, err := buildInteractiveMessage(sess.client, req)
	if err != nil {
		return "", err
	}
	resp, err := sess.client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("send interactive: %w", err)
	}
	return resp.ID, nil
}

func (m *Manager) SendGroupMessage(ctx context.Context, tenantID, instanceID string, req domain.GroupMessageRequest) (string, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return "", err
	}
	if !sess.client.IsConnected() {
		return "", domain.ErrSessionNotConnected
	}
	jid, err := wmtypes.ParseJID(req.GroupJID)
	if err != nil {
		return "", fmt.Errorf("%w: invalid group jid", domain.ErrBadRequest)
	}
	resp, err := sess.client.SendMessage(ctx, jid, &waProto.Message{
		Conversation: proto.String(req.Message),
	})
	if err != nil {
		return "", fmt.Errorf("send group message: %w", err)
	}
	return resp.ID, nil
}

func (m *Manager) PostStatus(ctx context.Context, tenantID, instanceID string, req domain.StatusMessageRequest) (string, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return "", err
	}
	if !sess.client.IsConnected() {
		return "", domain.ErrSessionNotConnected
	}

	var msg *waProto.Message
	switch strings.ToLower(req.Type) {
	case "", "text":
		msg = &waProto.Message{Conversation: proto.String(req.Message)}
	case "image", "video", "audio", "document":
		uploadType, err := mediaAppInfo(req.Type)
		if err != nil {
			return "", err
		}
		upload, err := sess.client.Upload(ctx, req.Data, uploadType)
		if err != nil {
			return "", fmt.Errorf("upload status media: %w", err)
		}
		msg = buildMediaMessage(domain.SendMediaMessageRequest{
			Type:     req.Type,
			Caption:  req.Caption,
			FileName: req.FileName,
			MimeType: req.MimeType,
		}, upload)
	default:
		return "", fmt.Errorf("%w: unsupported status type", domain.ErrBadRequest)
	}
	resp, err := sess.client.SendMessage(ctx, wmtypes.StatusBroadcastJID, msg)
	if err != nil {
		return "", fmt.Errorf("send status: %w", err)
	}
	return resp.ID, nil
}

func (m *Manager) ListGroups(ctx context.Context, tenantID, instanceID string) ([]domain.Group, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}
	if !sess.client.IsConnected() {
		return nil, domain.ErrSessionNotConnected
	}
	groups, err := sess.client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	result := make([]domain.Group, 0, len(groups))
	for _, item := range groups {
		count := item.ParticipantCount
		if count == 0 {
			count = len(item.Participants)
		}
		result = append(result, domain.Group{
			JID:              item.JID.String(),
			Name:             item.Name,
			Topic:            item.Topic,
			ParticipantCount: count,
			IsAnnounce:       item.IsAnnounce,
			IsLocked:         item.IsLocked,
			CreatedAt:        item.GroupCreated,
		})
	}
	return result, nil
}

func (m *Manager) ListContacts(ctx context.Context, tenantID, instanceID string) ([]domain.WAContact, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}
	if !sess.client.IsConnected() {
		return nil, domain.ErrSessionNotConnected
	}
	all, err := sess.client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	result := make([]domain.WAContact, 0, len(all))
	for jid, info := range all {
		if jid.Server != "s.whatsapp.net" {
			continue
		}
		result = append(result, domain.WAContact{
			Phone:        jid.User,
			FullName:     info.FullName,
			PushName:     info.PushName,
			BusinessName: info.BusinessName,
		})
	}
	return result, nil
}

func (m *Manager) ResolveContacts(ctx context.Context, tenantID, instanceID string, phones []string) ([]domain.ResolvedContact, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}
	if !sess.client.IsConnected() {
		return nil, domain.ErrSessionNotConnected
	}
	if len(phones) == 0 {
		return []domain.ResolvedContact{}, nil
	}

	lookupPhones := make([]string, 0, len(phones))
	indexByLookup := make(map[string]int, len(phones))
	for _, phone := range phones {
		lookup := phone
		if !strings.HasPrefix(lookup, "+") {
			lookup = "+" + lookup
		}
		if _, ok := indexByLookup[lookup]; ok {
			continue
		}
		indexByLookup[lookup] = len(lookupPhones)
		lookupPhones = append(lookupPhones, lookup)
	}

	responses, err := sess.client.IsOnWhatsApp(ctx, lookupPhones)
	if err != nil {
		return nil, fmt.Errorf("resolve contacts: %w", err)
	}

	recognized := make([]domain.ResolvedContact, 0, len(responses))
	for _, item := range responses {
		contact := domain.ResolvedContact{
			LookupPhone: item.Query,
			Phone:       strings.TrimSpace(item.JID.User),
			JID:         item.JID.String(),
			IsWhatsApp:  item.IsIn,
		}
		if item.VerifiedName != nil && item.VerifiedName.Details != nil {
			contact.VerifiedName = strings.TrimSpace(item.VerifiedName.Details.GetVerifiedName())
		}
		if !contact.IsWhatsApp {
			contact.Error = "phone is not registered on WhatsApp"
		}
		recognized = append(recognized, contact)
	}

	return recognized, nil
}

func (m *Manager) DownloadMedia(ctx context.Context, tenantID, instanceID string, msg *domain.Message) (*domain.MediaDownload, error) {
	sess, err := m.ensureSession(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}
	if !sess.client.IsConnected() {
		return nil, domain.ErrSessionNotConnected
	}
	if msg.DirectPath == "" || len(msg.MediaKey) == 0 {
		return nil, domain.ErrMessageMediaAbsent
	}

	downloadable, fileName, mimeType, err := buildDownloadableMessage(msg)
	if err != nil {
		return nil, err
	}

	data, err := sess.client.Download(ctx, downloadable)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}

	return &domain.MediaDownload{
		FileName: fileName,
		MimeType: mimeType,
		Data:     data,
	}, nil
}

func (m *Manager) GetStatus(ctx context.Context, tenantID, instanceID string) domain.SessionStatus {
	sess, err := m.GetSession(ctx, tenantID, instanceID)
	if err != nil {
		return domain.SessionStatusDisconnected
	}
	return sess.Status
}

func (m *Manager) GetSession(ctx context.Context, tenantID, instanceID string) (*domain.Session, error) {
	m.mu.RLock()
	if sess, ok := m.sessions[instanceID]; ok {
		defer m.mu.RUnlock()
		return &domain.Session{
			TenantID:   sess.tenantID,
			InstanceID: sess.instanceID,
			DeviceJID:  sess.deviceJID,
			Phone:      sess.phone,
			Status:     sess.status,
			UpdatedAt:  sess.updatedAt,
			LastEvent:  sess.lastEvent,
			LastError:  sess.lastError,
			QRCode:     sess.qrCode,
		}, nil
	}
	m.mu.RUnlock()

	meta, err := m.sessionRepo.GetByInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

func (m *Manager) ensureSession(ctx context.Context, tenantID, instanceID string) (*session, error) {
	m.mu.RLock()
	sess, ok := m.sessions[instanceID]
	m.mu.RUnlock()
	if ok {
		return sess, nil
	}

	meta, err := m.sessionRepo.GetByInstance(ctx, instanceID)
	if err != nil && !errors.Is(err, domain.ErrSessionMetadataNotFound) {
		return nil, err
	}

	var deviceStoreID string
	var deviceStore *wmstore.Device
	if meta != nil && meta.DeviceJID != "" {
		deviceStoreID = meta.DeviceJID
		jid, err := wmtypes.ParseJID(meta.DeviceJID)
		if err != nil {
			return nil, fmt.Errorf("parse device jid: %w", err)
		}
		found, err := m.container.GetDevice(ctx, jid)
		if err != nil {
			return nil, fmt.Errorf("get device store: %w", err)
		}
		if found != nil {
			deviceStore = found
		}
	}
	if deviceStore == nil {
		deviceStore = m.container.NewDevice()
	}

	client := whatsmeow.NewClient(deviceStore, newWhatsmeowLogger(m.log, "whatsmeow/client"))
	sess = &session{
		tenantID:   tenantID,
		instanceID: instanceID,
		client:     client,
		deviceJID:  deviceStoreID,
		status:     domain.SessionStatusDisconnected,
		updatedAt:  time.Now().UTC(),
	}
	if meta != nil {
		sess.phone = meta.Phone
		sess.status = meta.Status
		sess.updatedAt = meta.UpdatedAt
		sess.lastEvent = meta.LastEvent
		sess.lastError = meta.LastError
		sess.qrCode = meta.QRCode
	}

	client.AddEventHandler(func(evt interface{}) {
		m.handleEvent(tenantID, instanceID, evt)
	})

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, exists := m.sessions[instanceID]; exists {
		return existing, nil
	}
	m.sessions[instanceID] = sess
	return sess, nil
}

func (m *Manager) handleEvent(tenantID, instanceID string, evt interface{}) {
	ctx := context.Background()

	switch event := evt.(type) {
	case *wmevents.Connected:
		sess, err := m.ensureSession(ctx, tenantID, instanceID)
		if err != nil {
			return
		}
		phone := sess.phone
		deviceJID := sess.deviceJID
		if sess.client.Store.ID != nil {
			phone = sess.client.Store.ID.User
			deviceJID = sess.client.Store.ID.String()
		}
		sess.deviceJID = deviceJID
		if sess.qrCancel != nil {
			sess.qrCancel()
			sess.qrCancel = nil
		}
		m.setSessionState(ctx, sess, domain.SessionStatusConnected, phone, "connected", "")
		m.eventBus.Publish(domain.Event{
			Type:       domain.EventConnectionUpdate,
			TenantID:   tenantID,
			InstanceID: instanceID,
			Payload:    map[string]string{"status": string(domain.SessionStatusConnected), "phone": phone},
		})

	case *wmevents.Disconnected:
		if sess, err := m.ensureSession(ctx, tenantID, instanceID); err == nil {
			if sess.qrCancel != nil {
				sess.qrCancel()
				sess.qrCancel = nil
			}
			m.setSessionState(ctx, sess, domain.SessionStatusDisconnected, sess.phone, "disconnected", "")
		}
		m.eventBus.Publish(domain.Event{
			Type:       domain.EventConnectionUpdate,
			TenantID:   tenantID,
			InstanceID: instanceID,
			Payload:    map[string]string{"status": string(domain.SessionStatusDisconnected)},
		})

	case *wmevents.LoggedOut:
		if sess, err := m.ensureSession(ctx, tenantID, instanceID); err == nil {
			if sess.qrCancel != nil {
				sess.qrCancel()
				sess.qrCancel = nil
			}
			m.setSessionState(ctx, sess, domain.SessionStatusDisconnected, "", "logged_out", "")
			m.setSessionDiagnostic(ctx, sess, "logged_out", fmt.Sprintf("logged out by WhatsApp (reason=%d)", event.Reason), "")
		}
		m.eventBus.Publish(domain.Event{
			Type:       domain.EventConnectionUpdate,
			TenantID:   tenantID,
			InstanceID: instanceID,
			Payload:    map[string]string{"status": string(domain.SessionStatusDisconnected)},
		})

	case *wmevents.ConnectFailure:
		if sess, err := m.ensureSession(ctx, tenantID, instanceID); err == nil {
			if sess.qrCancel != nil {
				sess.qrCancel()
				sess.qrCancel = nil
			}
			m.setSessionState(ctx, sess, domain.SessionStatusDisconnected, sess.phone, "connect_failure", "")
			m.setSessionDiagnostic(ctx, sess, "connect_failure", fmt.Sprintf("%s (reason=%d)", event.Message, event.Reason), "")
		}
		m.log.Warn("whatsapp connection failure", map[string]interface{}{
			"tenant_id": tenantID,
			"reason":    int(event.Reason),
			"message":   event.Message,
		})

	case *wmevents.StreamReplaced:
		if sess, err := m.ensureSession(ctx, tenantID, instanceID); err == nil {
			if sess.qrCancel != nil {
				sess.qrCancel()
				sess.qrCancel = nil
			}
			m.setSessionState(ctx, sess, domain.SessionStatusDisconnected, sess.phone, "stream_replaced", "")
			m.setSessionDiagnostic(ctx, sess, "stream_replaced", "device session was replaced by another login", "")
		}

	case *wmevents.KeepAliveTimeout:
		if sess, err := m.ensureSession(ctx, tenantID, instanceID); err == nil {
			m.setSessionDiagnostic(ctx, sess, "keepalive_timeout", fmt.Sprintf("keepalive timed out after %d consecutive failures", event.ErrorCount), "")
		}

	case *wmevents.Message:
		if event.Info.IsFromMe {
			return
		}
		body := extractMessageText(event.Message)
		if body == "" {
			body = "[unsupported]"
		}
		meta := extractMessageMeta(event.Message)
		msg := &domain.Message{
			ID:            event.Info.ID,
			TenantID:      tenantID,
			InstanceID:    instanceID,
			WhatsAppID:    event.Info.ID,
			Phone:         event.Info.Sender.User,
			Body:          body,
			Type:          meta.msgType,
			MimeType:      meta.mimeType,
			FileName:      meta.fileName,
			MediaURL:      meta.mediaURL,
			DirectPath:    meta.directPath,
			FileLength:    meta.fileLength,
			MediaKey:      meta.mediaKey,
			FileSHA256:    meta.fileSHA256,
			FileEncSHA256: meta.fileEncSHA256,
			Direction:     "inbound",
			Status:        domain.MessageStatusDelivered,
			SentAt:        event.Info.Timestamp.UTC(),
			CreatedAt:     time.Now().UTC(),
		}
		if err := m.msgRepo.Create(ctx, msg); err != nil && !isUniqueViolation(err) {
			m.log.Error("failed to persist inbound message", map[string]interface{}{"err": err.Error()})
			return
		}
		_ = m.billing.TrackReceived(ctx, tenantID)
		m.eventBus.Publish(domain.Event{
			Type:       domain.EventMessageReceived,
			TenantID:   tenantID,
			InstanceID: instanceID,
			Payload:    msg,
		})

	case *wmevents.Receipt:
		status, ok := mapReceiptStatus(event.Type)
		if !ok {
			return
		}
		for _, messageID := range event.MessageIDs {
			msg, err := m.msgRepo.GetByWhatsAppID(ctx, tenantID, instanceID, messageID)
			if err != nil {
				continue
			}
			if err := m.msgRepo.UpdateStatus(ctx, msg.ID, status); err != nil {
				continue
			}
			msg.Status = status
			m.eventBus.Publish(domain.Event{
				Type:       domain.EventMessageStatus,
				TenantID:   tenantID,
				InstanceID: instanceID,
				Payload:    msg,
			})
		}
	}
}

func (m *Manager) setSessionState(ctx context.Context, sess *session, status domain.SessionStatus, phone, event, qrCode string) {
	m.mu.Lock()
	sess.status = status
	sess.phone = phone
	if event != "" {
		sess.lastEvent = event
	}
	if qrCode != "" {
		sess.qrCode = qrCode
		sess.lastError = ""
	}
	if status == domain.SessionStatusConnected || status == domain.SessionStatusDisconnected {
		sess.qrCode = ""
	}
	if sess.client.Store.ID != nil {
		sess.deviceJID = sess.client.Store.ID.String()
	}
	sess.updatedAt = time.Now().UTC()
	m.mu.Unlock()

	record := &domain.Session{
		TenantID:   sess.tenantID,
		InstanceID: sess.instanceID,
		DeviceJID:  sess.deviceJID,
		Phone:      sess.phone,
		Status:     sess.status,
		UpdatedAt:  sess.updatedAt,
		LastEvent:  sess.lastEvent,
		LastError:  sess.lastError,
		QRCode:     sess.qrCode,
	}
	if err := m.sessionRepo.Upsert(ctx, record); err != nil {
		m.log.Error("failed to persist whatsapp session", map[string]interface{}{"err": err.Error(), "tenant_id": sess.tenantID})
	}
	_ = m.instanceRepo.UpdateStatus(ctx, sess.instanceID, sess.status, sess.phone, sess.updatedAt)
}

func (m *Manager) setSessionDiagnostic(ctx context.Context, sess *session, event, lastError, qrCode string) {
	m.mu.Lock()
	if event != "" {
		sess.lastEvent = event
	}
	if lastError != "" {
		sess.lastError = lastError
	}
	if qrCode != "" {
		sess.qrCode = qrCode
	}
	if lastError != "" && qrCode == "" {
		sess.qrCode = ""
	}
	sess.updatedAt = time.Now().UTC()
	record := &domain.Session{
		TenantID:   sess.tenantID,
		InstanceID: sess.instanceID,
		DeviceJID:  sess.deviceJID,
		Phone:      sess.phone,
		Status:     sess.status,
		UpdatedAt:  sess.updatedAt,
		LastEvent:  sess.lastEvent,
		LastError:  sess.lastError,
		QRCode:     sess.qrCode,
	}
	m.mu.Unlock()

	if err := m.sessionRepo.Upsert(ctx, record); err != nil {
		m.log.Error("failed to persist whatsapp session diagnostics", map[string]interface{}{"err": err.Error(), "tenant_id": sess.tenantID})
	}
	_ = m.instanceRepo.UpdateStatus(ctx, sess.instanceID, sess.status, sess.phone, sess.updatedAt)
}

func extractMessageText(msg *waProto.Message) string {
	switch {
	case msg == nil:
		return ""
	case msg.GetConversation() != "":
		return msg.GetConversation()
	case msg.GetExtendedTextMessage() != nil:
		return msg.GetExtendedTextMessage().GetText()
	case msg.GetImageMessage() != nil:
		return msg.GetImageMessage().GetCaption()
	case msg.GetVideoMessage() != nil:
		return msg.GetVideoMessage().GetCaption()
	case msg.GetDocumentMessage() != nil:
		return msg.GetDocumentMessage().GetCaption()
	default:
		return ""
	}
}

func extractMessageMeta(msg *waProto.Message) inboundMediaMeta {
	switch {
	case msg == nil:
		return inboundMediaMeta{msgType: "unknown"}
	case msg.GetConversation() != "" || msg.GetExtendedTextMessage() != nil:
		return inboundMediaMeta{msgType: "text"}
	case msg.GetImageMessage() != nil:
		image := msg.GetImageMessage()
		return inboundMediaMeta{
			msgType:       "image",
			mimeType:      image.GetMimetype(),
			mediaURL:      image.GetURL(),
			directPath:    image.GetDirectPath(),
			fileLength:    int64(image.GetFileLength()),
			mediaKey:      image.GetMediaKey(),
			fileSHA256:    image.GetFileSHA256(),
			fileEncSHA256: image.GetFileEncSHA256(),
		}
	case msg.GetVideoMessage() != nil:
		video := msg.GetVideoMessage()
		return inboundMediaMeta{
			msgType:       "video",
			mimeType:      video.GetMimetype(),
			mediaURL:      video.GetURL(),
			directPath:    video.GetDirectPath(),
			fileLength:    int64(video.GetFileLength()),
			mediaKey:      video.GetMediaKey(),
			fileSHA256:    video.GetFileSHA256(),
			fileEncSHA256: video.GetFileEncSHA256(),
		}
	case msg.GetAudioMessage() != nil:
		audio := msg.GetAudioMessage()
		return inboundMediaMeta{
			msgType:       "audio",
			mimeType:      audio.GetMimetype(),
			mediaURL:      audio.GetURL(),
			directPath:    audio.GetDirectPath(),
			fileLength:    int64(audio.GetFileLength()),
			mediaKey:      audio.GetMediaKey(),
			fileSHA256:    audio.GetFileSHA256(),
			fileEncSHA256: audio.GetFileEncSHA256(),
		}
	case msg.GetDocumentMessage() != nil:
		document := msg.GetDocumentMessage()
		return inboundMediaMeta{
			msgType:       "document",
			mimeType:      document.GetMimetype(),
			fileName:      document.GetFileName(),
			mediaURL:      document.GetURL(),
			directPath:    document.GetDirectPath(),
			fileLength:    int64(document.GetFileLength()),
			mediaKey:      document.GetMediaKey(),
			fileSHA256:    document.GetFileSHA256(),
			fileEncSHA256: document.GetFileEncSHA256(),
		}
	default:
		return inboundMediaMeta{msgType: "unknown"}
	}
}

func buildDownloadableMessage(msg *domain.Message) (whatsmeow.DownloadableMessage, string, string, error) {
	switch msg.Type {
	case "image":
		return &waProto.ImageMessage{
			Mimetype:      proto.String(msg.MimeType),
			DirectPath:    proto.String(msg.DirectPath),
			MediaKey:      msg.MediaKey,
			FileSHA256:    msg.FileSHA256,
			FileEncSHA256: msg.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(msg.FileLength)),
		}, msg.FileName, msg.MimeType, nil
	case "video":
		return &waProto.VideoMessage{
			Mimetype:      proto.String(msg.MimeType),
			DirectPath:    proto.String(msg.DirectPath),
			MediaKey:      msg.MediaKey,
			FileSHA256:    msg.FileSHA256,
			FileEncSHA256: msg.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(msg.FileLength)),
		}, msg.FileName, msg.MimeType, nil
	case "audio":
		return &waProto.AudioMessage{
			Mimetype:      proto.String(msg.MimeType),
			DirectPath:    proto.String(msg.DirectPath),
			MediaKey:      msg.MediaKey,
			FileSHA256:    msg.FileSHA256,
			FileEncSHA256: msg.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(msg.FileLength)),
		}, msg.FileName, msg.MimeType, nil
	case "document":
		return &waProto.DocumentMessage{
			Mimetype:      proto.String(msg.MimeType),
			FileName:      proto.String(msg.FileName),
			DirectPath:    proto.String(msg.DirectPath),
			MediaKey:      msg.MediaKey,
			FileSHA256:    msg.FileSHA256,
			FileEncSHA256: msg.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(msg.FileLength)),
		}, msg.FileName, msg.MimeType, nil
	default:
		return nil, "", "", domain.ErrMessageMediaAbsent
	}
}

func buildInteractiveMessage(client *whatsmeow.Client, req domain.InteractiveMessageRequest) (*waProto.Message, error) {
	switch strings.ToLower(strings.TrimSpace(req.Type)) {
	case "buttons":
		if len(req.Buttons) == 0 {
			return nil, fmt.Errorf("%w: at least one button is required", domain.ErrBadRequest)
		}
		if len(req.Buttons) > 3 {
			return nil, fmt.Errorf("%w: buttons message supports up to 3 buttons", domain.ErrBadRequest)
		}

		buttons := make([]*waProto.ButtonsMessage_Button, 0, len(req.Buttons))
		for idx, button := range req.Buttons {
			title := strings.TrimSpace(button.Title)
			if title == "" {
				return nil, fmt.Errorf("%w: button title is required", domain.ErrBadRequest)
			}
			buttonID := strings.TrimSpace(button.ID)
			if buttonID == "" {
				buttonID = fmt.Sprintf("btn_%d", idx+1)
			}
			buttons = append(buttons, &waProto.ButtonsMessage_Button{
				ButtonID: proto.String(buttonID),
				ButtonText: &waProto.ButtonsMessage_Button_ButtonText{
					DisplayText: proto.String(title),
				},
				Type: waProto.ButtonsMessage_Button_RESPONSE.Enum(),
			})
		}

		msg := &waProto.ButtonsMessage{
			ContentText: proto.String(strings.TrimSpace(req.Body)),
			FooterText:  proto.String(strings.TrimSpace(req.Footer)),
			Buttons:     buttons,
		}
		header := strings.TrimSpace(req.Header)
		if header != "" {
			msg.Header = &waProto.ButtonsMessage_Text{Text: header}
			msg.HeaderType = waProto.ButtonsMessage_TEXT.Enum()
		} else {
			msg.HeaderType = waProto.ButtonsMessage_EMPTY.Enum()
		}
		return &waProto.Message{ButtonsMessage: msg}, nil

	case "list":
		if strings.TrimSpace(req.ButtonText) == "" {
			return nil, fmt.Errorf("%w: button_text is required for list message", domain.ErrBadRequest)
		}
		if len(req.Sections) == 0 {
			return nil, fmt.Errorf("%w: at least one section is required", domain.ErrBadRequest)
		}

		sections := make([]*waProto.ListMessage_Section, 0, len(req.Sections))
		for sectionIdx, section := range req.Sections {
			if len(section.Rows) == 0 {
				return nil, fmt.Errorf("%w: section rows are required", domain.ErrBadRequest)
			}
			rows := make([]*waProto.ListMessage_Row, 0, len(section.Rows))
			for rowIdx, row := range section.Rows {
				title := strings.TrimSpace(row.Title)
				if title == "" {
					return nil, fmt.Errorf("%w: row title is required", domain.ErrBadRequest)
				}
				rowID := strings.TrimSpace(row.ID)
				if rowID == "" {
					rowID = fmt.Sprintf("row_%d_%d", sectionIdx+1, rowIdx+1)
				}
				rows = append(rows, &waProto.ListMessage_Row{
					Title:       proto.String(title),
					Description: proto.String(strings.TrimSpace(row.Description)),
					RowID:       proto.String(rowID),
				})
			}
			sections = append(sections, &waProto.ListMessage_Section{
				Title: proto.String(strings.TrimSpace(section.Title)),
				Rows:  rows,
			})
		}

		return &waProto.Message{ListMessage: &waProto.ListMessage{
			Title:       proto.String(strings.TrimSpace(req.Header)),
			Description: proto.String(strings.TrimSpace(req.Body)),
			ButtonText:  proto.String(strings.TrimSpace(req.ButtonText)),
			ListType:    waProto.ListMessage_SINGLE_SELECT.Enum(),
			Sections:    sections,
			FooterText:  proto.String(strings.TrimSpace(req.Footer)),
		}}, nil

	case "poll":
		if len(req.Options) < 2 {
			return nil, fmt.Errorf("%w: poll requires at least two options", domain.ErrBadRequest)
		}
		options := make([]string, 0, len(req.Options))
		for _, option := range req.Options {
			option = strings.TrimSpace(option)
			if option == "" {
				return nil, fmt.Errorf("%w: poll options cannot be empty", domain.ErrBadRequest)
			}
			if !slices.Contains(options, option) {
				options = append(options, option)
			}
		}
		if req.MaxSelect <= 0 {
			req.MaxSelect = 1
		}
		return client.BuildPollCreation(strings.TrimSpace(req.Body), options, req.MaxSelect), nil

	default:
		return nil, fmt.Errorf("%w: unsupported interactive type", domain.ErrBadRequest)
	}
}

func mediaAppInfo(mediaType string) (whatsmeow.MediaType, error) {
	switch mediaType {
	case "image":
		return whatsmeow.MediaImage, nil
	case "video":
		return whatsmeow.MediaVideo, nil
	case "audio":
		return whatsmeow.MediaAudio, nil
	case "document":
		return whatsmeow.MediaDocument, nil
	default:
		return "", fmt.Errorf("%w: unsupported media type", domain.ErrBadRequest)
	}
}

func buildMediaMessage(req domain.SendMediaMessageRequest, upload whatsmeow.UploadResponse) *waProto.Message {
	switch req.Type {
	case "image":
		return &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:       proto.String(req.Caption),
				Mimetype:      proto.String(req.MimeType),
				URL:           proto.String(upload.URL),
				DirectPath:    proto.String(upload.DirectPath),
				MediaKey:      upload.MediaKey,
				FileEncSHA256: upload.FileEncSHA256,
				FileSHA256:    upload.FileSHA256,
				FileLength:    proto.Uint64(upload.FileLength),
			},
		}
	case "video":
		return &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Caption:       proto.String(req.Caption),
				Mimetype:      proto.String(req.MimeType),
				URL:           proto.String(upload.URL),
				DirectPath:    proto.String(upload.DirectPath),
				MediaKey:      upload.MediaKey,
				FileEncSHA256: upload.FileEncSHA256,
				FileSHA256:    upload.FileSHA256,
				FileLength:    proto.Uint64(upload.FileLength),
			},
		}
	case "audio":
		return &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				Mimetype:      proto.String(req.MimeType),
				URL:           proto.String(upload.URL),
				DirectPath:    proto.String(upload.DirectPath),
				MediaKey:      upload.MediaKey,
				FileEncSHA256: upload.FileEncSHA256,
				FileSHA256:    upload.FileSHA256,
				FileLength:    proto.Uint64(upload.FileLength),
				PTT:           proto.Bool(false),
			},
		}
	default:
		return &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Title:         proto.String(req.FileName),
				FileName:      proto.String(req.FileName),
				Caption:       proto.String(req.Caption),
				Mimetype:      proto.String(req.MimeType),
				URL:           proto.String(upload.URL),
				DirectPath:    proto.String(upload.DirectPath),
				MediaKey:      upload.MediaKey,
				FileEncSHA256: upload.FileEncSHA256,
				FileSHA256:    upload.FileSHA256,
				FileLength:    proto.Uint64(upload.FileLength),
			},
		}
	}
}

func mapReceiptStatus(receiptType wmtypes.ReceiptType) (domain.MessageStatus, bool) {
	switch receiptType {
	case wmtypes.ReceiptTypeDelivered:
		return domain.MessageStatusDelivered, true
	case wmtypes.ReceiptTypeRead, wmtypes.ReceiptTypeReadSelf, wmtypes.ReceiptTypePlayed:
		return domain.MessageStatusRead, true
	default:
		return "", false
	}
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
