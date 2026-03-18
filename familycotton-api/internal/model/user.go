package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	IsDeleted    bool      `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateUserRequest struct {
	Name     string `json:"name"`
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (r *CreateUserRequest) Validate() error {
	if r.Name == "" {
		return NewAppError(ErrValidation, "name is required")
	}
	if r.Login == "" {
		return NewAppError(ErrValidation, "login is required")
	}
	if len(r.Password) < 6 {
		return NewAppError(ErrValidation, "password must be at least 6 characters")
	}
	if r.Role != "owner" && r.Role != "employee" {
		return NewAppError(ErrValidation, "role must be 'owner' or 'employee'")
	}
	return nil
}

type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty"`
	Login    *string `json:"login,omitempty"`
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"`
}

func (r *UpdateUserRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "name cannot be empty")
	}
	if r.Login != nil && *r.Login == "" {
		return NewAppError(ErrValidation, "login cannot be empty")
	}
	if r.Password != nil && len(*r.Password) < 6 {
		return NewAppError(ErrValidation, "password must be at least 6 characters")
	}
	if r.Role != nil && *r.Role != "owner" && *r.Role != "employee" {
		return NewAppError(ErrValidation, "role must be 'owner' or 'employee'")
	}
	return nil
}
