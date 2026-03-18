package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Product struct {
	ID           uuid.UUID       `json:"id"`
	SKU          string          `json:"sku"`
	Name         string          `json:"name"`
	Brand        *string         `json:"brand"`
	SupplierID   *uuid.UUID      `json:"supplier_id"`
	PhotoURL     *string         `json:"photo_url"`
	CostPrice    decimal.Decimal `json:"cost_price"`
	SellPrice    decimal.Decimal `json:"sell_price"`
	Margin       decimal.Decimal `json:"margin"`
	QtyShop      int             `json:"qty_shop"`
	QtyWarehouse int             `json:"qty_warehouse"`
	IsDeleted    bool            `json:"-"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CreateProductRequest struct {
	SKU        string          `json:"sku"`
	Name       string          `json:"name"`
	Brand      *string         `json:"brand"`
	SupplierID *uuid.UUID      `json:"supplier_id"`
	PhotoURL   *string         `json:"photo_url"`
	CostPrice  decimal.Decimal `json:"cost_price"`
	SellPrice  decimal.Decimal `json:"sell_price"`
}

func (r *CreateProductRequest) Validate() error {
	if r.SKU == "" {
		return NewAppError(ErrValidation, "sku is required")
	}
	if r.Name == "" {
		return NewAppError(ErrValidation, "name is required")
	}
	if r.CostPrice.IsNegative() {
		return NewAppError(ErrValidation, "cost_price cannot be negative")
	}
	if r.SellPrice.IsNegative() {
		return NewAppError(ErrValidation, "sell_price cannot be negative")
	}
	return nil
}

type UpdateProductRequest struct {
	SKU        *string          `json:"sku,omitempty"`
	Name       *string          `json:"name,omitempty"`
	Brand      *string          `json:"brand,omitempty"`
	SupplierID *uuid.UUID       `json:"supplier_id,omitempty"`
	PhotoURL   *string          `json:"photo_url,omitempty"`
	CostPrice  *decimal.Decimal `json:"cost_price,omitempty"`
	SellPrice  *decimal.Decimal `json:"sell_price,omitempty"`
}

func (r *UpdateProductRequest) Validate() error {
	if r.SKU != nil && *r.SKU == "" {
		return NewAppError(ErrValidation, "sku cannot be empty")
	}
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "name cannot be empty")
	}
	if r.CostPrice != nil && r.CostPrice.IsNegative() {
		return NewAppError(ErrValidation, "cost_price cannot be negative")
	}
	if r.SellPrice != nil && r.SellPrice.IsNegative() {
		return NewAppError(ErrValidation, "sell_price cannot be negative")
	}
	return nil
}

type ProductFilter struct {
	Search     string
	SupplierID *uuid.UUID
	Brand      string
}
