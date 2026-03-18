package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type ClientPaymentService struct {
	pool        *pgxpool.Pool
	paymentRepo *repository.ClientPaymentRepository
	clientRepo  *repository.ClientRepository
	safeRepo    *repository.SafeTransactionRepository
}

func NewClientPaymentService(
	pool *pgxpool.Pool,
	paymentRepo *repository.ClientPaymentRepository,
	clientRepo *repository.ClientRepository,
	safeRepo *repository.SafeTransactionRepository,
) *ClientPaymentService {
	return &ClientPaymentService{
		pool: pool, paymentRepo: paymentRepo, clientRepo: clientRepo, safeRepo: safeRepo,
	}
}

func (s *ClientPaymentService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateClientPaymentRequest) (*model.ClientPayment, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify client exists.
	if _, err := s.clientRepo.GetByID(ctx, req.ClientID); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	cp := &model.ClientPayment{
		ID:            uuid.New(),
		ClientID:      req.ClientID,
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		CreatedBy:     userID,
	}

	if err := s.paymentRepo.Create(ctx, tx, cp); err != nil {
		return nil, err
	}

	// Reduce client debt.
	if err := s.clientRepo.UpdateDebt(ctx, tx, req.ClientID, req.Amount.Neg()); err != nil {
		return nil, err
	}

	// Income to safe.
	desc := "Client debt payment"
	if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
		Type: "income", Source: "client_payment", BalanceType: req.PaymentMethod,
		Amount: req.Amount, Description: &desc, ReferenceID: &cp.ID,
	}); err != nil {
		return nil, err
	}

	return cp, tx.Commit(ctx)
}
