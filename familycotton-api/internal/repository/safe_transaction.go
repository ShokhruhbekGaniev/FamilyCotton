package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type SafeTransactionRepository struct {
	db *pgxpool.Pool
}

func NewSafeTransactionRepository(db *pgxpool.Pool) *SafeTransactionRepository {
	return &SafeTransactionRepository{db: db}
}

func (r *SafeTransactionRepository) Create(ctx context.Context, tx DBTX, st *model.SafeTransaction) error {
	st.ID = uuid.New()
	_, err := tx.Exec(ctx,
		`INSERT INTO safe_transactions (id, type, source, balance_type, amount, description, reference_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		st.ID, st.Type, st.Source, st.BalanceType, st.Amount, st.Description, st.ReferenceID,
	)
	return err
}

func (r *SafeTransactionRepository) GetBalance(ctx context.Context) (*model.SafeBalance, error) {
	b := &model.SafeBalance{}
	err := r.db.QueryRow(ctx,
		`SELECT
		   COALESCE(SUM(CASE WHEN balance_type='cash' AND type='income' THEN amount ELSE 0 END), 0) -
		   COALESCE(SUM(CASE WHEN balance_type='cash' AND type='expense' THEN amount ELSE 0 END), 0),
		   COALESCE(SUM(CASE WHEN balance_type='terminal' AND type='income' THEN amount ELSE 0 END), 0) -
		   COALESCE(SUM(CASE WHEN balance_type='terminal' AND type='expense' THEN amount ELSE 0 END), 0),
		   COALESCE(SUM(CASE WHEN balance_type='online' AND type='income' THEN amount ELSE 0 END), 0) -
		   COALESCE(SUM(CASE WHEN balance_type='online' AND type='expense' THEN amount ELSE 0 END), 0)
		 FROM safe_transactions`,
	).Scan(&b.Cash, &b.Terminal, &b.Online)
	return b, err
}

func (r *SafeTransactionRepository) List(ctx context.Context, page, limit int) ([]model.SafeTransaction, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM safe_transactions`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, type, source, balance_type, amount, description, reference_id, created_at
		 FROM safe_transactions ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txns []model.SafeTransaction
	for rows.Next() {
		var t model.SafeTransaction
		if err := rows.Scan(&t.ID, &t.Type, &t.Source, &t.BalanceType,
			&t.Amount, &t.Description, &t.ReferenceID, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		txns = append(txns, t)
	}
	return txns, total, rows.Err()
}
