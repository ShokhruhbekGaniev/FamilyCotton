package model

import (
	"time"

	"github.com/google/uuid"
)

type Brand struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	IsDeleted bool      `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateBrandRequest struct {
	Name string `json:"name"`
}

func (r *CreateBrandRequest) Validate() error {
	if r.Name == "" {
		return NewAppError(ErrValidation, "Название бренда обязательно")
	}
	return nil
}

type UpdateBrandRequest struct {
	Name *string `json:"name,omitempty"`
}

func (r *UpdateBrandRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "Название бренда не может быть пустым")
	}
	return nil
}
