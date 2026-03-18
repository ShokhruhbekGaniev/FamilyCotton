package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type SupplierPaymentRepository struct {
	db *pgxpool.Pool
}

func NewSupplierPaymentRepository(db *pgxpool.Pool) *SupplierPaymentRepository {
	return &SupplierPaymentRepository{db: db}
}

func (r *SupplierPaymentRepository) Create(ctx context.Context, tx DBTX, sp *model.SupplierPayment) error {
	return tx.QueryRow(ctx,
		`INSERT INTO supplier_payments (id, supplier_id, purchase_order_id, payment_type, amount, returned_product_id, returned_qty, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at`,
		sp.ID, sp.SupplierID, sp.PurchaseOrderID, sp.PaymentType, sp.Amount,
		sp.ReturnedProductID, sp.ReturnedQty, sp.CreatedBy,
	).Scan(&sp.CreatedAt)
}
