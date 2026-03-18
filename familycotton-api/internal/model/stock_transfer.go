package model

import (
	"time"

	"github.com/google/uuid"
)

type StockTransfer struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	Direction string    `json:"direction"`
	Quantity  int       `json:"quantity"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateStockTransferRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Direction string    `json:"direction"`
	Quantity  int       `json:"quantity"`
}

func (r *CreateStockTransferRequest) Validate() error {
	if r.Direction != "warehouse_to_shop" && r.Direction != "shop_to_warehouse" {
		return NewAppError(ErrValidation, "direction must be 'warehouse_to_shop' or 'shop_to_warehouse'")
	}
	if r.Quantity <= 0 {
		return NewAppError(ErrValidation, "quantity must be positive")
	}
	return nil
}
