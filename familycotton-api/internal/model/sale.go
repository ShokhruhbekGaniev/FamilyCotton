package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Sale is the full detail model (used by GetByID / show page).
type Sale struct {
	ID             uuid.UUID       `json:"id"`
	ShiftID        uuid.UUID       `json:"shift_id"`
	ClientID       *uuid.UUID      `json:"client_id"`
	TotalAmount    decimal.Decimal `json:"total_amount"`
	DiscountType   string          `json:"discount_type"`
	DiscountValue  decimal.Decimal `json:"discount_value"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	PaidCash       decimal.Decimal `json:"paid_cash"`
	PaidTerminal   decimal.Decimal `json:"paid_terminal"`
	PaidOnline     decimal.Decimal `json:"paid_online"`
	PaidDebt       decimal.Decimal `json:"paid_debt"`
	CreatedBy      uuid.UUID       `json:"created_by"`
	CreatedByName  string          `json:"created_by_name"`
	ClientName     *string         `json:"client_name,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	Items          []SaleItem      `json:"items,omitempty"`
}

// SaleListItem is a lightweight model for the list endpoint.
type SaleListItem struct {
	ID            uuid.UUID  `json:"id"`
	ShiftID       uuid.UUID  `json:"shift_id"`
	ClientID      *uuid.UUID `json:"client_id"`
	TotalAmount   string     `json:"total_amount"`
	PaymentType   string     `json:"payment_type"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	CreatedByName string     `json:"created_by_name"`
	ClientName    *string    `json:"client_name,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	Items         []SaleItem `json:"items,omitempty"`
}

// ComputePaymentType returns "cash", "terminal", "online", "debt", or "mix".
func ComputePaymentType(cash, terminal, online, debt decimal.Decimal) string {
	count := 0
	last := ""
	if cash.IsPositive() {
		count++
		last = "cash"
	}
	if terminal.IsPositive() {
		count++
		last = "terminal"
	}
	if online.IsPositive() {
		count++
		last = "online"
	}
	if debt.IsPositive() {
		count++
		last = "debt"
	}
	if count > 1 {
		return "mix"
	}
	if count == 0 {
		return "cash"
	}
	return last
}

type SaleItem struct {
	ID          uuid.UUID       `json:"id"`
	SaleID      uuid.UUID       `json:"sale_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name"`
	Quantity    int             `json:"quantity"`
	ReturnedQty int             `json:"returned_qty"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Subtotal    decimal.Decimal `json:"subtotal"`
}

type CreateSaleItemRequest struct {
	ProductID uuid.UUID       `json:"product_id"`
	Quantity  int             `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
}

type CreateSaleRequest struct {
	ClientID      *uuid.UUID              `json:"client_id"`
	Items         []CreateSaleItemRequest `json:"items"`
	DiscountType  string                  `json:"discount_type"`
	DiscountValue decimal.Decimal         `json:"discount_value"`
	PaidCash      decimal.Decimal         `json:"paid_cash"`
	PaidTerminal  decimal.Decimal         `json:"paid_terminal"`
	PaidOnline    decimal.Decimal         `json:"paid_online"`
	PaidDebt      decimal.Decimal         `json:"paid_debt"`
}

func (r *CreateSaleRequest) Validate() error {
	if len(r.Items) == 0 {
		return NewAppError(ErrValidation, "Необходимо добавить хотя бы один товар")
	}
	for _, item := range r.Items {
		if item.Quantity <= 0 {
			return NewAppError(ErrValidation, "Количество товара должно быть положительным")
		}
		if item.UnitPrice.IsNegative() {
			return NewAppError(ErrValidation, "Цена товара не может быть отрицательной")
		}
	}
	// Default to "none" if empty.
	if r.DiscountType == "" {
		r.DiscountType = "none"
	}
	validDiscountTypes := map[string]bool{"none": true, "percent": true, "fixed": true}
	if !validDiscountTypes[r.DiscountType] {
		return NewAppError(ErrValidation, "Тип скидки должен быть 'none', 'percent' или 'fixed'")
	}
	if r.DiscountValue.IsNegative() {
		return NewAppError(ErrValidation, "Сумма скидки не может быть отрицательной")
	}
	if r.DiscountType == "percent" && r.DiscountValue.GreaterThan(decimal.NewFromInt(100)) {
		return NewAppError(ErrValidation, "Процент скидки не может превышать 100")
	}
	if r.PaidCash.IsNegative() || r.PaidTerminal.IsNegative() || r.PaidOnline.IsNegative() || r.PaidDebt.IsNegative() {
		return NewAppError(ErrValidation, "Сумма оплаты не может быть отрицательной")
	}
	if r.PaidDebt.IsPositive() && r.ClientID == nil {
		return NewAppError(ErrValidation, "Клиент обязателен при оплате в долг")
	}
	return nil
}
