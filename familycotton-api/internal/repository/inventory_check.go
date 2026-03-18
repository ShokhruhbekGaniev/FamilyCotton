package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type InventoryCheckRepository struct {
	db *pgxpool.Pool
}

func NewInventoryCheckRepository(db *pgxpool.Pool) *InventoryCheckRepository {
	return &InventoryCheckRepository{db: db}
}

func (r *InventoryCheckRepository) Create(ctx context.Context, ic *model.InventoryCheck) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO inventory_checks (id, location, checked_by)
		 VALUES ($1, $2, $3)
		 RETURNING status, created_at`,
		ic.ID, ic.Location, ic.CheckedBy,
	).Scan(&ic.Status, &ic.CreatedAt)
}

func (r *InventoryCheckRepository) CreateItem(ctx context.Context, tx DBTX, item *model.InventoryCheckItem) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO inventory_check_items (id, inventory_check_id, product_id, expected_qty)
		 VALUES ($1, $2, $3, $4)`,
		item.ID, item.InventoryCheckID, item.ProductID, item.ExpectedQty,
	)
	return err
}

func (r *InventoryCheckRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.InventoryCheck, error) {
	ic := &model.InventoryCheck{}
	err := r.db.QueryRow(ctx,
		`SELECT id, location, checked_by, status, created_at, completed_at
		 FROM inventory_checks WHERE id = $1`, id,
	).Scan(&ic.ID, &ic.Location, &ic.CheckedBy, &ic.Status, &ic.CreatedAt, &ic.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "inventory check not found")
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, inventory_check_id, product_id, expected_qty, actual_qty, difference
		 FROM inventory_check_items WHERE inventory_check_id = $1`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item model.InventoryCheckItem
		if err := rows.Scan(&item.ID, &item.InventoryCheckID, &item.ProductID,
			&item.ExpectedQty, &item.ActualQty, &item.Difference); err != nil {
			return nil, err
		}
		ic.Items = append(ic.Items, item)
	}
	return ic, rows.Err()
}

func (r *InventoryCheckRepository) UpdateItemActualQty(ctx context.Context, tx DBTX, itemID uuid.UUID, actualQty int) error {
	_, err := tx.Exec(ctx,
		`UPDATE inventory_check_items SET actual_qty = $1, difference = $1 - expected_qty
		 WHERE id = $2`,
		actualQty, itemID,
	)
	return err
}

func (r *InventoryCheckRepository) Complete(ctx context.Context, tx DBTX, id uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE inventory_checks SET status = 'completed', completed_at = NOW()
		 WHERE id = $1 AND status = 'in_progress'`,
		id,
	)
	return err
}

func (r *InventoryCheckRepository) GetItemsByCheckID(ctx context.Context, checkID uuid.UUID) ([]model.InventoryCheckItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, inventory_check_id, product_id, expected_qty, actual_qty, difference
		 FROM inventory_check_items WHERE inventory_check_id = $1`, checkID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.InventoryCheckItem
	for rows.Next() {
		var item model.InventoryCheckItem
		if err := rows.Scan(&item.ID, &item.InventoryCheckID, &item.ProductID,
			&item.ExpectedQty, &item.ActualQty, &item.Difference); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
