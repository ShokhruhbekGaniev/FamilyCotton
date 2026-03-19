package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type SaleRepository struct {
	db *pgxpool.Pool
}

func NewSaleRepository(db *pgxpool.Pool) *SaleRepository {
	return &SaleRepository{db: db}
}

func (r *SaleRepository) CreateSale(ctx context.Context, tx DBTX, s *model.Sale) error {
	return tx.QueryRow(ctx,
		`INSERT INTO sales (id, shift_id, client_id, total_amount, paid_cash, paid_terminal, paid_online, paid_debt, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at`,
		s.ID, s.ShiftID, s.ClientID, s.TotalAmount,
		s.PaidCash, s.PaidTerminal, s.PaidOnline, s.PaidDebt, s.CreatedBy,
	).Scan(&s.CreatedAt)
}

func (r *SaleRepository) CreateSaleItem(ctx context.Context, tx DBTX, item *model.SaleItem) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO sale_items (id, sale_id, product_id, quantity, unit_price, subtotal)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		item.ID, item.SaleID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal,
	)
	return err
}

func (r *SaleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Sale, error) {
	s := &model.Sale{}
	err := r.db.QueryRow(ctx,
		`SELECT s.id, s.shift_id, s.client_id, s.total_amount, s.paid_cash, s.paid_terminal,
		        s.paid_online, s.paid_debt, s.created_by, s.created_at,
		        u.name, c.name
		 FROM sales s
		 JOIN users u ON u.id = s.created_by
		 LEFT JOIN clients c ON c.id = s.client_id
		 WHERE s.id = $1`, id,
	).Scan(&s.ID, &s.ShiftID, &s.ClientID, &s.TotalAmount,
		&s.PaidCash, &s.PaidTerminal, &s.PaidOnline, &s.PaidDebt,
		&s.CreatedBy, &s.CreatedAt, &s.CreatedByName, &s.ClientName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "sale not found")
	}
	if err != nil {
		return nil, err
	}

	// Load items with product names.
	rows, err := r.db.Query(ctx,
		`SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal
		 FROM sale_items si
		 JOIN products p ON p.id = si.product_id
		 WHERE si.sale_id = $1`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item model.SaleItem
		if err := rows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.ProductName,
			&item.Quantity, &item.UnitPrice, &item.Subtotal); err != nil {
			return nil, err
		}
		s.Items = append(s.Items, item)
	}
	return s, rows.Err()
}

func (r *SaleRepository) List(ctx context.Context, shiftID *uuid.UUID, clientID *uuid.UUID, page, limit int) ([]model.Sale, int, error) {
	where := ""
	var args []any
	idx := 1

	if shiftID != nil {
		where += fmt.Sprintf(" AND s.shift_id = $%d", idx)
		args = append(args, *shiftID)
		idx++
	}
	if clientID != nil {
		where += fmt.Sprintf(" AND s.client_id = $%d", idx)
		args = append(args, *clientID)
		idx++
	}

	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM sales s WHERE 1=1 %s", where)
	err := r.db.QueryRow(ctx, countQ, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, (page-1)*limit)
	listQ := fmt.Sprintf(
		`SELECT s.id, s.shift_id, s.client_id, s.total_amount, s.paid_cash, s.paid_terminal,
		        s.paid_online, s.paid_debt, s.created_by, s.created_at,
		        u.name
		 FROM sales s
		 JOIN users u ON u.id = s.created_by
		 WHERE 1=1 %s
		 ORDER BY s.created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1,
	)

	rows, err := r.db.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sales []model.Sale
	for rows.Next() {
		var s model.Sale
		if err := rows.Scan(&s.ID, &s.ShiftID, &s.ClientID, &s.TotalAmount,
			&s.PaidCash, &s.PaidTerminal, &s.PaidOnline, &s.PaidDebt,
			&s.CreatedBy, &s.CreatedAt, &s.CreatedByName); err != nil {
			return nil, 0, err
		}
		sales = append(sales, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Load items for all sales in one query.
	if len(sales) > 0 {
		saleIDs := make([]uuid.UUID, len(sales))
		saleMap := make(map[uuid.UUID]*model.Sale, len(sales))
		for i := range sales {
			saleIDs[i] = sales[i].ID
			saleMap[sales[i].ID] = &sales[i]
		}

		itemRows, err := r.db.Query(ctx,
			`SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal
			 FROM sale_items si
			 JOIN products p ON p.id = si.product_id
			 WHERE si.sale_id = ANY($1)`, saleIDs,
		)
		if err != nil {
			return nil, 0, err
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var item model.SaleItem
			if err := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.ProductName,
				&item.Quantity, &item.UnitPrice, &item.Subtotal); err != nil {
				return nil, 0, err
			}
			if sale, ok := saleMap[item.SaleID]; ok {
				sale.Items = append(sale.Items, item)
			}
		}
		if err := itemRows.Err(); err != nil {
			return nil, 0, err
		}
	}

	return sales, total, nil
}

// GetSaleItemByID fetches a single sale_item.
func (r *SaleRepository) GetSaleItemByID(ctx context.Context, id uuid.UUID) (*model.SaleItem, error) {
	item := &model.SaleItem{}
	err := r.db.QueryRow(ctx,
		`SELECT id, sale_id, product_id, quantity, unit_price, subtotal
		 FROM sale_items WHERE id = $1`, id,
	).Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "sale item not found")
	}
	return item, err
}

// SumReturnedQty returns total already-returned quantity for a sale_item.
func (r *SaleRepository) SumReturnedQty(ctx context.Context, saleItemID uuid.UUID) (int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity), 0) FROM sale_returns WHERE sale_item_id = $1`, saleItemID,
	).Scan(&total)
	return total, err
}

// HasReturns checks if a sale has any associated returns.
func (r *SaleRepository) HasReturns(ctx context.Context, saleID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sale_returns WHERE sale_id = $1)`, saleID,
	).Scan(&exists)
	return exists, err
}

// GetSaleItemsBySaleID fetches all items for a sale within a transaction.
func (r *SaleRepository) GetSaleItemsBySaleID(ctx context.Context, tx DBTX, saleID uuid.UUID) ([]model.SaleItem, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, sale_id, product_id, quantity, unit_price, subtotal
		 FROM sale_items WHERE sale_id = $1`, saleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.SaleItem
	for rows.Next() {
		var item model.SaleItem
		if err := rows.Scan(&item.ID, &item.SaleID, &item.ProductID,
			&item.Quantity, &item.UnitPrice, &item.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// DeleteSaleItems removes all items for a sale within a transaction.
func (r *SaleRepository) DeleteSaleItems(ctx context.Context, tx DBTX, saleID uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM sale_items WHERE sale_id = $1`, saleID)
	return err
}

// DeleteSale removes a sale within a transaction.
func (r *SaleRepository) DeleteSale(ctx context.Context, tx DBTX, saleID uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM sales WHERE id = $1`, saleID)
	return err
}
