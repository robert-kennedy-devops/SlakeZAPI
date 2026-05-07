package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type CampaignUsecase struct {
	campaignRepo domain.CampaignRepository
	instanceRepo domain.InstanceRepository
	msgUC        *MessageUsecase
	log          *logger.Logger
}

func NewCampaignUsecase(
	campaignRepo domain.CampaignRepository,
	instanceRepo domain.InstanceRepository,
	msgUC *MessageUsecase,
	log *logger.Logger,
) *CampaignUsecase {
	return &CampaignUsecase{
		campaignRepo: campaignRepo,
		instanceRepo: instanceRepo,
		msgUC:        msgUC,
		log:          log,
	}
}

func (u *CampaignUsecase) Create(ctx context.Context, tenantID string, req domain.CreateCampaignRequest) (*domain.Campaign, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("%w: campaign name is required", domain.ErrBadRequest)
	}
	if strings.TrimSpace(req.Message) == "" {
		return nil, fmt.Errorf("%w: campaign message is required", domain.ErrBadRequest)
	}
	if len(req.Recipients) == 0 {
		return nil, fmt.Errorf("%w: at least one recipient is required", domain.ErrBadRequest)
	}
	instanceID, err := u.resolveInstanceID(ctx, tenantID, req.InstanceID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	status := domain.CampaignStatusDraft
	if req.ScheduledAt != nil {
		status = domain.CampaignStatusScheduled
	}
	campaign := &domain.Campaign{
		ID:            uuid.NewString(),
		TenantID:      tenantID,
		InstanceID:    instanceID,
		Name:          strings.TrimSpace(req.Name),
		Message:       req.Message,
		Status:        status,
		ScheduledAt:   req.ScheduledAt,
		TotalContacts: len(req.Recipients),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	recipients := make([]domain.CampaignRecipient, 0, len(req.Recipients))
	for _, recipient := range req.Recipients {
		if strings.TrimSpace(recipient.Phone) == "" {
			return nil, fmt.Errorf("%w: recipient phone is required", domain.ErrBadRequest)
		}
		payload, _ := json.Marshal(recipient.Variables)
		recipients = append(recipients, domain.CampaignRecipient{
			ID:         uuid.NewString(),
			CampaignID: campaign.ID,
			InputPhone: recipient.Phone,
			Name:       recipient.Name,
			Variables:  string(payload),
			Status:     domain.MessageStatusPending,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	if err := u.campaignRepo.Create(ctx, campaign, recipients); err != nil {
		return nil, err
	}
	if campaign.Status == domain.CampaignStatusDraft {
		if err := u.Run(ctx, tenantID, campaign.ID); err != nil {
			return nil, err
		}
		return u.campaignRepo.GetByID(ctx, campaign.ID)
	}
	return campaign, nil
}

func (u *CampaignUsecase) List(ctx context.Context, tenantID, requestedInstanceID string) ([]domain.Campaign, error) {
	instanceID, err := u.resolveInstanceID(ctx, tenantID, requestedInstanceID)
	if err != nil {
		return nil, err
	}
	return u.campaignRepo.ListByTenant(ctx, tenantID, instanceID)
}

func (u *CampaignUsecase) RunDue(ctx context.Context) error {
	campaigns, err := u.campaignRepo.ListDue(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
	for _, campaign := range campaigns {
		if err := u.Run(ctx, campaign.TenantID, campaign.ID); err != nil {
			u.log.WithContext(ctx).Error("scheduled campaign failed", map[string]interface{}{"campaign_id": campaign.ID, "err": err.Error()})
		}
	}
	return nil
}

func (u *CampaignUsecase) Run(ctx context.Context, tenantID, campaignID string) error {
	campaign, err := u.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return err
	}
	if campaign.TenantID != tenantID {
		return domain.ErrCampaignNotFound
	}
	recipients, err := u.campaignRepo.ListRecipients(ctx, campaign.ID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := u.campaignRepo.UpdateStatus(ctx, campaign.ID, domain.CampaignStatusRunning, &now); err != nil {
		return err
	}

	sentCount := 0
	failedCount := 0
	for _, recipient := range recipients {
		contacts, err := u.msgUC.ResolveContacts(ctx, tenantID, domain.ResolveContactsRequest{
			InstanceID: campaign.InstanceID,
			Phones:     []string{recipient.InputPhone},
		})
		if err != nil || len(contacts) == 0 || !contacts[0].IsWhatsApp {
			failedCount++
			reason := "contact could not be resolved"
			if err != nil {
				reason = err.Error()
			} else if len(contacts) > 0 && contacts[0].Error != "" {
				reason = contacts[0].Error
			}
			_ = u.campaignRepo.UpdateRecipientResult(ctx, recipient.ID, "", "", domain.MessageStatusFailed, false, reason)
			continue
		}

		content := applyVariables(campaign.Message, recipient.Name, recipient.Variables)
		resp, sendErr := u.msgUC.sendTextToResolvedPhone(ctx, tenantID, campaign.InstanceID, contacts[0].Phone, content)
		if sendErr != nil {
			failedCount++
			_ = u.campaignRepo.UpdateRecipientResult(ctx, recipient.ID, contacts[0].Phone, "", domain.MessageStatusFailed, true, sendErr.Error())
			continue
		}
		sentCount++
		_ = u.campaignRepo.UpdateRecipientResult(ctx, recipient.ID, contacts[0].Phone, resp.MessageID, domain.MessageStatusSent, true, "")
	}

	if err := u.campaignRepo.UpdateCounters(ctx, campaign.ID, sentCount, failedCount); err != nil {
		return err
	}
	finishedStatus := domain.CampaignStatusCompleted
	if sentCount == 0 && failedCount > 0 {
		finishedStatus = domain.CampaignStatusFailed
	}
	if err := u.campaignRepo.UpdateStatus(ctx, campaign.ID, finishedStatus, &now); err != nil {
		return err
	}
	return nil
}

func (u *CampaignUsecase) resolveInstanceID(ctx context.Context, tenantID, requestedInstanceID string) (string, error) {
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

func applyVariables(template, fallbackName, rawVariables string) string {
	out := template
	if fallbackName != "" {
		out = strings.ReplaceAll(out, "{{name}}", fallbackName)
	}
	var variables map[string]string
	if rawVariables != "" && rawVariables != "{}" {
		_ = json.Unmarshal([]byte(rawVariables), &variables)
	}
	for key, value := range variables {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}
