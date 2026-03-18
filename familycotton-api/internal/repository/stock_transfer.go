package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type StockTransferRepository struct {
	db *pgxpool.Pool
}

func NewStockTransferRepository(db *pgxpool.Pool) *StockTransferRepository {
	return &StockTransferRepository{db: db}
}

func (r *StockTransferRepository) Create(ctx context.Context, tx DBTX, st *model.StockTransfer) error {
	return tx.QueryRow(ctx,
		`INSERT INTO stock_transfers (id, product_id, direction, quantity, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		st.ID, st.ProductID, st.Direction, st.Quantity, st.CreatedBy,
	).Scan(&st.CreatedAt)
}
