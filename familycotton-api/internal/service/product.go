package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type ProductService struct {
	repo *repository.ProductRepository
}

func NewProductService(repo *repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Create(ctx context.Context, req *model.CreateProductRequest) (*model.Product, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	product := &model.Product{
		ID:         uuid.New(),
		SKU:        req.SKU,
		Name:       req.Name,
		Brand:      req.Brand,
		SupplierID: req.SupplierID,
		PhotoURL:   req.PhotoURL,
		CostPrice:  req.CostPrice,
		SellPrice:  req.SellPrice,
	}
	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}
	product.Margin = product.SellPrice.Sub(product.CostPrice)
	return product, nil
}

func (s *ProductService) GetByID(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) List(ctx context.Context, filter model.ProductFilter, page, limit int) ([]model.Product, int, error) {
	return s.repo.List(ctx, filter, page, limit)
}

func (s *ProductService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateProductRequest) (*model.Product, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.SKU != nil {
		product.SKU = *req.SKU
	}
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Brand != nil {
		product.Brand = req.Brand
	}
	if req.SupplierID != nil {
		product.SupplierID = req.SupplierID
	}
	if req.PhotoURL != nil {
		product.PhotoURL = req.PhotoURL
	}
	if req.CostPrice != nil {
		product.CostPrice = *req.CostPrice
	}
	if req.SellPrice != nil {
		product.SellPrice = *req.SellPrice
	}
	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}
	product.Margin = product.SellPrice.Sub(product.CostPrice)
	return product, nil
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
