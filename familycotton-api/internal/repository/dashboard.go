package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type DashboardRepository struct {
	db *pgxpool.Pool
}

func NewDashboardRepository(db *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{db: db}
}

func (r *DashboardRepository) Revenue(ctx context.Context, from, to time.Time) (*model.RevenueReport, error) {
	rpt := &model.RevenueReport{}
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_amount), 0),
		        COALESCE(SUM(paid_cash), 0),
		        COALESCE(SUM(paid_terminal), 0),
		        COALESCE(SUM(paid_online), 0),
		        COALESCE(SUM(paid_debt), 0)
		 FROM sales WHERE created_at >= $1 AND created_at <= $2`,
		from, to,
	).Scan(&rpt.TotalRevenue, &rpt.Cash, &rpt.Terminal, &rpt.Online, &rpt.Debt)
	return rpt, err
}

func (r *DashboardRepository) Profit(ctx context.Context, from, to time.Time) (*model.ProfitReport, error) {
	rpt := &model.ProfitReport{}
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(si.subtotal), 0) AS revenue,
		        COALESCE(SUM(p.cost_price * si.quantity), 0) AS cost
		 FROM sale_items si
		 JOIN sales s ON s.id = si.sale_id
		 JOIN products p ON p.id = si.product_id
		 WHERE s.created_at >= $1 AND s.created_at <= $2`,
		from, to,
	).Scan(&rpt.TotalRevenue, &rpt.TotalCost)
	rpt.GrossProfit = rpt.TotalRevenue.Sub(rpt.TotalCost)
	return rpt, err
}

func (r *DashboardRepository) StockValue(ctx context.Context) (*model.StockValueReport, error) {
	rpt := &model.StockValueReport{}
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(cost_price * (qty_shop + qty_warehouse)), 0),
		        COALESCE(SUM(sell_price * (qty_shop + qty_warehouse)), 0),
		        COALESCE(SUM(qty_shop + qty_warehouse), 0)
		 FROM products WHERE is_deleted = false`,
	).Scan(&rpt.TotalCostValue, &rpt.TotalSellValue, &rpt.TotalItems)
	return rpt, err
}

func (r *DashboardRepository) SalesBySupplier(ctx context.Context, from, to time.Time) ([]model.SupplierSalesReport, error) {
	rows, err := r.db.Query(ctx,
		`SELECT COALESCE(p.supplier_id::text, 'unknown'), COALESCE(sup.name, 'No Supplier'),
		        COALESCE(SUM(si.subtotal), 0), COALESCE(SUM(si.quantity), 0)
		 FROM sale_items si
		 JOIN sales s ON s.id = si.sale_id
		 JOIN products p ON p.id = si.product_id
		 LEFT JOIN suppliers sup ON sup.id = p.supplier_id
		 WHERE s.created_at >= $1 AND s.created_at <= $2
		 GROUP BY p.supplier_id, sup.name
		 ORDER BY SUM(si.subtotal) DESC`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.SupplierSalesReport
	for rows.Next() {
		var rpt model.SupplierSalesReport
		if err := rows.Scan(&rpt.SupplierID, &rpt.SupplierName, &rpt.TotalSales, &rpt.ItemsSold); err != nil {
			return nil, err
		}
		reports = append(reports, rpt)
	}
	return reports, rows.Err()
}

func (r *DashboardRepository) PaidVsDebt(ctx context.Context) (*model.PaidVsDebtReport, error) {
	rpt := &model.PaidVsDebtReport{}
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(paid_cash + paid_terminal + paid_online), 0),
		        COALESCE(SUM(paid_debt), 0)
		 FROM sales`,
	).Scan(&rpt.TotalPaid, &rpt.TotalDebt)
	return rpt, err
}
