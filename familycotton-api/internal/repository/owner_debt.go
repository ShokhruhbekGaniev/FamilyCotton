package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
)

type OwnerDebtRepository struct{}

func NewOwnerDebtRepository() *OwnerDebtRepository {
	return &OwnerDebtRepository{}
}

func (r *OwnerDebtRepository) Create(ctx context.Context, tx DBTX, od *model.OwnerDebt) error {
	od.ID = uuid.New()
	return tx.QueryRow(ctx,
		`INSERT INTO owner_debts (id, shift_id, amount) VALUES ($1, $2, $3)
		 RETURNING created_at`,
		od.ID, od.ShiftID, od.Amount,
	).Scan(&od.CreatedAt)
}
