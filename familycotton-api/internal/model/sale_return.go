package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SaleReturn struct {
	ID              uuid.UUID       `json:"id"`
	SaleID          uuid.UUID       `json:"sale_id"`
	SaleItemID      uuid.UUID       `json:"sale_item_id"`
	NewProductID    *uuid.UUID      `json:"new_product_id"`
	Quantity        int             `json:"quantity"`
	ReturnType      string          `json:"return_type"`
	RefundAmount    decimal.Decimal `json:"refund_amount"`
	SurchargeAmount decimal.Decimal `json:"surcharge_amount"`
	CreatedBy       uuid.UUID       `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
}

type CreateSaleReturnRequest struct {
	SaleID       uuid.UUID  `json:"sale_id"`
	SaleItemID   uuid.UUID  `json:"sale_item_id"`
	NewProductID *uuid.UUID `json:"new_product_id"`
	Quantity     int        `json:"quantity"`
	ReturnType   string     `json:"return_type"`
}

func (r *CreateSaleReturnRequest) Validate() error {
	if r.Quantity <= 0 {
		return NewAppError(ErrValidation, "quantity must be positive")
	}
	validTypes := map[string]bool{"full": true, "exchange": true, "exchange_diff": true}
	if !validTypes[r.ReturnType] {
		return NewAppError(ErrValidation, "return_type must be 'full', 'exchange', or 'exchange_diff'")
	}
	if (r.ReturnType == "exchange" || r.ReturnType == "exchange_diff") && r.NewProductID == nil {
		return NewAppError(ErrValidation, "new_product_id is required for exchange returns")
	}
	return nil
}
