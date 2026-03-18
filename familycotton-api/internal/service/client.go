package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type ClientService struct {
	repo *repository.ClientRepository
}

func NewClientService(repo *repository.ClientRepository) *ClientService {
	return &ClientService{repo: repo}
}

func (s *ClientService) Create(ctx context.Context, req *model.CreateClientRequest) (*model.Client, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	client := &model.Client{
		ID:    uuid.New(),
		Name:  req.Name,
		Phone: req.Phone,
	}
	if err := s.repo.Create(ctx, client); err != nil {
		return nil, err
	}
	return client, nil
}

func (s *ClientService) GetByID(ctx context.Context, id uuid.UUID) (*model.Client, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ClientService) List(ctx context.Context, page, limit int) ([]model.Client, int, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *ClientService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateClientRequest) (*model.Client, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	client, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		client.Name = *req.Name
	}
	if req.Phone != nil {
		client.Phone = req.Phone
	}
	if err := s.repo.Update(ctx, client); err != nil {
		return nil, err
	}
	return client, nil
}

func (s *ClientService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
