package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type StockTransferService struct {
	pool        *pgxpool.Pool
	stRepo      *repository.StockTransferRepository
	productRepo *repository.ProductRepository
}

func NewStockTransferService(
	pool *pgxpool.Pool,
	stRepo *repository.StockTransferRepository,
	productRepo *repository.ProductRepository,
) *StockTransferService {
	return &StockTransferService{
		pool:        pool,
		stRepo:      stRepo,
		productRepo: productRepo,
	}
}

func (s *StockTransferService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateStockTransferRequest) (*model.StockTransfer, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Lock the product row.
	product, err := s.productRepo.GetByIDForUpdate(ctx, tx, req.ProductID)
	if err != nil {
		return nil, err
	}

	newQtyShop := product.QtyShop
	newQtyWarehouse := product.QtyWarehouse

	switch req.Direction {
	case "warehouse_to_shop":
		if product.QtyWarehouse < req.Quantity {
			return nil, model.NewAppError(model.ErrValidation, "Недостаточно товара на складе")
		}
		newQtyShop += req.Quantity
		newQtyWarehouse -= req.Quantity
	case "shop_to_warehouse":
		if product.QtyShop < req.Quantity {
			return nil, model.NewAppError(model.ErrValidation, "Недостаточно товара в магазине")
		}
		newQtyShop -= req.Quantity
		newQtyWarehouse += req.Quantity
	}

	if err := s.productRepo.UpdateStock(ctx, tx, product.ID, newQtyShop, newQtyWarehouse); err != nil {
		return nil, err
	}

	st := &model.StockTransfer{
		ID:        uuid.New(),
		ProductID: req.ProductID,
		Direction: req.Direction,
		Quantity:  req.Quantity,
		CreatedBy: userID,
	}

	if err := s.stRepo.Create(ctx, tx, st); err != nil {
		return nil, err
	}

	return st, tx.Commit(ctx)
}
