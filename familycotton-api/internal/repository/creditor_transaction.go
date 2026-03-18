package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type CreditorTransactionRepository struct {
	db *pgxpool.Pool
}

func NewCreditorTransactionRepository(db *pgxpool.Pool) *CreditorTransactionRepository {
	return &CreditorTransactionRepository{db: db}
}

func (r *CreditorTransactionRepository) Create(ctx context.Context, tx DBTX, ct *model.CreditorTransaction) error {
	return tx.QueryRow(ctx,
		`INSERT INTO creditor_transactions (id, creditor_id, type, currency, amount, exchange_rate, amount_uzs, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at`,
		ct.ID, ct.CreditorID, ct.Type, ct.Currency, ct.Amount, ct.ExchangeRate, ct.AmountUZS, ct.CreatedBy,
	).Scan(&ct.CreatedAt)
}

func (r *CreditorTransactionRepository) ListByCreditor(ctx context.Context, creditorID uuid.UUID, page, limit int) ([]model.CreditorTransaction, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM creditor_transactions WHERE creditor_id = $1`, creditorID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, creditor_id, type, currency, amount, exchange_rate, amount_uzs, created_by, created_at
		 FROM creditor_transactions WHERE creditor_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		creditorID, limit, (page-1)*limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txns []model.CreditorTransaction
	for rows.Next() {
		var ct model.CreditorTransaction
		if err := rows.Scan(&ct.ID, &ct.CreditorID, &ct.Type, &ct.Currency,
			&ct.Amount, &ct.ExchangeRate, &ct.AmountUZS, &ct.CreatedBy, &ct.CreatedAt); err != nil {
			return nil, 0, err
		}
		txns = append(txns, ct)
	}
	return txns, total, rows.Err()
}
