package service

import (
	"context"
	"time"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type DashboardService struct {
	repo *repository.DashboardRepository
}

func NewDashboardService(repo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

func (s *DashboardService) Revenue(ctx context.Context, from, to time.Time) (*model.RevenueReport, error) {
	return s.repo.Revenue(ctx, from, to)
}

func (s *DashboardService) Profit(ctx context.Context, from, to time.Time) (*model.ProfitReport, error) {
	return s.repo.Profit(ctx, from, to)
}

func (s *DashboardService) StockValue(ctx context.Context) (*model.StockValueReport, error) {
	return s.repo.StockValue(ctx)
}

func (s *DashboardService) SalesBySupplier(ctx context.Context, from, to time.Time) ([]model.SupplierSalesReport, error) {
	return s.repo.SalesBySupplier(ctx, from, to)
}

func (s *DashboardService) PaidVsDebt(ctx context.Context) (*model.PaidVsDebtReport, error) {
	return s.repo.PaidVsDebt(ctx)
}
