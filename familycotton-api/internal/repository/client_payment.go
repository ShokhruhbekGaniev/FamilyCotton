package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type ClientPaymentRepository struct {
	db *pgxpool.Pool
}

func NewClientPaymentRepository(db *pgxpool.Pool) *ClientPaymentRepository {
	return &ClientPaymentRepository{db: db}
}

func (r *ClientPaymentRepository) Create(ctx context.Context, tx DBTX, cp *model.ClientPayment) error {
	return tx.QueryRow(ctx,
		`INSERT INTO client_payments (id, client_id, amount, payment_method, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		cp.ID, cp.ClientID, cp.Amount, cp.PaymentMethod, cp.CreatedBy,
	).Scan(&cp.CreatedAt)
}

func (r *ClientPaymentRepository) List(ctx context.Context, clientID uuid.UUID, page, limit int) ([]model.ClientPayment, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM client_payments WHERE client_id = $1`, clientID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, client_id, amount, payment_method, created_by, created_at
		 FROM client_payments WHERE client_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		clientID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []model.ClientPayment
	for rows.Next() {
		var cp model.ClientPayment
		if err := rows.Scan(&cp.ID, &cp.ClientID, &cp.Amount, &cp.PaymentMethod, &cp.CreatedBy, &cp.CreatedAt); err != nil {
			return nil, 0, err
		}
		payments = append(payments, cp)
	}
	return payments, total, rows.Err()
}
