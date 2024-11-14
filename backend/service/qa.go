package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/repository"
	"gorm.io/gorm"
)

type QAService struct {
	Repository *repository.QARepository
}

func NewQAService(db *gorm.DB, timezone *time.Location) *QAService {
	return &QAService{
		Repository: repository.NewQARepository(db, timezone),
	}
}

func (s *QAService) GetQA(ctx context.Context, qaID string) (*model.QA, error) {
	return s.Repository.GetQA(ctx, qaID)
}

func (s *QAService) DeleteQA(ctx context.Context, qaID string) error {
	qaIDUUID, err := uuid.Parse(qaID)
	if err != nil {
		return err
	}
	return s.Repository.DeleteQA(ctx, qaIDUUID)
}

func (s *QAService) DeleteQAs(ctx context.Context, ids []string) error {
	return s.Repository.DeleteQAs(ctx, ids)
}

func (s *QAService) ListQAsWithPagination(ctx context.Context, from time.Time, to time.Time, searchQuery string, offset int, limit int) (*model.QAsWithPagination, error) {
	totalCount, err := s.Repository.CountQAs(ctx, from, to, searchQuery)
	if err != nil {
		return nil, err
	}

	qas, err := s.Repository.ListQAsWithPagination(ctx, from, to, searchQuery, offset, limit)
	if err != nil {
		return nil, err
	}

	metadata := &model.PaginationMetadata{
		TotalCount:    int(totalCount),
		ReturnedCount: len(qas),
		CurrentOffset: offset,
	}
	return &model.QAsWithPagination{
		QAs:      qas,
		Metadata: metadata,
	}, nil
}

func (s *QAService) CreateQA(ctx context.Context, qa *model.QA) (*model.QA, error) {
	return s.Repository.CreateQA(ctx, qa)
}

func (s *QAService) UpdateQA(ctx context.Context, qa *model.QA) (*model.QA, error) {
	return s.Repository.UpdateQA(ctx, qa)
}

func (s *QAService) UpsertQA(ctx context.Context, qa *model.QA) (*model.QA, error) {
	return s.Repository.UpsertQA(ctx, qa)
}

func (s *QAService) GetVote(ctx context.Context, qaID string) (string, error) {
	return s.Repository.GetVote(ctx, qaID)
}

func (s *QAService) DeleteVote(ctx context.Context, qaID string) error {
	return s.Repository.UpdateVote(ctx, qaID, "")
}

func (s *QAService) UpvoteQA(ctx context.Context, qaID string) error {
	return s.Repository.UpdateVote(ctx, qaID, "up")
}

func (s *QAService) DownvoteQA(ctx context.Context, qaID string) error {
	return s.Repository.UpdateVote(ctx, qaID, "down")
}
