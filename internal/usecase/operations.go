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

func (u *OperationsUsecase) DeadLetters(limit int) []domain.QueueJobView {
	return u.queue.DeadLetters(limit)
}

func (u *OperationsUsecase) RequeueDeadLetter(id string) error {
	return u.queue.RequeueDeadLetter(id)
}
