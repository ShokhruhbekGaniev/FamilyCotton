package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type SupplierService struct {
	repo *repository.SupplierRepository
}

func NewSupplierService(repo *repository.SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}

func (s *SupplierService) Create(ctx context.Context, req *model.CreateSupplierRequest) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	supplier := &model.Supplier{
		ID:    uuid.New(),
		Name:  req.Name,
		Phone: req.Phone,
		Notes: req.Notes,
	}
	if err := s.repo.Create(ctx, supplier); err != nil {
		return nil, err
	}
	return supplier, nil
}

func (s *SupplierService) GetByID(ctx context.Context, id uuid.UUID) (*model.Supplier, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SupplierService) List(ctx context.Context, page, limit int) ([]model.Supplier, int, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *SupplierService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateSupplierRequest) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	supplier, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		supplier.Name = *req.Name
	}
	if req.Phone != nil {
		supplier.Phone = req.Phone
	}
	if req.Notes != nil {
		supplier.Notes = req.Notes
	}
	if err := s.repo.Update(ctx, supplier); err != nil {
		return nil, err
	}
	return supplier, nil
}

func (s *SupplierService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
