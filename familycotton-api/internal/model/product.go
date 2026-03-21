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
		return NewAppError(ErrValidation, "Артикул обязателен")
	}
	if r.Name == "" {
		return NewAppError(ErrValidation, "Название обязательно")
	}
	if r.CostPrice.IsNegative() {
		return NewAppError(ErrValidation, "Себестоимость не может быть отрицательной")
	}
	if r.SellPrice.IsNegative() {
		return NewAppError(ErrValidation, "Цена продажи не может быть отрицательной")
	}
	return nil
}

type UpdateProductRequest struct {
	SKU          *string          `json:"sku,omitempty"`
	Name         *string          `json:"name,omitempty"`
	Brand        *string          `json:"brand,omitempty"`
	SupplierID   *uuid.UUID       `json:"supplier_id,omitempty"`
	PhotoURL     *string          `json:"photo_url,omitempty"`
	CostPrice    *decimal.Decimal `json:"cost_price,omitempty"`
	SellPrice    *decimal.Decimal `json:"sell_price,omitempty"`
	QtyShop      *int             `json:"qty_shop,omitempty"`
	QtyWarehouse *int             `json:"qty_warehouse,omitempty"`
}

func (r *UpdateProductRequest) Validate() error {
	if r.SKU != nil && *r.SKU == "" {
		return NewAppError(ErrValidation, "Артикул не может быть пустым")
	}
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "Название не может быть пустым")
	}
	if r.CostPrice != nil && r.CostPrice.IsNegative() {
		return NewAppError(ErrValidation, "Себестоимость не может быть отрицательной")
	}
	if r.SellPrice != nil && r.SellPrice.IsNegative() {
		return NewAppError(ErrValidation, "Цена продажи не может быть отрицательной")
	}
	if r.QtyShop != nil && *r.QtyShop < 0 {
		return NewAppError(ErrValidation, "Количество в магазине не может быть отрицательным")
	}
	if r.QtyWarehouse != nil && *r.QtyWarehouse < 0 {
		return NewAppError(ErrValidation, "Количество на складе не может быть отрицательным")
	}
	return nil
}

type ProductFilter struct {
	Search     string
	SupplierID *uuid.UUID
	Brand      string
}
