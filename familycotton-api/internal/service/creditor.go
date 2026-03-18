package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type CreditorService struct {
	repo *repository.CreditorRepository
}

func NewCreditorService(repo *repository.CreditorRepository) *CreditorService {
	return &CreditorService{repo: repo}
}

func (s *CreditorService) Create(ctx context.Context, req *model.CreateCreditorRequest) (*model.Creditor, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	creditor := &model.Creditor{
		ID:    uuid.New(),
		Name:  req.Name,
		Phone: req.Phone,
		Notes: req.Notes,
	}
	if err := s.repo.Create(ctx, creditor); err != nil {
		return nil, err
	}
	return creditor, nil
}

func (s *CreditorService) GetByID(ctx context.Context, id uuid.UUID) (*model.Creditor, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CreditorService) List(ctx context.Context, page, limit int) ([]model.Creditor, int, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *CreditorService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateCreditorRequest) (*model.Creditor, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	creditor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		creditor.Name = *req.Name
	}
	if req.Phone != nil {
		creditor.Phone = req.Phone
	}
	if req.Notes != nil {
		creditor.Notes = req.Notes
	}
	if err := s.repo.Update(ctx, creditor); err != nil {
		return nil, err
	}
	return creditor, nil
}

func (s *CreditorService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
