package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SupplierPayment struct {
	ID                uuid.UUID       `json:"id"`
	SupplierID        uuid.UUID       `json:"supplier_id"`
	PurchaseOrderID   *uuid.UUID      `json:"purchase_order_id"`
	PaymentType       string          `json:"payment_type"`
	Amount            decimal.Decimal `json:"amount"`
	ReturnedProductID *uuid.UUID      `json:"returned_product_id"`
	ReturnedQty       *int            `json:"returned_qty"`
	CreatedBy         uuid.UUID       `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
}

type CreateSupplierPaymentRequest struct {
	SupplierID        uuid.UUID       `json:"supplier_id"`
	PurchaseOrderID   *uuid.UUID      `json:"purchase_order_id"`
	PaymentType       string          `json:"payment_type"`
	Amount            decimal.Decimal `json:"amount"`
	ReturnedProductID *uuid.UUID      `json:"returned_product_id"`
	ReturnedQty       *int            `json:"returned_qty"`
}

func (r *CreateSupplierPaymentRequest) Validate() error {
	if r.PaymentType != "money" && r.PaymentType != "product_return" {
		return NewAppError(ErrValidation, "Тип оплаты должен быть 'money' или 'product_return'")
	}
	if r.PaymentType == "money" && r.Amount.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "Сумма должна быть положительной для денежной оплаты")
	}
	if r.PaymentType == "product_return" {
		if r.ReturnedProductID == nil || r.ReturnedQty == nil || *r.ReturnedQty <= 0 {
			return NewAppError(ErrValidation, "Для возврата товара необходимо указать товар и количество")
		}
	}
	return nil
}
