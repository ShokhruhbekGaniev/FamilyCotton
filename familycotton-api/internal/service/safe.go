package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type SafeService struct {
	pool          *pgxpool.Pool
	safeRepo      *repository.SafeTransactionRepository
	ownerDebtRepo *repository.OwnerDebtRepository
}

func NewSafeService(pool *pgxpool.Pool, safeRepo *repository.SafeTransactionRepository, ownerDebtRepo *repository.OwnerDebtRepository) *SafeService {
	return &SafeService{pool: pool, safeRepo: safeRepo, ownerDebtRepo: ownerDebtRepo}
}

func (s *SafeService) GetBalance(ctx context.Context) (*model.SafeBalance, error) {
	return s.safeRepo.GetBalance(ctx)
}

func (s *SafeService) ListTransactions(ctx context.Context, page, limit int) ([]model.SafeTransaction, int, error) {
	return s.safeRepo.List(ctx, page, limit)
}

func (s *SafeService) ListOwnerDebts(ctx context.Context) ([]model.OwnerDebt, error) {
	return s.ownerDebtRepo.ListUnsettled(ctx)
}

type OwnerDepositRequest struct {
	Amount decimal.Decimal `json:"amount"`
}

func (s *SafeService) OwnerDeposit(ctx context.Context, req *OwnerDepositRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return model.NewAppError(model.ErrValidation, "Сумма должна быть положительной")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Record safe expense (owner takes cash from safe).
	desc := "Owner deposit — settling online debt"
	refID := uuid.New()
	if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
		Type: "expense", Source: "owner_deposit", BalanceType: "cash",
		Amount: req.Amount, Description: &desc, ReferenceID: &refID,
	}); err != nil {
		return err
	}

	// Settle owner debts up to amount.
	remaining := req.Amount
	debts, err := s.ownerDebtRepo.ListUnsettled(ctx)
	if err != nil {
		return err
	}
	for _, d := range debts {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		if d.Amount.LessThanOrEqual(remaining) {
			if err := s.ownerDebtRepo.Settle(ctx, tx, d.ID); err != nil {
				return err
			}
			remaining = remaining.Sub(d.Amount)
		}
	}

	return tx.Commit(ctx)
}
