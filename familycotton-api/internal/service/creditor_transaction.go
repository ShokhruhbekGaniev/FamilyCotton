package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type CreditorTransactionService struct {
	pool          *pgxpool.Pool
	ctRepo        *repository.CreditorTransactionRepository
	creditorRepo  *repository.CreditorRepository
	safeRepo      *repository.SafeTransactionRepository
}

func NewCreditorTransactionService(
	pool *pgxpool.Pool,
	ctRepo *repository.CreditorTransactionRepository,
	creditorRepo *repository.CreditorRepository,
	safeRepo *repository.SafeTransactionRepository,
) *CreditorTransactionService {
	return &CreditorTransactionService{
		pool:         pool,
		ctRepo:       ctRepo,
		creditorRepo: creditorRepo,
		safeRepo:     safeRepo,
	}
}

func (s *CreditorTransactionService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateCreditorTransactionRequest) (*model.CreditorTransaction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Compute amount_uzs = amount * exchange_rate.
	amountUZS := req.Amount.Mul(req.ExchangeRate)

	ct := &model.CreditorTransaction{
		ID:           uuid.New(),
		CreditorID:   req.CreditorID,
		Type:         req.Type,
		Currency:     req.Currency,
		Amount:       req.Amount,
		ExchangeRate: req.ExchangeRate,
		AmountUZS:    amountUZS,
		CreatedBy:    userID,
	}

	switch req.Type {
	case "receive":
		// Creditor lends money to the business — safe income, increment creditor debt.
		desc := "Creditor loan received"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type:        "income",
			Source:      "creditor_loan",
			BalanceType: "cash",
			Amount:      amountUZS,
			Description: &desc,
			ReferenceID: &ct.ID,
		}); err != nil {
			return nil, err
		}
		if err := s.creditorRepo.UpdateDebt(ctx, tx, req.CreditorID, amountUZS); err != nil {
			return nil, err
		}

	case "repay":
		// Business repays creditor — safe expense, reduce creditor debt.
		desc := "Creditor loan repayment"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type:        "expense",
			Source:      "creditor_repay",
			BalanceType: "cash",
			Amount:      amountUZS,
			Description: &desc,
			ReferenceID: &ct.ID,
		}); err != nil {
			return nil, err
		}
		if err := s.creditorRepo.UpdateDebt(ctx, tx, req.CreditorID, amountUZS.Neg()); err != nil {
			return nil, err
		}
	}

	if err := s.ctRepo.Create(ctx, tx, ct); err != nil {
		return nil, err
	}

	return ct, tx.Commit(ctx)
}

func (s *CreditorTransactionService) ListByCreditor(ctx context.Context, creditorID uuid.UUID, page, limit int) ([]model.CreditorTransaction, int, error) {
	return s.ctRepo.ListByCreditor(ctx, creditorID, page, limit)
}
