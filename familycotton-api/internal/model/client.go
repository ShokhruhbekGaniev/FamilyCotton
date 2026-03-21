package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Client struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Phone     *string         `json:"phone"`
	TotalDebt decimal.Decimal `json:"total_debt"`
	IsDeleted bool            `json:"-"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateClientRequest struct {
	Name  string  `json:"name"`
	Phone *string `json:"phone"`
}

func (r *CreateClientRequest) Validate() error {
	if r.Name == "" {
		return NewAppError(ErrValidation, "Имя обязательно")
	}
	return nil
}

type UpdateClientRequest struct {
	Name  *string `json:"name,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

func (r *UpdateClientRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "Имя не может быть пустым")
	}
	return nil
}
