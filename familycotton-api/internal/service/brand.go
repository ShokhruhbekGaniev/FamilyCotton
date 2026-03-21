package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type BrandService struct {
	repo *repository.BrandRepository
}

func NewBrandService(repo *repository.BrandRepository) *BrandService {
	return &BrandService{repo: repo}
}

func (s *BrandService) Create(ctx context.Context, req *model.CreateBrandRequest) (*model.Brand, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	brand := &model.Brand{
		ID:   uuid.New(),
		Name: req.Name,
	}
	if err := s.repo.Create(ctx, brand); err != nil {
		return nil, err
	}
	return brand, nil
}

func (s *BrandService) GetByID(ctx context.Context, id uuid.UUID) (*model.Brand, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BrandService) List(ctx context.Context, page, limit int) ([]model.Brand, int, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *BrandService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateBrandRequest) (*model.Brand, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	brand, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		brand.Name = *req.Name
	}
	if err := s.repo.Update(ctx, brand); err != nil {
		return nil, err
	}
	return brand, nil
}

func (s *BrandService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
