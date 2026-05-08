package usecase

import (
	"context"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type WhatsAppUsecase struct {
	whatsapp     domain.WhatsAppService
	instanceRepo domain.InstanceRepository
	eventBus     domain.EventBus
	log          *logger.Logger
}

func NewWhatsAppUsecase(
	whatsapp domain.WhatsAppService,
	instanceRepo domain.InstanceRepository,
	eventBus domain.EventBus,
	log *logger.Logger,
) *WhatsAppUsecase {
	return &WhatsAppUsecase{
		whatsapp:     whatsapp,
		instanceRepo: instanceRepo,
		eventBus:     eventBus,
		log:          log,
	}
}

// Connect initiates a WhatsApp session and returns a QR code for scanning.
func (u *WhatsAppUsecase) Connect(ctx context.Context, tenantID, requestedInstanceID string) (*domain.ConnectResponse, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	session, err := u.whatsapp.GetSession(ctx, tenantID, instanceID)
	if err == nil && session.Status == domain.SessionStatusConnected {
		return &domain.ConnectResponse{
			InstanceID: instanceID,
			Status:     string(domain.SessionStatusConnected),
			Phone:      session.Phone,
		}, nil
	}

	status := u.whatsapp.GetStatus(ctx, tenantID, instanceID)
	if status == domain.SessionStatusConnected {
		return &domain.ConnectResponse{InstanceID: instanceID, Status: string(domain.SessionStatusConnected)}, nil
	}

	qr, err := u.whatsapp.Connect(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}

	u.eventBus.Publish(domain.Event{
		Type:       domain.EventConnectionUpdate,
		TenantID:   tenantID,
		InstanceID: instanceID,
		Payload:    map[string]string{"status": string(domain.SessionStatusConnecting)},
	})

	u.log.WithContext(ctx).Info("qr code generated for tenant")

	return &domain.ConnectResponse{
		InstanceID: instanceID,
		QRCode:     qr,
		Status:     string(domain.SessionStatusConnecting),
	}, nil
}

func (u *WhatsAppUsecase) Disconnect(ctx context.Context, tenantID, requestedInstanceID string) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.Disconnect(ctx, tenantID, instanceID)
}

func (u *WhatsAppUsecase) Logout(ctx context.Context, tenantID, requestedInstanceID string) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.Logout(ctx, tenantID, instanceID)
}

func (u *WhatsAppUsecase) GetSession(ctx context.Context, tenantID, requestedInstanceID string) (*domain.Session, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.whatsapp.GetSession(ctx, tenantID, instanceID)
}

// GetStatus returns the current connection status for a tenant.
func (u *WhatsAppUsecase) GetStatus(ctx context.Context, tenantID, requestedInstanceID string) domain.SessionStatus {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return domain.SessionStatusDisconnected
	}
	return u.whatsapp.GetStatus(ctx, tenantID, instanceID)
}

func (u *WhatsAppUsecase) GetGroupInfo(ctx context.Context, tenantID, requestedInstanceID, groupJID string) (*domain.Group, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.whatsapp.GetGroupInfo(ctx, tenantID, instanceID, groupJID)
}

func (u *WhatsAppUsecase) CreateGroup(ctx context.Context, tenantID string, req domain.CreateGroupRequest) (*domain.Group, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}
	req.InstanceID = instanceID
	return u.whatsapp.CreateGroup(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) UpdateGroupParticipants(ctx context.Context, tenantID string, req domain.UpdateGroupParticipantsRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	req.InstanceID = instanceID
	return u.whatsapp.UpdateGroupParticipants(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) UpdateGroupInfo(ctx context.Context, tenantID string, req domain.UpdateGroupInfoRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	req.InstanceID = instanceID
	return u.whatsapp.UpdateGroupInfo(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) GetGroupInviteLink(ctx context.Context, tenantID, requestedInstanceID, groupJID string) (*domain.GroupInviteLink, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.whatsapp.GetGroupInviteLink(ctx, tenantID, instanceID, groupJID)
}

func (u *WhatsAppUsecase) LeaveGroup(ctx context.Context, tenantID, requestedInstanceID, groupJID string) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.LeaveGroup(ctx, tenantID, instanceID, groupJID)
}

func (u *WhatsAppUsecase) BlockContact(ctx context.Context, tenantID, requestedInstanceID, phone string) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.BlockContact(ctx, tenantID, instanceID, phone)
}

func (u *WhatsAppUsecase) UnblockContact(ctx context.Context, tenantID, requestedInstanceID, phone string) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.UnblockContact(ctx, tenantID, instanceID, phone)
}

func (u *WhatsAppUsecase) GetContactAvatar(ctx context.Context, tenantID, requestedInstanceID, phone string) (*domain.ContactAvatar, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.whatsapp.GetContactAvatar(ctx, tenantID, instanceID, phone)
}

func (u *WhatsAppUsecase) GetProfile(ctx context.Context, tenantID, requestedInstanceID string) (*domain.InstanceProfile, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.whatsapp.GetProfile(ctx, tenantID, instanceID)
}

func (u *WhatsAppUsecase) UpdateProfile(ctx context.Context, tenantID string, req domain.UpdateProfileRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.UpdateProfile(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) GetPrivacySettings(ctx context.Context, tenantID, requestedInstanceID string) (*domain.PrivacySettings, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.whatsapp.GetPrivacySettings(ctx, tenantID, instanceID)
}

func (u *WhatsAppUsecase) UpdatePrivacySettings(ctx context.Context, tenantID string, req domain.UpdatePrivacyRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.UpdatePrivacySettings(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) ArchiveChat(ctx context.Context, tenantID string, req domain.ArchiveChatRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.ArchiveChat(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) MuteChat(ctx context.Context, tenantID string, req domain.MuteChatRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.MuteChat(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) PinChat(ctx context.Context, tenantID string, req domain.PinChatRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.PinChat(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) MarkChatRead(ctx context.Context, tenantID string, req domain.MarkChatReadRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.MarkChatRead(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) EditMessage(ctx context.Context, tenantID string, req domain.EditMessageRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.EditMessage(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) ForwardMessage(ctx context.Context, tenantID string, req domain.ForwardMessageRequest) (string, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return "", err
	}
	return u.whatsapp.ForwardMessage(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) StarMessage(ctx context.Context, tenantID string, req domain.StarMessageRequest) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.StarMessage(ctx, tenantID, instanceID, req)
}

func (u *WhatsAppUsecase) PairPhone(ctx context.Context, tenantID, requestedInstanceID, phone string) (string, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return "", err
	}
	return u.whatsapp.PairPhone(ctx, tenantID, instanceID, phone)
}

func (u *WhatsAppUsecase) RestartInstance(ctx context.Context, tenantID, requestedInstanceID string) error {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return err
	}
	return u.whatsapp.RestartInstance(ctx, tenantID, instanceID)
}

func (u *WhatsAppUsecase) resolveInstanceID(ctx context.Context, tenantID, requestedInstanceID string) (string, error) {
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
