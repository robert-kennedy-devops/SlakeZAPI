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
