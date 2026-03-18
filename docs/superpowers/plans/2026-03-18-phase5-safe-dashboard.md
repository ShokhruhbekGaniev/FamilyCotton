# Phase 5 — Safe & Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement safe balance/transactions endpoints, owner debt management, and dashboard analytics (revenue, profit, stock value, sales by supplier, paid vs debt).

**Architecture:** Mostly read-only aggregation queries. Safe balance computed via SUM on safe_transactions. Dashboard queries use date range filters. Owner deposit is the only write operation (settle debt + safe expense).

**Tech Stack:** Go 1.25, chi v5, pgx v5, shopspring/decimal

**Spec:** `docs/superpowers/specs/2026-03-18-familycotton-backend-design.md` (Sections 4.10, 4.11)

---

## File Map

```
familycotton-api/internal/
├── model/
│   └── dashboard.go            # NEW: dashboard response types
├── repository/
│   ├── safe_transaction.go     # MODIFY: add ListTx, GetBalance
│   ├── owner_debt.go           # MODIFY: add List, Settle
│   └── dashboard.go            # NEW: aggregation queries
├── service/
│   ├── safe.go                 # NEW: balance, transactions, owner deposit
│   └── dashboard.go            # NEW: revenue, profit, stock-value, etc.
├── handler/
│   ├── safe.go                 # NEW
│   └── dashboard.go            # NEW
├── router/router.go            # MODIFY
cmd/api/main.go                 # MODIFY
```

---

### Task 1: Dashboard Models + Safe/OwnerDebt Repository Extensions

**Files:**
- Create: `internal/model/dashboard.go`
- Modify: `internal/repository/safe_transaction.go`
- Modify: `internal/repository/owner_debt.go`
- Create: `internal/repository/dashboard.go`

- [ ] **Step 1: Create dashboard model**

Write `internal/model/dashboard.go`:
```go
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
```

- [ ] **Step 2: Extend safe_transaction repository**

Add to `internal/repository/safe_transaction.go` — the repo currently has no pool. Refactor to add a pool and list/balance methods:

```go
// Change struct to hold pool.
type SafeTransactionRepository struct {
	db *pgxpool.Pool
}

func NewSafeTransactionRepository(db *pgxpool.Pool) *SafeTransactionRepository {
	return &SafeTransactionRepository{db: db}
}

// Keep existing Create method (accepts DBTX).

func (r *SafeTransactionRepository) GetBalance(ctx context.Context) (*model.SafeBalance, error) {
	b := &model.SafeBalance{}
	err := r.db.QueryRow(ctx,
		`SELECT
		   COALESCE(SUM(CASE WHEN balance_type='cash' AND type='income' THEN amount ELSE 0 END), 0) -
		   COALESCE(SUM(CASE WHEN balance_type='cash' AND type='expense' THEN amount ELSE 0 END), 0),
		   COALESCE(SUM(CASE WHEN balance_type='terminal' AND type='income' THEN amount ELSE 0 END), 0) -
		   COALESCE(SUM(CASE WHEN balance_type='terminal' AND type='expense' THEN amount ELSE 0 END), 0),
		   COALESCE(SUM(CASE WHEN balance_type='online' AND type='income' THEN amount ELSE 0 END), 0) -
		   COALESCE(SUM(CASE WHEN balance_type='online' AND type='expense' THEN amount ELSE 0 END), 0)
		 FROM safe_transactions`,
	).Scan(&b.Cash, &b.Terminal, &b.Online)
	return b, err
}

func (r *SafeTransactionRepository) List(ctx context.Context, page, limit int) ([]model.SafeTransaction, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM safe_transactions`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, type, source, balance_type, amount, description, reference_id, created_at
		 FROM safe_transactions ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txns []model.SafeTransaction
	for rows.Next() {
		var t model.SafeTransaction
		if err := rows.Scan(&t.ID, &t.Type, &t.Source, &t.BalanceType,
			&t.Amount, &t.Description, &t.ReferenceID, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		txns = append(txns, t)
	}
	return txns, total, rows.Err()
}
```

**IMPORTANT:** This changes the constructor from `NewSafeTransactionRepository()` to `NewSafeTransactionRepository(db *pgxpool.Pool)`. Update ALL callers in main.go accordingly.

- [ ] **Step 3: Extend owner_debt repository**

Refactor `internal/repository/owner_debt.go` similarly — add pool and new methods:

```go
type OwnerDebtRepository struct {
	db *pgxpool.Pool
}

func NewOwnerDebtRepository(db *pgxpool.Pool) *OwnerDebtRepository {
	return &OwnerDebtRepository{db: db}
}

// Keep existing Create method (accepts DBTX).

func (r *OwnerDebtRepository) ListUnsettled(ctx context.Context) ([]model.OwnerDebt, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, shift_id, amount, is_settled, created_at, settled_at
		 FROM owner_debts WHERE is_settled = false ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var debts []model.OwnerDebt
	for rows.Next() {
		var d model.OwnerDebt
		if err := rows.Scan(&d.ID, &d.ShiftID, &d.Amount, &d.IsSettled, &d.CreatedAt, &d.SettledAt); err != nil {
			return nil, err
		}
		debts = append(debts, d)
	}
	return debts, rows.Err()
}

func (r *OwnerDebtRepository) Settle(ctx context.Context, tx DBTX, id uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE owner_debts SET is_settled = true, settled_at = NOW() WHERE id = $1 AND is_settled = false`,
		id,
	)
	return err
}

func (r *OwnerDebtRepository) TotalUnsettled(ctx context.Context) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM owner_debts WHERE is_settled = false`,
	).Scan(&total)
	return total, err
}
```

- [ ] **Step 4: Create dashboard repository**

Write `internal/repository/dashboard.go`:
```go
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
		var r model.SupplierSalesReport
		if err := rows.Scan(&r.SupplierID, &r.SupplierName, &r.TotalSales, &r.ItemsSold); err != nil {
			return nil, err
		}
		reports = append(reports, r)
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
```

- [ ] **Step 5: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/model/dashboard.go internal/repository/dashboard.go \
  internal/repository/safe_transaction.go internal/repository/owner_debt.go && \
git commit -m "feat: add dashboard models and repository, extend safe/owner_debt repos"
```

---

### Task 2: Safe Service + Dashboard Service

**Files:**
- Create: `internal/service/safe.go`
- Create: `internal/service/dashboard.go`

- [ ] **Step 1: Create safe service**

Write `internal/service/safe.go`:
```go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type SafeService struct {
	pool          *pgxpool.Pool
	safeRepo      *repository.SafeTransactionRepository
	ownerDebtRepo *repository.OwnerDebtRepository
}

func NewSafeService(pool *pgxpool.Pool, safeRepo *repository.SafeTransactionRepository, ownerDebtRepo *repository.OwnerDebtRepository) *SafeService {
	return &SafeService{pool: pool, safeRepo: safeRepo, ownerDebtRepo: ownerDebtRepo}
}

func (s *SafeService) GetBalance(ctx context.Context) (*model.SafeBalance, error) {
	return s.safeRepo.GetBalance(ctx)
}

func (s *SafeService) ListTransactions(ctx context.Context, page, limit int) ([]model.SafeTransaction, int, error) {
	return s.safeRepo.List(ctx, page, limit)
}

func (s *SafeService) ListOwnerDebts(ctx context.Context) ([]model.OwnerDebt, error) {
	return s.ownerDebtRepo.ListUnsettled(ctx)
}

type OwnerDepositRequest struct {
	Amount decimal.Decimal `json:"amount"`
}

func (s *SafeService) OwnerDeposit(ctx context.Context, req *OwnerDepositRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return model.NewAppError(model.ErrValidation, "amount must be positive")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Record safe expense (owner takes cash from safe).
	desc := "Owner deposit — settling online debt"
	refID := uuid.New()
	if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
		Type: "expense", Source: "owner_deposit", BalanceType: "cash",
		Amount: req.Amount, Description: &desc, ReferenceID: &refID,
	}); err != nil {
		return err
	}

	// Settle owner debts up to amount.
	remaining := req.Amount
	debts, err := s.ownerDebtRepo.ListUnsettled(ctx)
	if err != nil {
		return err
	}
	for _, d := range debts {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		if d.Amount.LessThanOrEqual(remaining) {
			if err := s.ownerDebtRepo.Settle(ctx, tx, d.ID); err != nil {
				return err
			}
			remaining = remaining.Sub(d.Amount)
		}
	}

	return tx.Commit(ctx)
}
```

- [ ] **Step 2: Create dashboard service**

Write `internal/service/dashboard.go`:
```go
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
```

- [ ] **Step 3: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/service/safe.go internal/service/dashboard.go && \
git commit -m "feat: add safe and dashboard services"
```

---

### Task 3: Handlers + Wire Routes + DI

**Files:**
- Create: `internal/handler/safe.go`
- Create: `internal/handler/dashboard.go`
- Modify: `internal/router/router.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Create safe handler**

Write `internal/handler/safe.go`:
```go
package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type SafeHandler struct {
	svc *service.SafeService
}

func NewSafeHandler(svc *service.SafeService) *SafeHandler {
	return &SafeHandler{svc: svc}
}

func (h *SafeHandler) Balance(w http.ResponseWriter, r *http.Request) {
	balance, err := h.svc.GetBalance(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, balance)
}

func (h *SafeHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	txns, total, err := h.svc.ListTransactions(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, txns, page, limit, total)
}

func (h *SafeHandler) OwnerDebts(w http.ResponseWriter, r *http.Request) {
	debts, err := h.svc.ListOwnerDebts(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, debts)
}

func (h *SafeHandler) OwnerDeposit(w http.ResponseWriter, r *http.Request) {
	var req service.OwnerDepositRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	if err := h.svc.OwnerDeposit(r.Context(), &req); err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, map[string]string{"message": "deposit recorded"})
}
```

Note: `decodeJSON` uses `model.NewAppError` for validation, but `OwnerDepositRequest` is in the service package. This is fine — `decodeJSON` only validates JSON parsing. The `OwnerDeposit` method in the service handles amount validation.

- [ ] **Step 2: Create dashboard handler**

Write `internal/handler/dashboard.go`:
```go
package handler

import (
	"net/http"
	"time"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		return time.Time{}, time.Time{}, model.NewAppError(model.ErrValidation, "from and to query params are required")
	}
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, model.NewAppError(model.ErrValidation, "invalid from date format, use YYYY-MM-DD")
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return time.Time{}, time.Time{}, model.NewAppError(model.ErrValidation, "invalid to date format, use YYYY-MM-DD")
	}
	// Set to end of day.
	to = to.Add(24*time.Hour - time.Nanosecond)
	return from, to, nil
}

func (h *DashboardHandler) Revenue(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		respondError(w, err)
		return
	}
	rpt, err := h.svc.Revenue(r.Context(), from, to)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}

func (h *DashboardHandler) Profit(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		respondError(w, err)
		return
	}
	rpt, err := h.svc.Profit(r.Context(), from, to)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}

func (h *DashboardHandler) StockValue(w http.ResponseWriter, r *http.Request) {
	rpt, err := h.svc.StockValue(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}

func (h *DashboardHandler) SalesBySupplier(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		respondError(w, err)
		return
	}
	rpt, err := h.svc.SalesBySupplier(r.Context(), from, to)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}

func (h *DashboardHandler) PaidVsDebt(w http.ResponseWriter, r *http.Request) {
	rpt, err := h.svc.PaidVsDebt(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, rpt)
}
```

- [ ] **Step 3: Update router — add safe and dashboard routes**

Add `safeHandler` and `dashboardHandler` params to `router.New()`. Add routes:

```go
// Safe (owner only).
r.Route("/safe", func(r chi.Router) {
    r.Use(middleware.RequireRole("owner"))
    r.Get("/balance", safeHandler.Balance)
    r.Get("/transactions", safeHandler.Transactions)
    r.Get("/owner-debts", safeHandler.OwnerDebts)
    r.Post("/owner-deposit", safeHandler.OwnerDeposit)
})

// Dashboard (owner only).
r.Route("/dashboard", func(r chi.Router) {
    r.Use(middleware.RequireRole("owner"))
    r.Get("/revenue", dashboardHandler.Revenue)
    r.Get("/profit", dashboardHandler.Profit)
    r.Get("/stock-value", dashboardHandler.StockValue)
    r.Get("/sales-by-supplier", dashboardHandler.SalesBySupplier)
    r.Get("/paid-vs-debt", dashboardHandler.PaidVsDebt)
})
```

- [ ] **Step 4: Update main.go**

IMPORTANT: `NewSafeTransactionRepository` now requires `pool` as argument. Update the existing call:
```go
safeTransactionRepo := repository.NewSafeTransactionRepository(pool)
ownerDebtRepo := repository.NewOwnerDebtRepository(pool)
```

Add Phase 5:
```go
// Phase 5 repositories.
dashboardRepo := repository.NewDashboardRepository(pool)

// Phase 5 services.
safeService := service.NewSafeService(pool, safeTransactionRepo, ownerDebtRepo)
dashboardService := service.NewDashboardService(dashboardRepo)

// Phase 5 handlers.
safeHandler := handler.NewSafeHandler(safeService)
dashboardHandler := handler.NewDashboardHandler(dashboardService)
```

Update `router.New(...)` to pass safeHandler and dashboardHandler.

- [ ] **Step 5: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/ cmd/ && \
git commit -m "feat: add safe and dashboard handlers, wire Phase 5 routes"
```

---

### Task 4: Integration Smoke Test

- [ ] **Step 1: Rebuild Docker, test safe balance**
- [ ] **Step 2: Test dashboard revenue/profit with date range**
- [ ] **Step 3: Test owner deposit**
- [ ] **Step 4: Stop Docker**
