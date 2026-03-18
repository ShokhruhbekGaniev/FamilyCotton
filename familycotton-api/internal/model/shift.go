package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Shift struct {
	ID             uuid.UUID       `json:"id"`
	OpenedBy       uuid.UUID       `json:"opened_by"`
	ClosedBy       *uuid.UUID      `json:"closed_by"`
	OpenedAt       time.Time       `json:"opened_at"`
	ClosedAt       *time.Time      `json:"closed_at"`
	TotalCash      decimal.Decimal `json:"total_cash"`
	TotalTerminal  decimal.Decimal `json:"total_terminal"`
	TotalOnline    decimal.Decimal `json:"total_online"`
	TotalDebtSales decimal.Decimal `json:"total_debt_sales"`
	Status         string          `json:"status"`
}
