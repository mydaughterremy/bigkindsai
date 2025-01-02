package repository

import (
	"context"
	"time"

	"bigkinds.or.kr/backend/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QARepository struct {
	DB       *gorm.DB // nolint:unused
	timezone *time.Location
}

func NewQARepository(db *gorm.DB, timezone *time.Location) *QARepository {
	return &QARepository{DB: db, timezone: timezone}
}

func (s *QARepository) Begin() *QARepository {
	return &QARepository{DB: s.DB.Begin()}
}

func (s *QARepository) Commit() error {
	return s.DB.Commit().Error
}

func (s *QARepository) correctTimezone(qa *model.QA) *model.QA {
	qa.CreatedAt = qa.CreatedAt.In(s.timezone).Add(-9 * time.Hour)
	qa.UpdatedAt = qa.UpdatedAt.In(s.timezone).Add(-9 * time.Hour)
	return qa
}

func (s *QARepository) GetQA(ctx context.Context, id string) (*model.QA, error) {
	var qa model.QA
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&qa).Error; err != nil {
		return nil, err
	}

	return s.correctTimezone(&qa), nil
}

func (s *QARepository) ListQAs(ctx context.Context, from *time.Time, to *time.Time) ([]*model.QA, error) {
	var qas []*model.QA
	if err := s.DB.WithContext(ctx).Where("created_at BETWEEN ? AND ?", from, to).Order("created_at desc").Find(&qas).Error; err != nil {
		return nil, err
	}

	for _, qa := range qas {
		s.correctTimezone(qa)
	}

	return qas, nil
}

func (s *QARepository) ListChatIdQAs(ctx context.Context, chatID string) ([]*model.QA, error) {
	var qas []*model.QA
	if err := s.DB.WithContext(ctx).Where("chat_id = ?", chatID).Order("created_at desc").Find(&qas).Error; err != nil {
		return nil, err
	}

	return qas, nil
}

func (s *QARepository) ListChatQAs(ctx context.Context, session, chatID string) ([]*model.QA, error) {
	var qas []*model.QA
	if err := s.DB.WithContext(ctx).Where("session_id = ? AND chat_id = ?", session, chatID).Order("created_at").Find(&qas).Error; err != nil {
		return nil, err
	}

	for _, qa := range qas {
		s.correctTimezone(qa)
	}

	return qas, nil
}

func (s *QARepository) LastChatQA(ctx context.Context, chatID string) (*model.QA, error) {
	var qa *model.QA
	if err := s.DB.WithContext(ctx).Where("chat_id = ? AND answer != '' AND question != ''", chatID).Order("created_at desc").Limit(1).Find(&qa).Error; err != nil {
		return nil, err
	}

	return qa, nil

}

func (s *QARepository) ListQAsWithPagination(ctx context.Context, from time.Time, to time.Time, searchQuery string, offset int, limit int) ([]*model.QA, error) {
	var qas []*model.QA
	tx := s.DB.WithContext(ctx)

	if !from.IsZero() {
		tx = tx.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		tx = tx.Where("created_at <= ?", to)
	}
	if searchQuery != "" {
		tx = tx.Select("*, MATCH (question) AGAINST (? IN NATURAL LANGUAGE MODE) AS `score`", searchQuery).Having("`score` > 0").Order("score desc")
	}
	tx = tx.Order("created_at desc").Offset(offset).Limit(limit)

	if err := tx.Find(&qas).Error; err != nil {
		return nil, err
	}

	for _, qa := range qas {
		s.correctTimezone(qa)
	}

	return qas, nil
}

func (s *QARepository) CountQAs(ctx context.Context, from time.Time, to time.Time, searchQuery string) (int64, error) {
	var count int64
	if searchQuery == "" {
		if err := s.DB.WithContext(ctx).Model(&model.QA{}).Where("created_at BETWEEN ? AND ?", from, to).Count(&count).Error; err != nil {
			return 0, err
		}
	} else {
		if err := s.DB.WithContext(ctx).Model(&model.QA{}).Where("created_at BETWEEN ? AND ?", from, to).Where("MATCH (question) AGAINST (? IN NATURAL LANGUAGE MODE)", searchQuery).Count(&count).Error; err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (s *QARepository) DeleteQA(ctx context.Context, qaID uuid.UUID) error {
	result := s.DB.WithContext(ctx).Delete(&model.QA{}, qaID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *QARepository) DeleteQAs(ctx context.Context, qaIDs []string) error {
	result := s.DB.WithContext(ctx).Delete(&model.QA{}, qaIDs)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *QARepository) CreateQA(ctx context.Context, qa *model.QA) (*model.QA, error) {
	result := s.DB.WithContext(ctx).Clauses(clause.Returning{}).Create(qa)
	if result.Error != nil {
		return nil, result.Error
	}

	return qa, nil
}

func (s *QARepository) UpdateQA(ctx context.Context, qa *model.QA) (*model.QA, error) {
	result := s.DB.WithContext(ctx).Model(&model.QA{
		ID: qa.ID,
	}).Clauses(clause.Returning{}).Updates(qa)

	if result.Error != nil {
		return nil, result.Error
	}
	return qa, nil
}

func (s *QARepository) UpsertQA(ctx context.Context, qa *model.QA) (*model.QA, error) {
	result := s.DB.WithContext(ctx).Clauses(
		clause.Returning{},
		clause.OnConflict{
			UpdateAll: true,
		}).Create(qa)

	if result.Error != nil {
		return nil, result.Error
	}
	return qa, nil
}

func (s *QARepository) GetVote(ctx context.Context, qaID string) (string, error) {
	var qa model.QA
	if err := s.DB.WithContext(ctx).Where("id = ?", qaID).Select("vote").First(&qa).Error; err != nil {
		return "", err
	}
	return qa.Vote, nil
}

func (s *QARepository) UpdateVote(ctx context.Context, qaID string, vote string) error {
	err := s.DB.WithContext(ctx).Model(&model.QA{}).Where("id = ?", qaID).Update("vote", vote).Error
	return err
}
