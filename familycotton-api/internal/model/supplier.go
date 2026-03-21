package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Supplier struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Phone     *string         `json:"phone"`
	Notes     *string         `json:"notes"`
	TotalDebt decimal.Decimal `json:"total_debt"`
	IsDeleted bool            `json:"-"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateSupplierRequest struct {
	Name  string  `json:"name"`
	Phone *string `json:"phone"`
	Notes *string `json:"notes"`
}

func (r *CreateSupplierRequest) Validate() error {
	if r.Name == "" {
		return NewAppError(ErrValidation, "Название обязательно")
	}
	return nil
}

type UpdateSupplierRequest struct {
	Name  *string `json:"name,omitempty"`
	Phone *string `json:"phone,omitempty"`
	Notes *string `json:"notes,omitempty"`
}

func (r *UpdateSupplierRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "Название не может быть пустым")
	}
	return nil
}
