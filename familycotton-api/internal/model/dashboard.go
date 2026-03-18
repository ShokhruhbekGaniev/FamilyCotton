package model

import "github.com/shopspring/decimal"

type SafeBalance struct {
	Cash     decimal.Decimal `json:"cash"`
	Terminal decimal.Decimal `json:"terminal"`
	Online   decimal.Decimal `json:"online"`
}

type RevenueReport struct {
	TotalRevenue decimal.Decimal `json:"total_revenue"`
	Cash         decimal.Decimal `json:"cash"`
	Terminal     decimal.Decimal `json:"terminal"`
	Online       decimal.Decimal `json:"online"`
	Debt         decimal.Decimal `json:"debt"`
}

type ProfitReport struct {
	TotalRevenue decimal.Decimal `json:"total_revenue"`
	TotalCost    decimal.Decimal `json:"total_cost"`
	GrossProfit  decimal.Decimal `json:"gross_profit"`
}

type StockValueReport struct {
	TotalCostValue decimal.Decimal `json:"total_cost_value"`
	TotalSellValue decimal.Decimal `json:"total_sell_value"`
	TotalItems     int             `json:"total_items"`
}

type SupplierSalesReport struct {
	SupplierID   string          `json:"supplier_id"`
	SupplierName string          `json:"supplier_name"`
	TotalSales   decimal.Decimal `json:"total_sales"`
	ItemsSold    int             `json:"items_sold"`
}

type PaidVsDebtReport struct {
	TotalPaid decimal.Decimal `json:"total_paid"`
	TotalDebt decimal.Decimal `json:"total_debt"`
}
