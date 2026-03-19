package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Sale struct {
	ID            uuid.UUID       `json:"id"`
	ShiftID       uuid.UUID       `json:"shift_id"`
	ClientID      *uuid.UUID      `json:"client_id"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	PaidCash      decimal.Decimal `json:"paid_cash"`
	PaidTerminal  decimal.Decimal `json:"paid_terminal"`
	PaidOnline    decimal.Decimal `json:"paid_online"`
	PaidDebt      decimal.Decimal `json:"paid_debt"`
	CreatedBy     uuid.UUID       `json:"created_by"`
	CreatedByName string          `json:"created_by_name"`
	ClientName    *string         `json:"client_name,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	Items         []SaleItem      `json:"items,omitempty"`
}

type SaleItem struct {
	ID          uuid.UUID       `json:"id"`
	SaleID      uuid.UUID       `json:"sale_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Subtotal    decimal.Decimal `json:"subtotal"`
}

type CreateSaleItemRequest struct {
	ProductID uuid.UUID       `json:"product_id"`
	Quantity  int             `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
}

type CreateSaleRequest struct {
	ClientID     *uuid.UUID              `json:"client_id"`
	Items        []CreateSaleItemRequest `json:"items"`
	PaidCash     decimal.Decimal         `json:"paid_cash"`
	PaidTerminal decimal.Decimal         `json:"paid_terminal"`
	PaidOnline   decimal.Decimal         `json:"paid_online"`
	PaidDebt     decimal.Decimal         `json:"paid_debt"`
}

func (r *CreateSaleRequest) Validate() error {
	if len(r.Items) == 0 {
		return NewAppError(ErrValidation, "at least one item is required")
	}
	for _, item := range r.Items {
		if item.Quantity <= 0 {
			return NewAppError(ErrValidation, "item quantity must be positive")
		}
		if item.UnitPrice.IsNegative() {
			return NewAppError(ErrValidation, "item unit_price cannot be negative")
		}
	}
	if r.PaidCash.IsNegative() || r.PaidTerminal.IsNegative() || r.PaidOnline.IsNegative() || r.PaidDebt.IsNegative() {
		return NewAppError(ErrValidation, "payment amounts cannot be negative")
	}
	if r.PaidDebt.IsPositive() && r.ClientID == nil {
		return NewAppError(ErrValidation, "client_id is required when paid_debt > 0")
	}
	return nil
}
