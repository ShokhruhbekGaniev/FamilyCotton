package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/familycotton/api/internal/model"
)

type OwnerDebtRepository struct {
	db *pgxpool.Pool
}

func NewOwnerDebtRepository(db *pgxpool.Pool) *OwnerDebtRepository {
	return &OwnerDebtRepository{db: db}
}

func (r *OwnerDebtRepository) Create(ctx context.Context, tx DBTX, od *model.OwnerDebt) error {
	od.ID = uuid.New()
	return tx.QueryRow(ctx,
		`INSERT INTO owner_debts (id, shift_id, amount) VALUES ($1, $2, $3)
		 RETURNING created_at`,
		od.ID, od.ShiftID, od.Amount,
	).Scan(&od.CreatedAt)
}

func (r *OwnerDebtRepository) ListUnsettled(ctx context.Context) ([]model.OwnerDebt, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, shift_id, amount, is_settled, created_at, settled_at
		 FROM owner_debts WHERE is_settled = false ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var debts []model.OwnerDebt
	for rows.Next() {
		var d model.OwnerDebt
		if err := rows.Scan(&d.ID, &d.ShiftID, &d.Amount, &d.IsSettled, &d.CreatedAt, &d.SettledAt); err != nil {
			return nil, err
		}
		debts = append(debts, d)
	}
	return debts, rows.Err()
}

func (r *OwnerDebtRepository) Settle(ctx context.Context, tx DBTX, id uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE owner_debts SET is_settled = true, settled_at = NOW() WHERE id = $1 AND is_settled = false`,
		id,
	)
	return err
}

func (r *OwnerDebtRepository) TotalUnsettled(ctx context.Context) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM owner_debts WHERE is_settled = false`,
	).Scan(&total)
	return total, err
}
