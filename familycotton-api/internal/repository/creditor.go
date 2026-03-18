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

type CreditorRepository struct {
	db *pgxpool.Pool
}

func NewCreditorRepository(db *pgxpool.Pool) *CreditorRepository {
	return &CreditorRepository{db: db}
}

func (r *CreditorRepository) Create(ctx context.Context, c *model.Creditor) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO creditors (id, name, phone, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		c.ID, c.Name, c.Phone, c.Notes,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CreditorRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Creditor, error) {
	c := &model.Creditor{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, phone, notes, total_debt, is_deleted, created_at, updated_at
		 FROM creditors WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&c.ID, &c.Name, &c.Phone, &c.Notes, &c.TotalDebt, &c.IsDeleted, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "creditor not found")
	}
	return c, err
}

func (r *CreditorRepository) List(ctx context.Context, page, limit int) ([]model.Creditor, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM creditors WHERE is_deleted = false`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, name, phone, notes, total_debt, is_deleted, created_at, updated_at
		 FROM creditors WHERE is_deleted = false
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var creditors []model.Creditor
	for rows.Next() {
		var c model.Creditor
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Notes, &c.TotalDebt, &c.IsDeleted, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		creditors = append(creditors, c)
	}
	return creditors, total, rows.Err()
}

func (r *CreditorRepository) Update(ctx context.Context, c *model.Creditor) error {
	err := r.db.QueryRow(ctx,
		`UPDATE creditors SET name=$1, phone=$2, notes=$3, updated_at=NOW()
		 WHERE id=$4 AND is_deleted = false
		 RETURNING updated_at`,
		c.Name, c.Phone, c.Notes, c.ID,
	).Scan(&c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.NewAppError(model.ErrNotFound, "creditor not found")
	}
	return err
}

func (r *CreditorRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE creditors SET is_deleted = true, updated_at = NOW()
		 WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "creditor not found")
	}
	return nil
}

func (r *CreditorRepository) UpdateDebt(ctx context.Context, tx DBTX, creditorID uuid.UUID, delta decimal.Decimal) error {
	_, err := tx.Exec(ctx,
		`UPDATE creditors SET total_debt = total_debt + $1, updated_at = NOW()
		 WHERE id = $2 AND is_deleted = false`,
		delta, creditorID,
	)
	return err
}
