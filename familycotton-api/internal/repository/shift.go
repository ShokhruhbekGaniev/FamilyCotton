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

type ShiftRepository struct {
	db *pgxpool.Pool
}

func NewShiftRepository(db *pgxpool.Pool) *ShiftRepository {
	return &ShiftRepository{db: db}
}

func (r *ShiftRepository) Create(ctx context.Context, s *model.Shift) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO shifts (id, opened_by) VALUES ($1, $2)
		 RETURNING opened_at, status`,
		s.ID, s.OpenedBy,
	).Scan(&s.OpenedAt, &s.Status)
}

func (r *ShiftRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Shift, error) {
	s := &model.Shift{}
	err := r.db.QueryRow(ctx,
		`SELECT id, opened_by, closed_by, opened_at, closed_at,
		        total_cash, total_terminal, total_online, total_debt_sales, status
		 FROM shifts WHERE id = $1`, id,
	).Scan(&s.ID, &s.OpenedBy, &s.ClosedBy, &s.OpenedAt, &s.ClosedAt,
		&s.TotalCash, &s.TotalTerminal, &s.TotalOnline, &s.TotalDebtSales, &s.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "shift not found")
	}
	return s, err
}

func (r *ShiftRepository) GetCurrentOpen(ctx context.Context) (*model.Shift, error) {
	s := &model.Shift{}
	err := r.db.QueryRow(ctx,
		`SELECT id, opened_by, closed_by, opened_at, closed_at,
		        total_cash, total_terminal, total_online, total_debt_sales, status
		 FROM shifts WHERE status = 'open' ORDER BY opened_at DESC LIMIT 1`,
	).Scan(&s.ID, &s.OpenedBy, &s.ClosedBy, &s.OpenedAt, &s.ClosedAt,
		&s.TotalCash, &s.TotalTerminal, &s.TotalOnline, &s.TotalDebtSales, &s.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *ShiftRepository) CloseShift(ctx context.Context, tx DBTX, s *model.Shift) error {
	return tx.QueryRow(ctx,
		`UPDATE shifts SET closed_by=$1, closed_at=NOW(), total_cash=$2, total_terminal=$3,
		        total_online=$4, total_debt_sales=$5, status='closed'
		 WHERE id=$6 AND status='open'
		 RETURNING closed_at`,
		s.ClosedBy, s.TotalCash, s.TotalTerminal, s.TotalOnline, s.TotalDebtSales, s.ID,
	).Scan(&s.ClosedAt)
}

func (r *ShiftRepository) List(ctx context.Context, page, limit int) ([]model.Shift, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM shifts`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, opened_by, closed_by, opened_at, closed_at,
		        total_cash, total_terminal, total_online, total_debt_sales, status
		 FROM shifts ORDER BY opened_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shifts []model.Shift
	for rows.Next() {
		var s model.Shift
		if err := rows.Scan(&s.ID, &s.OpenedBy, &s.ClosedBy, &s.OpenedAt, &s.ClosedAt,
			&s.TotalCash, &s.TotalTerminal, &s.TotalOnline, &s.TotalDebtSales, &s.Status); err != nil {
			return nil, 0, err
		}
		shifts = append(shifts, s)
	}
	return shifts, total, rows.Err()
}

// AggregateSales sums payment fields for all sales in a shift.
func (r *ShiftRepository) AggregateSales(ctx context.Context, tx DBTX, shiftID uuid.UUID) (cash, terminal, online, debt decimal.Decimal, err error) {
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(paid_cash),0), COALESCE(SUM(paid_terminal),0),
		        COALESCE(SUM(paid_online),0), COALESCE(SUM(paid_debt),0)
		 FROM sales WHERE shift_id = $1`, shiftID,
	).Scan(&cash, &terminal, &online, &debt)
	return
}
