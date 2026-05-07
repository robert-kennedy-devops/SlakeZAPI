package usecase

import "github.com/whatsapp-saas/api/internal/domain"

type OperationsUsecase struct {
	queue domain.QueueService
}

func NewOperationsUsecase(queue domain.QueueService) *OperationsUsecase {
	return &OperationsUsecase{queue: queue}
}

func (u *OperationsUsecase) QueueSnapshot() domain.QueueSnapshot {
	return u.queue.Snapshot()
}
