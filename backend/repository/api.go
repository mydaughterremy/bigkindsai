package repository

import (
	"context"
	"time"

	"bigkinds.or.kr/backend/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApiRepository struct {
	DB       *gorm.DB
	timezone *time.Location
}

func NewApiRepository(db *gorm.DB, tz *time.Location) *ApiRepository {
	return &ApiRepository{
		DB:       db,
		timezone: tz,
	}
}

func (r *ApiRepository) CreateApikey(ctx context.Context, ak *model.Apikey) (*model.Apikey, error) {
	res := r.DB.WithContext(ctx).Clauses(clause.Returning{}).Create(ak)
	if res.Error != nil {
		return nil, res.Error
	}
	return ak, nil
}

func (r *ApiRepository) GetApikey(ctx context.Context, akId string) (*model.Apikey, error) {
	var ak model.Apikey
	if err := r.DB.WithContext(ctx).Where("id = ?", akId).First(&ak).Error; err != nil {
		return nil, err
	}

	return &ak, nil
}

func (r *ApiRepository) UpdateApikey(ctx context.Context, ak *model.Apikey) (*model.Apikey, error) {
	res := r.DB.WithContext(ctx).Model(&model.Apikey{
		ID: ak.ID,
	}).Clauses(clause.Returning{}).Updates(ak)

	if res.Error != nil {
		return nil, res.Error
	}
	return ak, nil
}

func (r *ApiRepository) DeleteApikey(ctx context.Context, akId string) error {
	res := r.DB.WithContext(ctx).Delete(&model.Apikey{
		ID: akId,
	})

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
