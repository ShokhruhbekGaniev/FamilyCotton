package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ClientPayment struct {
	ID            uuid.UUID       `json:"id"`
	ClientID      uuid.UUID       `json:"client_id"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
	CreatedBy     uuid.UUID       `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`
}

type CreateClientPaymentRequest struct {
	ClientID      uuid.UUID       `json:"client_id"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
}

func (r *CreateClientPaymentRequest) Validate() error {
	if r.Amount.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "amount must be positive")
	}
	validMethods := map[string]bool{"cash": true, "terminal": true, "online": true}
	if !validMethods[r.PaymentMethod] {
		return NewAppError(ErrValidation, "payment_method must be 'cash', 'terminal', or 'online'")
	}
	return nil
}
