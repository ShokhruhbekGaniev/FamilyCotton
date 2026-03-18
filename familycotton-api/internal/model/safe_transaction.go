package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SafeTransaction struct {
	ID          uuid.UUID       `json:"id"`
	Type        string          `json:"type"`
	Source      string          `json:"source"`
	BalanceType string          `json:"balance_type"`
	Amount      decimal.Decimal `json:"amount"`
	Description *string         `json:"description"`
	ReferenceID *uuid.UUID      `json:"reference_id"`
	CreatedAt   time.Time       `json:"created_at"`
}
