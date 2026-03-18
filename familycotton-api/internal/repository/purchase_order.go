package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/familycotton/api/internal/model"
)

type PurchaseOrderRepository struct {
	db *pgxpool.Pool
}

func NewPurchaseOrderRepository(db *pgxpool.Pool) *PurchaseOrderRepository {
	return &PurchaseOrderRepository{db: db}
}

func (r *PurchaseOrderRepository) Create(ctx context.Context, tx DBTX, po *model.PurchaseOrder) error {
	return tx.QueryRow(ctx,
		`INSERT INTO purchase_orders (id, supplier_id, total_amount, paid_amount, status, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at, updated_at`,
		po.ID, po.SupplierID, po.TotalAmount, po.PaidAmount, po.Status, po.CreatedBy,
	).Scan(&po.CreatedAt, &po.UpdatedAt)
}

func (r *PurchaseOrderRepository) CreateItem(ctx context.Context, tx DBTX, item *model.PurchaseOrderItem) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO purchase_order_items (id, purchase_order_id, product_id, quantity, unit_cost, destination)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		item.ID, item.PurchaseOrderID, item.ProductID, item.Quantity, item.UnitCost, item.Destination,
	)
	return err
}

func (r *PurchaseOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.PurchaseOrder, error) {
	po := &model.PurchaseOrder{}
	err := r.db.QueryRow(ctx,
		`SELECT id, supplier_id, total_amount, paid_amount, status, created_by, created_at, updated_at
		 FROM purchase_orders WHERE id = $1`, id,
	).Scan(&po.ID, &po.SupplierID, &po.TotalAmount, &po.PaidAmount, &po.Status,
		&po.CreatedBy, &po.CreatedAt, &po.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "purchase order not found")
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, purchase_order_id, product_id, quantity, unit_cost, destination
		 FROM purchase_order_items WHERE purchase_order_id = $1`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item model.PurchaseOrderItem
		if err := rows.Scan(&item.ID, &item.PurchaseOrderID, &item.ProductID,
			&item.Quantity, &item.UnitCost, &item.Destination); err != nil {
			return nil, err
		}
		po.Items = append(po.Items, item)
	}
	return po, rows.Err()
}

func (r *PurchaseOrderRepository) List(ctx context.Context, supplierID *uuid.UUID, status string, page, limit int) ([]model.PurchaseOrder, int, error) {
	where := ""
	var args []any
	idx := 1

	if supplierID != nil {
		where += fmt.Sprintf(" AND supplier_id = $%d", idx)
		args = append(args, *supplierID)
		idx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, status)
		idx++
	}

	var total int
	err := r.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM purchase_orders WHERE 1=1 %s", where), args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(
			`SELECT id, supplier_id, total_amount, paid_amount, status, created_by, created_at, updated_at
			 FROM purchase_orders WHERE 1=1 %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
			where, idx, idx+1,
		), args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []model.PurchaseOrder
	for rows.Next() {
		var po model.PurchaseOrder
		if err := rows.Scan(&po.ID, &po.SupplierID, &po.TotalAmount, &po.PaidAmount,
			&po.Status, &po.CreatedBy, &po.CreatedAt, &po.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, po)
	}
	return orders, total, rows.Err()
}

func (r *PurchaseOrderRepository) UpdatePaidAmount(ctx context.Context, tx DBTX, poID uuid.UUID, addAmount decimal.Decimal) error {
	_, err := tx.Exec(ctx,
		`UPDATE purchase_orders SET paid_amount = paid_amount + $1, updated_at = NOW() WHERE id = $2`,
		addAmount, poID,
	)
	return err
}

func (r *PurchaseOrderRepository) RecalcStatus(ctx context.Context, tx DBTX, poID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE purchase_orders SET status = CASE
		   WHEN paid_amount >= total_amount THEN 'paid'
		   WHEN paid_amount > 0 THEN 'partial'
		   ELSE 'unpaid'
		 END, updated_at = NOW()
		 WHERE id = $1`, poID,
	)
	return err
}
