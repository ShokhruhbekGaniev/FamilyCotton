package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type ShiftService struct {
	pool          *pgxpool.Pool
	shiftRepo     *repository.ShiftRepository
	safeRepo      *repository.SafeTransactionRepository
	ownerDebtRepo *repository.OwnerDebtRepository
}

func NewShiftService(
	pool *pgxpool.Pool,
	shiftRepo *repository.ShiftRepository,
	safeRepo *repository.SafeTransactionRepository,
	ownerDebtRepo *repository.OwnerDebtRepository,
) *ShiftService {
	return &ShiftService{
		pool:          pool,
		shiftRepo:     shiftRepo,
		safeRepo:      safeRepo,
		ownerDebtRepo: ownerDebtRepo,
	}
}

func (s *ShiftService) Open(ctx context.Context, userID uuid.UUID) (*model.Shift, error) {
	// Check no open shift exists.
	current, err := s.shiftRepo.GetCurrentOpen(ctx)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return nil, model.NewAppError(model.ErrValidation, "a shift is already open")
	}

	shift := &model.Shift{
		ID:       uuid.New(),
		OpenedBy: userID,
	}
	if err := s.shiftRepo.Create(ctx, shift); err != nil {
		return nil, err
	}
	return shift, nil
}

func (s *ShiftService) Close(ctx context.Context, userID uuid.UUID) (*model.Shift, error) {
	current, err := s.shiftRepo.GetCurrentOpen(ctx)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, model.NewAppError(model.ErrValidation, "no open shift to close")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Aggregate sales.
	cash, terminal, online, debt, err := s.shiftRepo.AggregateSales(ctx, tx, current.ID)
	if err != nil {
		return nil, err
	}

	current.ClosedBy = &userID
	current.TotalCash = cash
	current.TotalTerminal = terminal
	current.TotalOnline = online
	current.TotalDebtSales = debt

	if err := s.shiftRepo.CloseShift(ctx, tx, current); err != nil {
		return nil, err
	}

	// Safe transactions for cash.
	if cash.IsPositive() {
		desc := "Shift cash income"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "income", Source: "shift", BalanceType: "cash",
			Amount: cash, Description: &desc, ReferenceID: &current.ID,
		}); err != nil {
			return nil, err
		}
	}

	// Safe transactions for terminal.
	if terminal.IsPositive() {
		desc := "Shift terminal income"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "income", Source: "shift", BalanceType: "terminal",
			Amount: terminal, Description: &desc, ReferenceID: &current.ID,
		}); err != nil {
			return nil, err
		}
	}

	// Online: transfer to safe as cash + create owner debt.
	if online.IsPositive() {
		desc := "Online payments transferred as cash"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "income", Source: "online_owner_debt", BalanceType: "cash",
			Amount: online, Description: &desc, ReferenceID: &current.ID,
		}); err != nil {
			return nil, err
		}

		if err := s.ownerDebtRepo.Create(ctx, tx, &model.OwnerDebt{
			ShiftID: current.ID,
			Amount:  online,
		}); err != nil {
			return nil, err
		}
	}

	current.Status = "closed"
	return current, tx.Commit(ctx)
}

func (s *ShiftService) GetCurrent(ctx context.Context) (*model.Shift, error) {
	current, err := s.shiftRepo.GetCurrentOpen(ctx)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, model.NewAppError(model.ErrNotFound, "no open shift")
	}
	return current, nil
}

func (s *ShiftService) List(ctx context.Context, page, limit int) ([]model.Shift, int, error) {
	return s.shiftRepo.List(ctx, page, limit)
}
