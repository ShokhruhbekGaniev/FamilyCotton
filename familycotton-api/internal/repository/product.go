package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type ProductRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, p *model.Product) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO products (id, sku, name, brand, supplier_id, photo_url, cost_price, sell_price)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at, updated_at`,
		p.ID, p.SKU, p.Name, p.Brand, p.SupplierID, p.PhotoURL, p.CostPrice, p.SellPrice,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if isDuplicateKey(err) {
			return model.NewAppError(model.ErrConflict, "sku already exists")
		}
		return err
	}
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	p := &model.Product{}
	err := r.db.QueryRow(ctx,
		`SELECT id, sku, name, brand, supplier_id, photo_url, cost_price, sell_price,
		        qty_shop, qty_warehouse, is_deleted, created_at, updated_at
		 FROM products WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&p.ID, &p.SKU, &p.Name, &p.Brand, &p.SupplierID, &p.PhotoURL,
		&p.CostPrice, &p.SellPrice, &p.QtyShop, &p.QtyWarehouse,
		&p.IsDeleted, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "product not found")
	}
	if err != nil {
		return nil, err
	}
	p.Margin = p.SellPrice.Sub(p.CostPrice)
	return p, nil
}

func (r *ProductRepository) List(ctx context.Context, filter model.ProductFilter, page, limit int) ([]model.Product, int, error) {
	where, args := buildProductFilter(filter)

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products WHERE is_deleted = false %s", where)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	args = append(args, limit, offset)
	listQuery := fmt.Sprintf(
		`SELECT id, sku, name, brand, supplier_id, photo_url, cost_price, sell_price,
		        qty_shop, qty_warehouse, is_deleted, created_at, updated_at
		 FROM products WHERE is_deleted = false %s
		 ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args),
	)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Brand, &p.SupplierID, &p.PhotoURL,
			&p.CostPrice, &p.SellPrice, &p.QtyShop, &p.QtyWarehouse,
			&p.IsDeleted, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		p.Margin = p.SellPrice.Sub(p.CostPrice)
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *ProductRepository) Update(ctx context.Context, p *model.Product) error {
	err := r.db.QueryRow(ctx,
		`UPDATE products SET sku=$1, name=$2, brand=$3, supplier_id=$4, photo_url=$5,
		        cost_price=$6, sell_price=$7, updated_at=NOW()
		 WHERE id=$8 AND is_deleted = false
		 RETURNING updated_at`,
		p.SKU, p.Name, p.Brand, p.SupplierID, p.PhotoURL, p.CostPrice, p.SellPrice, p.ID,
	).Scan(&p.UpdatedAt)
	if err != nil {
		if isDuplicateKey(err) {
			return model.NewAppError(model.ErrConflict, "sku already exists")
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return model.NewAppError(model.ErrNotFound, "product not found")
		}
		return err
	}
	return nil
}

func (r *ProductRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE products SET is_deleted = true, updated_at = NOW()
		 WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "product not found")
	}
	return nil
}

func buildProductFilter(f model.ProductFilter) (string, []any) {
	var conditions []string
	var args []any
	idx := 1

	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR sku ILIKE $%d)", idx, idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	if f.SupplierID != nil {
		conditions = append(conditions, fmt.Sprintf("supplier_id = $%d", idx))
		args = append(args, *f.SupplierID)
		idx++
	}
	if f.Brand != "" {
		conditions = append(conditions, fmt.Sprintf("brand ILIKE $%d", idx))
		args = append(args, "%"+f.Brand+"%")
		idx++
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return "AND " + strings.Join(conditions, " AND "), args
}
