package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OwnerDebt struct {
	ID        uuid.UUID       `json:"id"`
	ShiftID   uuid.UUID       `json:"shift_id"`
	Amount    decimal.Decimal `json:"amount"`
	IsSettled bool            `json:"is_settled"`
	CreatedAt time.Time       `json:"created_at"`
	SettledAt *time.Time      `json:"settled_at"`
}
