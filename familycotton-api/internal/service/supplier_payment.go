package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type SupplierPaymentService struct {
	pool         *pgxpool.Pool
	spRepo       *repository.SupplierPaymentRepository
	poRepo       *repository.PurchaseOrderRepository
	productRepo  *repository.ProductRepository
	supplierRepo *repository.SupplierRepository
	safeRepo     *repository.SafeTransactionRepository
}

func NewSupplierPaymentService(
	pool *pgxpool.Pool,
	spRepo *repository.SupplierPaymentRepository,
	poRepo *repository.PurchaseOrderRepository,
	productRepo *repository.ProductRepository,
	supplierRepo *repository.SupplierRepository,
	safeRepo *repository.SafeTransactionRepository,
) *SupplierPaymentService {
	return &SupplierPaymentService{
		pool:         pool,
		spRepo:       spRepo,
		poRepo:       poRepo,
		productRepo:  productRepo,
		supplierRepo: supplierRepo,
		safeRepo:     safeRepo,
	}
}

func (s *SupplierPaymentService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateSupplierPaymentRequest) (*model.SupplierPayment, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sp := &model.SupplierPayment{
		ID:                uuid.New(),
		SupplierID:        req.SupplierID,
		PurchaseOrderID:   req.PurchaseOrderID,
		PaymentType:       req.PaymentType,
		Amount:            req.Amount,
		ReturnedProductID: req.ReturnedProductID,
		ReturnedQty:       req.ReturnedQty,
		CreatedBy:         userID,
	}

	switch req.PaymentType {
	case "money":
		// Deduct from safe (cash expense).
		desc := "Supplier payment (money)"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type:        "expense",
			Source:      "supplier_payment",
			BalanceType: "cash",
			Amount:      req.Amount,
			Description: &desc,
			ReferenceID: &sp.ID,
		}); err != nil {
			return nil, err
		}

		// Reduce supplier debt.
		if err := s.supplierRepo.UpdateDebt(ctx, tx, req.SupplierID, req.Amount.Neg()); err != nil {
			return nil, err
		}

		// Update purchase order paid amount and recalc status.
		if req.PurchaseOrderID != nil {
			if err := s.poRepo.UpdatePaidAmount(ctx, tx, *req.PurchaseOrderID, req.Amount); err != nil {
				return nil, err
			}
			if err := s.poRepo.RecalcStatus(ctx, tx, *req.PurchaseOrderID); err != nil {
				return nil, err
			}
		}

	case "product_return":
		// Lock and deduct stock of returned product.
		product, err := s.productRepo.GetByIDForUpdate(ctx, tx, *req.ReturnedProductID)
		if err != nil {
			return nil, err
		}
		qty := *req.ReturnedQty
		if product.QtyShop < qty {
			return nil, model.NewAppError(model.ErrValidation, "insufficient shop stock for product return")
		}
		if err := s.productRepo.UpdateStock(ctx, tx, product.ID, product.QtyShop-qty, product.QtyWarehouse); err != nil {
			return nil, err
		}

		// Return value = cost_price * qty.
		returnValue := product.CostPrice.Mul(decimal.NewFromInt(int64(qty)))
		sp.Amount = returnValue

		// Reduce supplier debt by return value.
		if err := s.supplierRepo.UpdateDebt(ctx, tx, req.SupplierID, returnValue.Neg()); err != nil {
			return nil, err
		}

		// Update purchase order paid amount and recalc status.
		if req.PurchaseOrderID != nil {
			if err := s.poRepo.UpdatePaidAmount(ctx, tx, *req.PurchaseOrderID, returnValue); err != nil {
				return nil, err
			}
			if err := s.poRepo.RecalcStatus(ctx, tx, *req.PurchaseOrderID); err != nil {
				return nil, err
			}
		}
	}

	if err := s.spRepo.Create(ctx, tx, sp); err != nil {
		return nil, err
	}

	return sp, tx.Commit(ctx)
}
