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

type ClientRepository struct {
	db *pgxpool.Pool
}

func NewClientRepository(db *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{db: db}
}

func (r *ClientRepository) Create(ctx context.Context, c *model.Client) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO clients (id, name, phone)
		 VALUES ($1, $2, $3)
		 RETURNING created_at, updated_at`,
		c.ID, c.Name, c.Phone,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *ClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Client, error) {
	c := &model.Client{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, phone, total_debt, is_deleted, created_at, updated_at
		 FROM clients WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&c.ID, &c.Name, &c.Phone, &c.TotalDebt, &c.IsDeleted, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "client not found")
	}
	return c, err
}

func (r *ClientRepository) List(ctx context.Context, page, limit int) ([]model.Client, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM clients WHERE is_deleted = false`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, name, phone, total_debt, is_deleted, created_at, updated_at
		 FROM clients WHERE is_deleted = false
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var clients []model.Client
	for rows.Next() {
		var c model.Client
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.TotalDebt, &c.IsDeleted, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		clients = append(clients, c)
	}
	return clients, total, rows.Err()
}

func (r *ClientRepository) Update(ctx context.Context, c *model.Client) error {
	err := r.db.QueryRow(ctx,
		`UPDATE clients SET name=$1, phone=$2, updated_at=NOW()
		 WHERE id=$3 AND is_deleted = false
		 RETURNING updated_at`,
		c.Name, c.Phone, c.ID,
	).Scan(&c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.NewAppError(model.ErrNotFound, "client not found")
	}
	return err
}

// UpdateDebt adjusts client total_debt within a transaction.
func (r *ClientRepository) UpdateDebt(ctx context.Context, tx DBTX, clientID uuid.UUID, delta decimal.Decimal) error {
	_, err := tx.Exec(ctx,
		`UPDATE clients SET total_debt = total_debt + $1, updated_at = NOW()
		 WHERE id = $2 AND is_deleted = false`,
		delta, clientID,
	)
	return err
}

func (r *ClientRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE clients SET is_deleted = true, updated_at = NOW()
		 WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "client not found")
	}
	return nil
}
