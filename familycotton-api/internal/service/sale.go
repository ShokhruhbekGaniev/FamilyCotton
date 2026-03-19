package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type SaleService struct {
	pool        *pgxpool.Pool
	saleRepo    *repository.SaleRepository
	shiftRepo   *repository.ShiftRepository
	productRepo *repository.ProductRepository
	clientRepo  *repository.ClientRepository
}

func NewSaleService(
	pool *pgxpool.Pool,
	saleRepo *repository.SaleRepository,
	shiftRepo *repository.ShiftRepository,
	productRepo *repository.ProductRepository,
	clientRepo *repository.ClientRepository,
) *SaleService {
	return &SaleService{
		pool:        pool,
		saleRepo:    saleRepo,
		shiftRepo:   shiftRepo,
		productRepo: productRepo,
		clientRepo:  clientRepo,
	}
}

func (s *SaleService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateSaleRequest) (*model.Sale, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify shift is open.
	shift, err := s.shiftRepo.GetCurrentOpen(ctx)
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, model.NewAppError(model.ErrValidation, "no open shift")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Build items, compute total, lock and deduct stock.
	var items []model.SaleItem
	totalAmount := decimal.Zero

	for _, ri := range req.Items {
		product, err := s.productRepo.GetByIDForUpdate(ctx, tx, ri.ProductID)
		if err != nil {
			return nil, err
		}
		if product.QtyShop < ri.Quantity {
			return nil, model.NewAppError(model.ErrValidation, "insufficient stock for product "+product.Name)
		}

		subtotal := ri.UnitPrice.Mul(decimal.NewFromInt(int64(ri.Quantity)))
		item := model.SaleItem{
			ID:        uuid.New(),
			ProductID: ri.ProductID,
			Quantity:  ri.Quantity,
			UnitPrice: ri.UnitPrice,
			Subtotal:  subtotal,
		}
		items = append(items, item)
		totalAmount = totalAmount.Add(subtotal)

		// Deduct stock.
		if err := s.productRepo.UpdateStock(ctx, tx, product.ID, product.QtyShop-ri.Quantity, product.QtyWarehouse); err != nil {
			return nil, err
		}
	}

	// Validate payment split.
	paidTotal := req.PaidCash.Add(req.PaidTerminal).Add(req.PaidOnline).Add(req.PaidDebt)
	if !paidTotal.Equal(totalAmount) {
		return nil, model.NewAppError(model.ErrValidation, "payment amounts must equal total_amount")
	}

	// Create sale.
	sale := &model.Sale{
		ID:           uuid.New(),
		ShiftID:      shift.ID,
		ClientID:     req.ClientID,
		TotalAmount:  totalAmount,
		PaidCash:     req.PaidCash,
		PaidTerminal: req.PaidTerminal,
		PaidOnline:   req.PaidOnline,
		PaidDebt:     req.PaidDebt,
		CreatedBy:    userID,
	}

	if err := s.saleRepo.CreateSale(ctx, tx, sale); err != nil {
		return nil, err
	}

	// Create sale items.
	for i := range items {
		items[i].SaleID = sale.ID
		if err := s.saleRepo.CreateSaleItem(ctx, tx, &items[i]); err != nil {
			return nil, err
		}
	}

	// Update client debt if paid_debt > 0.
	if req.PaidDebt.IsPositive() && req.ClientID != nil {
		if err := s.clientRepo.UpdateDebt(ctx, tx, *req.ClientID, req.PaidDebt); err != nil {
			return nil, err
		}
	}

	sale.Items = items
	return sale, tx.Commit(ctx)
}

func (s *SaleService) GetByID(ctx context.Context, id uuid.UUID) (*model.Sale, error) {
	return s.saleRepo.GetByID(ctx, id)
}

func (s *SaleService) List(ctx context.Context, shiftID, clientID *uuid.UUID, page, limit int) ([]model.Sale, int, error) {
	return s.saleRepo.List(ctx, shiftID, clientID, page, limit)
}

func (s *SaleService) Delete(ctx context.Context, id uuid.UUID) error {
	// 1. Fetch the sale.
	sale, err := s.saleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 2. Check that the shift is still open.
	shift, err := s.shiftRepo.GetByID(ctx, sale.ShiftID)
	if err != nil {
		return err
	}
	if shift.Status != "open" {
		return model.NewAppError(model.ErrForbidden, "cannot delete sale in a closed shift")
	}

	// 3. Check that the sale has no returns.
	hasReturns, err := s.saleRepo.HasReturns(ctx, id)
	if err != nil {
		return err
	}
	if hasReturns {
		return model.NewAppError(model.ErrConflict, "cannot delete sale with returns")
	}

	// 4. Begin transaction.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 5. Restore stock for each item.
	items, err := s.saleRepo.GetSaleItemsBySaleID(ctx, tx, id)
	if err != nil {
		return err
	}
	for _, item := range items {
		product, err := s.productRepo.GetByIDForUpdate(ctx, tx, item.ProductID)
		if err != nil {
			return err
		}
		if err := s.productRepo.UpdateStock(ctx, tx, product.ID, product.QtyShop+item.Quantity, product.QtyWarehouse); err != nil {
			return err
		}
	}

	// 6. Reduce client debt if there was debt payment.
	if sale.PaidDebt.IsPositive() && sale.ClientID != nil {
		if err := s.clientRepo.UpdateDebt(ctx, tx, *sale.ClientID, sale.PaidDebt.Neg()); err != nil {
			return err
		}
	}

	// 7. Delete sale items, then sale.
	if err := s.saleRepo.DeleteSaleItems(ctx, tx, id); err != nil {
		return err
	}
	if err := s.saleRepo.DeleteSale(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
