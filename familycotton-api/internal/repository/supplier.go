package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/familycotton/api/internal/model"
)

type SupplierRepository struct {
	db *pgxpool.Pool
}

func NewSupplierRepository(db *pgxpool.Pool) *SupplierRepository {
	return &SupplierRepository{db: db}
}

func (r *SupplierRepository) Create(ctx context.Context, s *model.Supplier) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO suppliers (id, name, phone, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		s.ID, s.Name, s.Phone, s.Notes,
	).Scan(&s.CreatedAt, &s.UpdatedAt)
}

func (r *SupplierRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Supplier, error) {
	s := &model.Supplier{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, phone, notes, total_debt, is_deleted, created_at, updated_at
		 FROM suppliers WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&s.ID, &s.Name, &s.Phone, &s.Notes, &s.TotalDebt, &s.IsDeleted, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "Поставщик не найден")
	}
	return s, err
}

func (r *SupplierRepository) List(ctx context.Context, page, limit int) ([]model.Supplier, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM suppliers WHERE is_deleted = false`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, name, phone, notes, total_debt, is_deleted, created_at, updated_at
		 FROM suppliers WHERE is_deleted = false
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var suppliers []model.Supplier
	for rows.Next() {
		var s model.Supplier
		if err := rows.Scan(&s.ID, &s.Name, &s.Phone, &s.Notes, &s.TotalDebt, &s.IsDeleted, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, total, rows.Err()
}

func (r *SupplierRepository) Update(ctx context.Context, s *model.Supplier) error {
	err := r.db.QueryRow(ctx,
		`UPDATE suppliers SET name=$1, phone=$2, notes=$3, updated_at=NOW()
		 WHERE id=$4 AND is_deleted = false
		 RETURNING updated_at`,
		s.Name, s.Phone, s.Notes, s.ID,
	).Scan(&s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.NewAppError(model.ErrNotFound, "Поставщик не найден")
	}
	return err
}

func (r *SupplierRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE suppliers SET is_deleted = true, updated_at = NOW()
		 WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "Поставщик не найден")
	}
	return nil
}

func (r *SupplierRepository) UpdateDebt(ctx context.Context, tx DBTX, supplierID uuid.UUID, delta decimal.Decimal) error {
	_, err := tx.Exec(ctx,
		`UPDATE suppliers SET total_debt = total_debt + $1, updated_at = NOW()
		 WHERE id = $2 AND is_deleted = false`,
		delta, supplierID,
	)
	return err
}
