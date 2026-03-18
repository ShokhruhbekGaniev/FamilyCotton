package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PurchaseOrder struct {
	ID          uuid.UUID           `json:"id"`
	SupplierID  uuid.UUID           `json:"supplier_id"`
	TotalAmount decimal.Decimal     `json:"total_amount"`
	PaidAmount  decimal.Decimal     `json:"paid_amount"`
	Status      string              `json:"status"`
	CreatedBy   uuid.UUID           `json:"created_by"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Items       []PurchaseOrderItem `json:"items,omitempty"`
}

type PurchaseOrderItem struct {
	ID              uuid.UUID       `json:"id"`
	PurchaseOrderID uuid.UUID       `json:"purchase_order_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	Quantity        int             `json:"quantity"`
	UnitCost        decimal.Decimal `json:"unit_cost"`
	Destination     string          `json:"destination"`
}

type CreatePurchaseOrderItemRequest struct {
	ProductID   uuid.UUID       `json:"product_id"`
	Quantity    int             `json:"quantity"`
	UnitCost    decimal.Decimal `json:"unit_cost"`
	Destination string          `json:"destination"`
}

type CreatePurchaseOrderRequest struct {
	SupplierID uuid.UUID                       `json:"supplier_id"`
	Items      []CreatePurchaseOrderItemRequest `json:"items"`
	PaidAmount decimal.Decimal                  `json:"paid_amount"`
}

func (r *CreatePurchaseOrderRequest) Validate() error {
	if len(r.Items) == 0 {
		return NewAppError(ErrValidation, "at least one item is required")
	}
	if r.PaidAmount.IsNegative() {
		return NewAppError(ErrValidation, "paid_amount cannot be negative")
	}
	for _, item := range r.Items {
		if item.Quantity <= 0 {
			return NewAppError(ErrValidation, "item quantity must be positive")
		}
		if item.UnitCost.IsNegative() {
			return NewAppError(ErrValidation, "unit_cost cannot be negative")
		}
		if item.Destination != "shop" && item.Destination != "warehouse" {
			return NewAppError(ErrValidation, "destination must be 'shop' or 'warehouse'")
		}
	}
	return nil
}
