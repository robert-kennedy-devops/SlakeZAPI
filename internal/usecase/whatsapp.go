package usecase

import (
	"context"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type WhatsAppUsecase struct {
	whatsapp domain.WhatsAppService
	eventBus domain.EventBus
	log      *logger.Logger
}

func NewWhatsAppUsecase(
	whatsapp domain.WhatsAppService,
	eventBus domain.EventBus,
	log *logger.Logger,
) *WhatsAppUsecase {
	return &WhatsAppUsecase{
		whatsapp: whatsapp,
		eventBus: eventBus,
		log:      log,
	}
}

// Connect initiates a WhatsApp session and returns a QR code for scanning.
func (u *WhatsAppUsecase) Connect(ctx context.Context, tenantID string) (*domain.ConnectResponse, error) {
	session, err := u.whatsapp.GetSession(ctx, tenantID)
	if err == nil && session.Status == domain.SessionStatusConnected {
		return &domain.ConnectResponse{
			Status: string(domain.SessionStatusConnected),
			Phone:  session.Phone,
		}, nil
	}

	status := u.whatsapp.GetStatus(ctx, tenantID)
	if status == domain.SessionStatusConnected {
		return &domain.ConnectResponse{Status: string(domain.SessionStatusConnected)}, nil
	}

	qr, err := u.whatsapp.Connect(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	u.eventBus.Publish(domain.Event{
		Type:     domain.EventConnectionUpdate,
		TenantID: tenantID,
		Payload:  map[string]string{"status": string(domain.SessionStatusConnecting)},
	})

	u.log.WithContext(ctx).Info("qr code generated for tenant")

	return &domain.ConnectResponse{
		QRCode: qr,
		Status: string(domain.SessionStatusConnecting),
	}, nil
}

func (u *WhatsAppUsecase) Disconnect(ctx context.Context, tenantID string) error {
	return u.whatsapp.Disconnect(ctx, tenantID)
}

func (u *WhatsAppUsecase) Logout(ctx context.Context, tenantID string) error {
	return u.whatsapp.Logout(ctx, tenantID)
}

func (u *WhatsAppUsecase) GetSession(ctx context.Context, tenantID string) (*domain.Session, error) {
	return u.whatsapp.GetSession(ctx, tenantID)
}

// GetStatus returns the current connection status for a tenant.
func (u *WhatsAppUsecase) GetStatus(ctx context.Context, tenantID string) domain.SessionStatus {
	return u.whatsapp.GetStatus(ctx, tenantID)
}
