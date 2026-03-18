package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type SaleReturnRepository struct {
	db *pgxpool.Pool
}

func NewSaleReturnRepository(db *pgxpool.Pool) *SaleReturnRepository {
	return &SaleReturnRepository{db: db}
}

func (r *SaleReturnRepository) Create(ctx context.Context, tx DBTX, sr *model.SaleReturn) error {
	return tx.QueryRow(ctx,
		`INSERT INTO sale_returns (id, sale_id, sale_item_id, new_product_id, quantity, return_type, refund_amount, surcharge_amount, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at`,
		sr.ID, sr.SaleID, sr.SaleItemID, sr.NewProductID, sr.Quantity,
		sr.ReturnType, sr.RefundAmount, sr.SurchargeAmount, sr.CreatedBy,
	).Scan(&sr.CreatedAt)
}

func (r *SaleReturnRepository) List(ctx context.Context, saleID *uuid.UUID, page, limit int) ([]model.SaleReturn, int, error) {
	where := ""
	var args []any
	idx := 1

	if saleID != nil {
		where = fmt.Sprintf(" AND sale_id = $%d", idx)
		args = append(args, *saleID)
		idx++
	}

	var total int
	err := r.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM sale_returns WHERE 1=1 %s", where), args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(
			`SELECT id, sale_id, sale_item_id, new_product_id, quantity, return_type,
			        refund_amount, surcharge_amount, created_by, created_at
			 FROM sale_returns WHERE 1=1 %s
			 ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
			where, idx, idx+1,
		), args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var returns []model.SaleReturn
	for rows.Next() {
		var sr model.SaleReturn
		if err := rows.Scan(&sr.ID, &sr.SaleID, &sr.SaleItemID, &sr.NewProductID,
			&sr.Quantity, &sr.ReturnType, &sr.RefundAmount, &sr.SurchargeAmount,
			&sr.CreatedBy, &sr.CreatedAt); err != nil {
			return nil, 0, err
		}
		returns = append(returns, sr)
	}
	return returns, total, rows.Err()
}
