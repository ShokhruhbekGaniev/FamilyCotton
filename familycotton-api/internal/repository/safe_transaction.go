package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
)

type SafeTransactionRepository struct{}

func NewSafeTransactionRepository() *SafeTransactionRepository {
	return &SafeTransactionRepository{}
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
