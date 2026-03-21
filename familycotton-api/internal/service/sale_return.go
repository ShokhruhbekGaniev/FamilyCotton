package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type SaleReturnService struct {
	pool           *pgxpool.Pool
	saleReturnRepo *repository.SaleReturnRepository
	saleRepo       *repository.SaleRepository
	productRepo    *repository.ProductRepository
	clientRepo     *repository.ClientRepository
	safeRepo       *repository.SafeTransactionRepository
}

func NewSaleReturnService(
	pool *pgxpool.Pool,
	saleReturnRepo *repository.SaleReturnRepository,
	saleRepo *repository.SaleRepository,
	productRepo *repository.ProductRepository,
	clientRepo *repository.ClientRepository,
	safeRepo *repository.SafeTransactionRepository,
) *SaleReturnService {
	return &SaleReturnService{
		pool: pool, saleReturnRepo: saleReturnRepo, saleRepo: saleRepo,
		productRepo: productRepo, clientRepo: clientRepo, safeRepo: safeRepo,
	}
}

func (s *SaleReturnService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateSaleReturnRequest) (*model.SaleReturn, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Fetch original sale and item.
	sale, err := s.saleRepo.GetByID(ctx, req.SaleID)
	if err != nil {
		return nil, err
	}
	saleItem, err := s.saleRepo.GetSaleItemByID(ctx, req.SaleItemID)
	if err != nil {
		return nil, err
	}

	// Validate return quantity.
	alreadyReturned, err := s.saleRepo.SumReturnedQty(ctx, req.SaleItemID)
	if err != nil {
		return nil, err
	}
	if req.Quantity > saleItem.Quantity-alreadyReturned {
		return nil, model.NewAppError(model.ErrValidation, "Количество возврата превышает доступное")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sr := &model.SaleReturn{
		ID:         uuid.New(),
		SaleID:     req.SaleID,
		SaleItemID: req.SaleItemID,
		Quantity:   req.Quantity,
		ReturnType: req.ReturnType,
		CreatedBy:  userID,
	}

	returnValue := saleItem.UnitPrice.Mul(decimal.NewFromInt(int64(req.Quantity)))

	switch req.ReturnType {
	case "full":
		// Return product to shop stock.
		product, err := s.productRepo.GetByIDForUpdate(ctx, tx, saleItem.ProductID)
		if err != nil {
			return nil, err
		}
		if err := s.productRepo.UpdateStock(ctx, tx, product.ID, product.QtyShop+req.Quantity, product.QtyWarehouse); err != nil {
			return nil, err
		}

		// Proportional refund from safe.
		sr.RefundAmount = returnValue
		proportion := returnValue.Div(sale.TotalAmount)

		cashRefund := sale.PaidCash.Mul(proportion)
		if cashRefund.IsPositive() {
			desc := "Client return refund (cash)"
			if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
				Type: "expense", Source: "client_refund", BalanceType: "cash",
				Amount: cashRefund, Description: &desc, ReferenceID: &sr.ID,
			}); err != nil {
				return nil, err
			}
		}

		termRefund := sale.PaidTerminal.Mul(proportion)
		if termRefund.IsPositive() {
			desc := "Client return refund (terminal)"
			if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
				Type: "expense", Source: "client_refund", BalanceType: "terminal",
				Amount: termRefund, Description: &desc, ReferenceID: &sr.ID,
			}); err != nil {
				return nil, err
			}
		}

		// If debt was part of original sale, reduce client debt.
		debtRefund := sale.PaidDebt.Mul(proportion)
		if debtRefund.IsPositive() && sale.ClientID != nil {
			if err := s.clientRepo.UpdateDebt(ctx, tx, *sale.ClientID, debtRefund.Neg()); err != nil {
				return nil, err
			}
		}

	case "exchange":
		// Return old product.
		oldProduct, err := s.productRepo.GetByIDForUpdate(ctx, tx, saleItem.ProductID)
		if err != nil {
			return nil, err
		}
		if err := s.productRepo.UpdateStock(ctx, tx, oldProduct.ID, oldProduct.QtyShop+req.Quantity, oldProduct.QtyWarehouse); err != nil {
			return nil, err
		}

		// Deduct new product.
		sr.NewProductID = req.NewProductID
		newProduct, err := s.productRepo.GetByIDForUpdate(ctx, tx, *req.NewProductID)
		if err != nil {
			return nil, err
		}
		if newProduct.QtyShop < req.Quantity {
			return nil, model.NewAppError(model.ErrValidation, "Недостаточно товара для обмена")
		}
		if err := s.productRepo.UpdateStock(ctx, tx, newProduct.ID, newProduct.QtyShop-req.Quantity, newProduct.QtyWarehouse); err != nil {
			return nil, err
		}

	case "exchange_diff":
		// Return old product.
		oldProduct, err := s.productRepo.GetByIDForUpdate(ctx, tx, saleItem.ProductID)
		if err != nil {
			return nil, err
		}
		if err := s.productRepo.UpdateStock(ctx, tx, oldProduct.ID, oldProduct.QtyShop+req.Quantity, oldProduct.QtyWarehouse); err != nil {
			return nil, err
		}

		// Deduct new product.
		sr.NewProductID = req.NewProductID
		newProduct, err := s.productRepo.GetByIDForUpdate(ctx, tx, *req.NewProductID)
		if err != nil {
			return nil, err
		}
		if newProduct.QtyShop < req.Quantity {
			return nil, model.NewAppError(model.ErrValidation, "Недостаточно товара для обмена")
		}
		if err := s.productRepo.UpdateStock(ctx, tx, newProduct.ID, newProduct.QtyShop-req.Quantity, newProduct.QtyWarehouse); err != nil {
			return nil, err
		}

		// Calculate price difference.
		oldValue := saleItem.UnitPrice.Mul(decimal.NewFromInt(int64(req.Quantity)))
		newValue := newProduct.SellPrice.Mul(decimal.NewFromInt(int64(req.Quantity)))
		diff := newValue.Sub(oldValue)

		if diff.IsNegative() {
			// New is cheaper — refund difference.
			sr.RefundAmount = diff.Abs()
			desc := "Exchange diff refund"
			if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
				Type: "expense", Source: "client_refund", BalanceType: "cash",
				Amount: diff.Abs(), Description: &desc, ReferenceID: &sr.ID,
			}); err != nil {
				return nil, err
			}
		} else if diff.IsPositive() {
			// New is more expensive — surcharge.
			sr.SurchargeAmount = diff
			desc := "Exchange diff surcharge"
			if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
				Type: "income", Source: "client_payment", BalanceType: "cash",
				Amount: diff, Description: &desc, ReferenceID: &sr.ID,
			}); err != nil {
				return nil, err
			}
		}
	}

	if err := s.saleReturnRepo.Create(ctx, tx, sr); err != nil {
		return nil, err
	}

	return sr, tx.Commit(ctx)
}

func (s *SaleReturnService) List(ctx context.Context, saleID *uuid.UUID, page, limit int) ([]model.SaleReturn, int, error) {
	return s.saleReturnRepo.List(ctx, saleID, page, limit)
}
