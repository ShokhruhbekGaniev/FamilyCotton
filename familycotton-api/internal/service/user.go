package service

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Create(ctx context.Context, req *model.CreateUserRequest) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:           uuid.New(),
		Name:         req.Name,
		Login:        req.Login,
		PasswordHash: string(hash),
		Role:         req.Role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context) ([]model.User, error) {
	return s.repo.List(ctx)
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateUserRequest) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Login != nil {
		user.Login = *req.Login
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hash)
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
