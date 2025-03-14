package service

import (
	"context"

	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/repository"
)

type ApiService struct {
	ApiRepository *repository.ApiRepository
}

func (s *ApiService) CreateApikey(ctx context.Context, ak model.Apikey) (*model.Apikey, error) {
	newAk, err := s.ApiRepository.CreateApikey(ctx, &ak)
	if err != nil {
		return nil, err
	}
	return newAk, nil
}

func (s *ApiService) GetApikey(ctx context.Context, k string) (*model.Apikey, error) {
	ak, err := s.ApiRepository.GetApikey(ctx, k)
	if err != nil {
		return nil, err
	}
	return ak, nil

}

func (s *ApiService) UpdateApikey(ctx context.Context, ak model.Apikey) (*model.Apikey, error) {
	uAk, err := s.ApiRepository.UpdateApikey(ctx, &ak)
	if err != nil {
		return nil, err
	}
	return uAk, nil
}

func (s *ApiService) DeleteApikey(ctx context.Context, k string) error {
	err := s.ApiRepository.DeleteApikey(ctx, k)
	if err != nil {
		return err
	}
	return nil
}
