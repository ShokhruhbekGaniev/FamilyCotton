package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreditorTransaction struct {
	ID           uuid.UUID       `json:"id"`
	CreditorID   uuid.UUID       `json:"creditor_id"`
	Type         string          `json:"type"`
	Currency     string          `json:"currency"`
	Amount       decimal.Decimal `json:"amount"`
	ExchangeRate decimal.Decimal `json:"exchange_rate"`
	AmountUZS    decimal.Decimal `json:"amount_uzs"`
	CreatedBy    uuid.UUID       `json:"created_by"`
	CreatedAt    time.Time       `json:"created_at"`
}

type CreateCreditorTransactionRequest struct {
	CreditorID   uuid.UUID       `json:"creditor_id"`
	Type         string          `json:"type"`
	Currency     string          `json:"currency"`
	Amount       decimal.Decimal `json:"amount"`
	ExchangeRate decimal.Decimal `json:"exchange_rate"`
}

func (r *CreateCreditorTransactionRequest) Validate() error {
	if r.Type != "receive" && r.Type != "repay" {
		return NewAppError(ErrValidation, "type must be 'receive' or 'repay'")
	}
	if r.Currency != "UZS" && r.Currency != "USD" {
		return NewAppError(ErrValidation, "currency must be 'UZS' or 'USD'")
	}
	if r.Amount.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "amount must be positive")
	}
	if r.ExchangeRate.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "exchange_rate must be positive")
	}
	return nil
}
