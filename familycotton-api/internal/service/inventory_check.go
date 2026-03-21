package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type InventoryCheckService struct {
	pool        *pgxpool.Pool
	icRepo      *repository.InventoryCheckRepository
	productRepo *repository.ProductRepository
}

func NewInventoryCheckService(
	pool *pgxpool.Pool,
	icRepo *repository.InventoryCheckRepository,
	productRepo *repository.ProductRepository,
) *InventoryCheckService {
	return &InventoryCheckService{
		pool:        pool,
		icRepo:      icRepo,
		productRepo: productRepo,
	}
}

func (s *InventoryCheckService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateInventoryCheckRequest) (*model.InventoryCheck, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get all products to auto-generate items.
	products, _, err := s.productRepo.List(ctx, model.ProductFilter{}, 1, 10000)
	if err != nil {
		return nil, err
	}

	ic := &model.InventoryCheck{
		ID:        uuid.New(),
		Location:  req.Location,
		CheckedBy: userID,
	}

	// Create the inventory check record (no tx needed for the header).
	if err := s.icRepo.Create(ctx, ic); err != nil {
		return nil, err
	}

	// Create items in a transaction.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var items []model.InventoryCheckItem
	for _, p := range products {
		expectedQty := p.QtyShop
		if req.Location == "warehouse" {
			expectedQty = p.QtyWarehouse
		}
		item := model.InventoryCheckItem{
			ID:               uuid.New(),
			InventoryCheckID: ic.ID,
			ProductID:        p.ID,
			ExpectedQty:      expectedQty,
		}
		if err := s.icRepo.CreateItem(ctx, tx, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	ic.Items = items
	return ic, nil
}

func (s *InventoryCheckService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateInventoryCheckRequest) (*model.InventoryCheck, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Update actual qty for each provided item.
	for _, ui := range req.Items {
		if err := s.icRepo.UpdateItemActualQty(ctx, tx, ui.ItemID, ui.ActualQty); err != nil {
			return nil, err
		}
	}

	// If completing, verify all items are filled and auto-correct stock.
	if req.Status != nil && *req.Status == "completed" {
		items, err := s.icRepo.GetItemsByCheckID(ctx, id)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			if item.ActualQty == nil {
				return nil, model.NewAppError(model.ErrValidation, "Все позиции должны иметь фактическое количество перед завершением")
			}
		}

		// Auto-correct stock for each item.
		for _, item := range items {
			product, err := s.productRepo.GetByIDForUpdate(ctx, tx, item.ProductID)
			if err != nil {
				return nil, err
			}

			// Fetch the current item to get the latest actual_qty (may have been updated above).
			actualQty := *item.ActualQty

			// Update stock based on location (determined from inventory check).
			ic, err := s.icRepo.GetByID(ctx, id)
			if err != nil {
				return nil, err
			}

			newQtyShop := product.QtyShop
			newQtyWarehouse := product.QtyWarehouse
			if ic.Location == "shop" {
				newQtyShop = actualQty
			} else {
				newQtyWarehouse = actualQty
			}

			if err := s.productRepo.UpdateStock(ctx, tx, product.ID, newQtyShop, newQtyWarehouse); err != nil {
				return nil, err
			}
		}

		// Mark as completed.
		if err := s.icRepo.Complete(ctx, tx, id); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.icRepo.GetByID(ctx, id)
}

func (s *InventoryCheckService) GetByID(ctx context.Context, id uuid.UUID) (*model.InventoryCheck, error) {
	return s.icRepo.GetByID(ctx, id)
}
