package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type BrandRepository struct {
	db *pgxpool.Pool
}

func NewBrandRepository(db *pgxpool.Pool) *BrandRepository {
	return &BrandRepository{db: db}
}

func (r *BrandRepository) Create(ctx context.Context, b *model.Brand) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO brands (id, name)
		 VALUES ($1, $2)
		 RETURNING created_at, updated_at`,
		b.ID, b.Name,
	).Scan(&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if isDuplicateKey(err) {
			return model.NewAppError(model.ErrConflict, "Бренд с таким названием уже существует")
		}
		return err
	}
	return nil
}

func (r *BrandRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Brand, error) {
	b := &model.Brand{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, is_deleted, created_at, updated_at
		 FROM brands WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&b.ID, &b.Name, &b.IsDeleted, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "Бренд не найден")
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *BrandRepository) List(ctx context.Context, page, limit int) ([]model.Brand, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM brands WHERE is_deleted = false`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, name, is_deleted, created_at, updated_at
		 FROM brands WHERE is_deleted = false
		 ORDER BY name ASC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var brands []model.Brand
	for rows.Next() {
		var b model.Brand
		if err := rows.Scan(&b.ID, &b.Name, &b.IsDeleted, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, err
		}
		brands = append(brands, b)
	}
	return brands, total, rows.Err()
}

func (r *BrandRepository) Update(ctx context.Context, b *model.Brand) error {
	err := r.db.QueryRow(ctx,
		`UPDATE brands SET name=$1, updated_at=NOW()
		 WHERE id=$2 AND is_deleted = false
		 RETURNING updated_at`,
		b.Name, b.ID,
	).Scan(&b.UpdatedAt)
	if err != nil {
		if isDuplicateKey(err) {
			return model.NewAppError(model.ErrConflict, "Бренд с таким названием уже существует")
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return model.NewAppError(model.ErrNotFound, "Бренд не найден")
		}
		return err
	}
	return nil
}

func (r *BrandRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE brands SET is_deleted = true, updated_at = NOW()
		 WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "Бренд не найден")
	}
	return nil
}
