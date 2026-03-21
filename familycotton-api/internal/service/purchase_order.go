package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type PurchaseOrderService struct {
	pool         *pgxpool.Pool
	poRepo       *repository.PurchaseOrderRepository
	productRepo  *repository.ProductRepository
	supplierRepo *repository.SupplierRepository
	safeRepo     *repository.SafeTransactionRepository
}

func NewPurchaseOrderService(
	pool *pgxpool.Pool,
	poRepo *repository.PurchaseOrderRepository,
	productRepo *repository.ProductRepository,
	supplierRepo *repository.SupplierRepository,
	safeRepo *repository.SafeTransactionRepository,
) *PurchaseOrderService {
	return &PurchaseOrderService{
		pool:         pool,
		poRepo:       poRepo,
		productRepo:  productRepo,
		supplierRepo: supplierRepo,
		safeRepo:     safeRepo,
	}
}

func (s *PurchaseOrderService) Create(ctx context.Context, userID uuid.UUID, req *model.CreatePurchaseOrderRequest) (*model.PurchaseOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Compute total and build items, arriving stock per destination.
	totalAmount := decimal.Zero
	var items []model.PurchaseOrderItem

	for _, ri := range req.Items {
		product, err := s.productRepo.GetByIDForUpdate(ctx, tx, ri.ProductID)
		if err != nil {
			return nil, err
		}

		subtotal := ri.UnitCost.Mul(decimal.NewFromInt(int64(ri.Quantity)))
		totalAmount = totalAmount.Add(subtotal)

		item := model.PurchaseOrderItem{
			ID:          uuid.New(),
			ProductID:   ri.ProductID,
			Quantity:    ri.Quantity,
			UnitCost:    ri.UnitCost,
			Destination: ri.Destination,
		}
		items = append(items, item)

		// Add arriving stock to correct location.
		newQtyShop := product.QtyShop
		newQtyWarehouse := product.QtyWarehouse
		switch ri.Destination {
		case "shop":
			newQtyShop += ri.Quantity
		case "warehouse":
			newQtyWarehouse += ri.Quantity
		}
		if err := s.productRepo.UpdateStock(ctx, tx, product.ID, newQtyShop, newQtyWarehouse); err != nil {
			return nil, err
		}
	}

	// Validate paid amount does not exceed total.
	if req.PaidAmount.GreaterThan(totalAmount) {
		return nil, model.NewAppError(model.ErrValidation, "Сумма оплаты не может превышать общую сумму заказа")
	}

	po := &model.PurchaseOrder{
		ID:          uuid.New(),
		SupplierID:  req.SupplierID,
		TotalAmount: totalAmount,
		PaidAmount:  req.PaidAmount,
		Status:      "unpaid",
		CreatedBy:   userID,
	}
	if req.PaidAmount.Equal(totalAmount) {
		po.Status = "paid"
	} else if req.PaidAmount.IsPositive() {
		po.Status = "partial"
	}

	if err := s.poRepo.Create(ctx, tx, po); err != nil {
		return nil, err
	}

	for i := range items {
		items[i].PurchaseOrderID = po.ID
		if err := s.poRepo.CreateItem(ctx, tx, &items[i]); err != nil {
			return nil, err
		}
	}

	// Record safe transaction for paid amount.
	if req.PaidAmount.IsPositive() {
		desc := "Purchase order payment"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type:        "expense",
			Source:      "supplier_payment",
			BalanceType: "cash",
			Amount:      req.PaidAmount,
			Description: &desc,
			ReferenceID: &po.ID,
		}); err != nil {
			return nil, err
		}
	}

	// Record supplier debt for remaining amount.
	remainder := totalAmount.Sub(req.PaidAmount)
	if remainder.IsPositive() {
		if err := s.supplierRepo.UpdateDebt(ctx, tx, req.SupplierID, remainder); err != nil {
			return nil, err
		}
	}

	po.Items = items
	return po, tx.Commit(ctx)
}

func (s *PurchaseOrderService) GetByID(ctx context.Context, id uuid.UUID) (*model.PurchaseOrder, error) {
	return s.poRepo.GetByID(ctx, id)
}

func (s *PurchaseOrderService) List(ctx context.Context, supplierID *uuid.UUID, status string, page, limit int) ([]model.PurchaseOrder, int, error) {
	return s.poRepo.List(ctx, supplierID, status, page, limit)
}
