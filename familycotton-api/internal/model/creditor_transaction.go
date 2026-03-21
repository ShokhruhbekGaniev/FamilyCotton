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
		return NewAppError(ErrValidation, "Тип должен быть 'receive' или 'repay'")
	}
	if r.Currency != "UZS" && r.Currency != "USD" {
		return NewAppError(ErrValidation, "Валюта должна быть 'UZS' или 'USD'")
	}
	if r.Amount.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "Сумма должна быть положительной")
	}
	if r.ExchangeRate.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "Курс обмена должен быть положительным")
	}
	return nil
}
