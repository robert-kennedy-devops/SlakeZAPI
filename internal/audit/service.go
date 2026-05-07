package audit

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type Service struct {
	repo domain.AuditLogRepository
	log  *logger.Logger
}

func NewService(repo domain.AuditLogRepository, log *logger.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) Record(action string, fields map[string]interface{}) {
	if s.repo == nil {
		return
	}
	tenantID, _ := fields["tenant_id"].(string)
	if tenantID == "" {
		return
	}
	item := &domain.AuditLog{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		InstanceID: stringField(fields, "instance_id"),
		UserID:     stringField(fields, "user_id"),
		RequestID:  stringField(fields, "request_id"),
		Action:     action,
		Resource:   resourceFromAction(action),
		Payload:    cloneMap(fields),
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.repo.Create(context.Background(), item); err != nil {
		s.log.Error("audit persist failed", map[string]interface{}{
			"action": action,
			"err":    err.Error(),
		})
	}
}

func resourceFromAction(action string) string {
	if action == "" {
		return "unknown"
	}
	parts := strings.Split(action, ".")
	if len(parts) == 0 {
		return action
	}
	if parts[0] == "app" && len(parts) > 1 {
		return parts[1]
	}
	return parts[0]
}

func stringField(fields map[string]interface{}, key string) string {
	value, _ := fields[key].(string)
	return value
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
