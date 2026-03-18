package model

import (
	"time"

	"github.com/google/uuid"
)

type InventoryCheck struct {
	ID          uuid.UUID            `json:"id"`
	Location    string               `json:"location"`
	CheckedBy   uuid.UUID            `json:"checked_by"`
	Status      string               `json:"status"`
	CreatedAt   time.Time            `json:"created_at"`
	CompletedAt *time.Time           `json:"completed_at"`
	Items       []InventoryCheckItem `json:"items,omitempty"`
}

type InventoryCheckItem struct {
	ID               uuid.UUID `json:"id"`
	InventoryCheckID uuid.UUID `json:"inventory_check_id"`
	ProductID        uuid.UUID `json:"product_id"`
	ExpectedQty      int       `json:"expected_qty"`
	ActualQty        *int      `json:"actual_qty"`
	Difference       *int      `json:"difference"`
}

type CreateInventoryCheckRequest struct {
	Location string `json:"location"`
}

func (r *CreateInventoryCheckRequest) Validate() error {
	if r.Location != "shop" && r.Location != "warehouse" {
		return NewAppError(ErrValidation, "location must be 'shop' or 'warehouse'")
	}
	return nil
}

type UpdateInventoryCheckItemRequest struct {
	ItemID    uuid.UUID `json:"item_id"`
	ActualQty int       `json:"actual_qty"`
}

type UpdateInventoryCheckRequest struct {
	Items  []UpdateInventoryCheckItemRequest `json:"items"`
	Status *string                           `json:"status,omitempty"`
}
