# Phase 3 — Sales Operations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement shifts, sales with split payments, sale returns, and client payments — all with multi-table transactional integrity.

**Architecture:** Introduces DBTX interface for transaction support. Services manage `pgx.Tx` for complex operations (sale creation, shift close). `SELECT FOR UPDATE` on product rows prevents race conditions on stock. Safe transactions and owner debts are created as side-effects of shift close.

**Tech Stack:** Go 1.25, chi v5, pgx v5 (transactions), shopspring/decimal

**Spec:** `docs/superpowers/specs/2026-03-18-familycotton-backend-design.md` (Sections 6.1-6.3)

---

## File Map

```
familycotton-api/internal/
├── repository/
│   ├── db.go                  # NEW: DBTX interface
│   ├── product.go             # MODIFY: add GetByIDForUpdate(ctx, tx, id)
│   ├── shift.go               # NEW: ShiftRepository
│   ├── sale.go                # NEW: SaleRepository + SaleItemRepository
│   ├── sale_return.go         # NEW: SaleReturnRepository
│   ├── client_payment.go      # NEW: ClientPaymentRepository
│   ├── safe_transaction.go    # NEW: SafeTransactionRepository
│   └── owner_debt.go          # NEW: OwnerDebtRepository
├── model/
│   ├── shift.go               # NEW
│   ├── sale.go                # NEW: Sale, SaleItem, CreateSaleRequest
│   ├── sale_return.go         # NEW
│   ├── client_payment.go      # NEW
│   ├── safe_transaction.go    # NEW
│   └── owner_debt.go          # NEW
├── service/
│   ├── shift.go               # NEW: open/close with aggregation
│   ├── sale.go                # NEW: transactional sale creation
│   ├── sale_return.go         # NEW: return with proportional refund
│   └── client_payment.go      # NEW
├── handler/
│   ├── shift.go               # NEW
│   ├── sale.go                # NEW
│   ├── sale_return.go         # NEW
│   └── client_payment.go      # NEW
├── router/router.go           # MODIFY: add Phase 3 routes
cmd/api/main.go                # MODIFY: add Phase 3 DI
```

---

### Task 1: DBTX Interface

**Files:**
- Create: `familycotton-api/internal/repository/db.go`

- [ ] **Step 1: Create DBTX interface**

Write `familycotton-api/internal/repository/db.go`:
```go
package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd familycotton-api && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/repository/db.go
git commit -m "feat: add DBTX interface for transaction support"
```

---

### Task 2: All Phase 3 Models

**Files:**
- Create: `familycotton-api/internal/model/shift.go`
- Create: `familycotton-api/internal/model/sale.go`
- Create: `familycotton-api/internal/model/sale_return.go`
- Create: `familycotton-api/internal/model/client_payment.go`
- Create: `familycotton-api/internal/model/safe_transaction.go`
- Create: `familycotton-api/internal/model/owner_debt.go`

- [ ] **Step 1: Create shift model**

Write `familycotton-api/internal/model/shift.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Shift struct {
	ID             uuid.UUID       `json:"id"`
	OpenedBy       uuid.UUID       `json:"opened_by"`
	ClosedBy       *uuid.UUID      `json:"closed_by"`
	OpenedAt       time.Time       `json:"opened_at"`
	ClosedAt       *time.Time      `json:"closed_at"`
	TotalCash      decimal.Decimal `json:"total_cash"`
	TotalTerminal  decimal.Decimal `json:"total_terminal"`
	TotalOnline    decimal.Decimal `json:"total_online"`
	TotalDebtSales decimal.Decimal `json:"total_debt_sales"`
	Status         string          `json:"status"`
}
```

- [ ] **Step 2: Create sale model**

Write `familycotton-api/internal/model/sale.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Sale struct {
	ID           uuid.UUID       `json:"id"`
	ShiftID      uuid.UUID       `json:"shift_id"`
	ClientID     *uuid.UUID      `json:"client_id"`
	TotalAmount  decimal.Decimal `json:"total_amount"`
	PaidCash     decimal.Decimal `json:"paid_cash"`
	PaidTerminal decimal.Decimal `json:"paid_terminal"`
	PaidOnline   decimal.Decimal `json:"paid_online"`
	PaidDebt     decimal.Decimal `json:"paid_debt"`
	CreatedBy    uuid.UUID       `json:"created_by"`
	CreatedAt    time.Time       `json:"created_at"`
	Items        []SaleItem      `json:"items,omitempty"`
}

type SaleItem struct {
	ID        uuid.UUID       `json:"id"`
	SaleID    uuid.UUID       `json:"sale_id"`
	ProductID uuid.UUID       `json:"product_id"`
	Quantity  int             `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
	Subtotal  decimal.Decimal `json:"subtotal"`
}

type CreateSaleItemRequest struct {
	ProductID uuid.UUID       `json:"product_id"`
	Quantity  int             `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
}

type CreateSaleRequest struct {
	ClientID     *uuid.UUID            `json:"client_id"`
	Items        []CreateSaleItemRequest `json:"items"`
	PaidCash     decimal.Decimal       `json:"paid_cash"`
	PaidTerminal decimal.Decimal       `json:"paid_terminal"`
	PaidOnline   decimal.Decimal       `json:"paid_online"`
	PaidDebt     decimal.Decimal       `json:"paid_debt"`
}

func (r *CreateSaleRequest) Validate() error {
	if len(r.Items) == 0 {
		return NewAppError(ErrValidation, "at least one item is required")
	}
	for _, item := range r.Items {
		if item.Quantity <= 0 {
			return NewAppError(ErrValidation, "item quantity must be positive")
		}
		if item.UnitPrice.IsNegative() {
			return NewAppError(ErrValidation, "item unit_price cannot be negative")
		}
	}
	if r.PaidCash.IsNegative() || r.PaidTerminal.IsNegative() || r.PaidOnline.IsNegative() || r.PaidDebt.IsNegative() {
		return NewAppError(ErrValidation, "payment amounts cannot be negative")
	}
	if r.PaidDebt.IsPositive() && r.ClientID == nil {
		return NewAppError(ErrValidation, "client_id is required when paid_debt > 0")
	}
	return nil
}
```

- [ ] **Step 3: Create sale_return model**

Write `familycotton-api/internal/model/sale_return.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SaleReturn struct {
	ID              uuid.UUID       `json:"id"`
	SaleID          uuid.UUID       `json:"sale_id"`
	SaleItemID      uuid.UUID       `json:"sale_item_id"`
	NewProductID    *uuid.UUID      `json:"new_product_id"`
	Quantity        int             `json:"quantity"`
	ReturnType      string          `json:"return_type"`
	RefundAmount    decimal.Decimal `json:"refund_amount"`
	SurchargeAmount decimal.Decimal `json:"surcharge_amount"`
	CreatedBy       uuid.UUID       `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
}

type CreateSaleReturnRequest struct {
	SaleID       uuid.UUID  `json:"sale_id"`
	SaleItemID   uuid.UUID  `json:"sale_item_id"`
	NewProductID *uuid.UUID `json:"new_product_id"`
	Quantity     int        `json:"quantity"`
	ReturnType   string     `json:"return_type"`
}

func (r *CreateSaleReturnRequest) Validate() error {
	if r.Quantity <= 0 {
		return NewAppError(ErrValidation, "quantity must be positive")
	}
	validTypes := map[string]bool{"full": true, "exchange": true, "exchange_diff": true}
	if !validTypes[r.ReturnType] {
		return NewAppError(ErrValidation, "return_type must be 'full', 'exchange', or 'exchange_diff'")
	}
	if (r.ReturnType == "exchange" || r.ReturnType == "exchange_diff") && r.NewProductID == nil {
		return NewAppError(ErrValidation, "new_product_id is required for exchange returns")
	}
	return nil
}
```

- [ ] **Step 4: Create client_payment model**

Write `familycotton-api/internal/model/client_payment.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ClientPayment struct {
	ID            uuid.UUID       `json:"id"`
	ClientID      uuid.UUID       `json:"client_id"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
	CreatedBy     uuid.UUID       `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`
}

type CreateClientPaymentRequest struct {
	ClientID      uuid.UUID       `json:"client_id"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
}

func (r *CreateClientPaymentRequest) Validate() error {
	if r.Amount.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "amount must be positive")
	}
	validMethods := map[string]bool{"cash": true, "terminal": true, "online": true}
	if !validMethods[r.PaymentMethod] {
		return NewAppError(ErrValidation, "payment_method must be 'cash', 'terminal', or 'online'")
	}
	return nil
}
```

- [ ] **Step 5: Create safe_transaction model**

Write `familycotton-api/internal/model/safe_transaction.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SafeTransaction struct {
	ID          uuid.UUID       `json:"id"`
	Type        string          `json:"type"`
	Source      string          `json:"source"`
	BalanceType string          `json:"balance_type"`
	Amount      decimal.Decimal `json:"amount"`
	Description *string         `json:"description"`
	ReferenceID *uuid.UUID      `json:"reference_id"`
	CreatedAt   time.Time       `json:"created_at"`
}
```

- [ ] **Step 6: Create owner_debt model**

Write `familycotton-api/internal/model/owner_debt.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OwnerDebt struct {
	ID        uuid.UUID       `json:"id"`
	ShiftID   uuid.UUID       `json:"shift_id"`
	Amount    decimal.Decimal `json:"amount"`
	IsSettled bool            `json:"is_settled"`
	CreatedAt time.Time       `json:"created_at"`
	SettledAt *time.Time      `json:"settled_at"`
}
```

- [ ] **Step 7: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/model/shift.go internal/model/sale.go internal/model/sale_return.go \
  internal/model/client_payment.go internal/model/safe_transaction.go internal/model/owner_debt.go && \
git commit -m "feat: add Phase 3 models (shift, sale, sale_return, client_payment, safe_transaction, owner_debt)"
```

---

### Task 3: All Phase 3 Repositories

**Files:**
- Create: `familycotton-api/internal/repository/shift.go`
- Create: `familycotton-api/internal/repository/sale.go`
- Create: `familycotton-api/internal/repository/sale_return.go`
- Create: `familycotton-api/internal/repository/client_payment.go`
- Create: `familycotton-api/internal/repository/safe_transaction.go`
- Create: `familycotton-api/internal/repository/owner_debt.go`
- Modify: `familycotton-api/internal/repository/product.go` (add GetByIDForUpdate)

All transactional repositories accept DBTX so they can work with both pool and tx.

- [ ] **Step 1: Create shift repository**

Write `familycotton-api/internal/repository/shift.go`:
```go
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type ShiftRepository struct {
	db *pgxpool.Pool
}

func NewShiftRepository(db *pgxpool.Pool) *ShiftRepository {
	return &ShiftRepository{db: db}
}

func (r *ShiftRepository) Create(ctx context.Context, s *model.Shift) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO shifts (id, opened_by) VALUES ($1, $2)
		 RETURNING opened_at, status`,
		s.ID, s.OpenedBy,
	).Scan(&s.OpenedAt, &s.Status)
}

func (r *ShiftRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Shift, error) {
	s := &model.Shift{}
	err := r.db.QueryRow(ctx,
		`SELECT id, opened_by, closed_by, opened_at, closed_at,
		        total_cash, total_terminal, total_online, total_debt_sales, status
		 FROM shifts WHERE id = $1`, id,
	).Scan(&s.ID, &s.OpenedBy, &s.ClosedBy, &s.OpenedAt, &s.ClosedAt,
		&s.TotalCash, &s.TotalTerminal, &s.TotalOnline, &s.TotalDebtSales, &s.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "shift not found")
	}
	return s, err
}

func (r *ShiftRepository) GetCurrentOpen(ctx context.Context) (*model.Shift, error) {
	s := &model.Shift{}
	err := r.db.QueryRow(ctx,
		`SELECT id, opened_by, closed_by, opened_at, closed_at,
		        total_cash, total_terminal, total_online, total_debt_sales, status
		 FROM shifts WHERE status = 'open' ORDER BY opened_at DESC LIMIT 1`,
	).Scan(&s.ID, &s.OpenedBy, &s.ClosedBy, &s.OpenedAt, &s.ClosedAt,
		&s.TotalCash, &s.TotalTerminal, &s.TotalOnline, &s.TotalDebtSales, &s.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *ShiftRepository) CloseShift(ctx context.Context, tx DBTX, s *model.Shift) error {
	return tx.QueryRow(ctx,
		`UPDATE shifts SET closed_by=$1, closed_at=NOW(), total_cash=$2, total_terminal=$3,
		        total_online=$4, total_debt_sales=$5, status='closed'
		 WHERE id=$6 AND status='open'
		 RETURNING closed_at`,
		s.ClosedBy, s.TotalCash, s.TotalTerminal, s.TotalOnline, s.TotalDebtSales, s.ID,
	).Scan(&s.ClosedAt)
}

func (r *ShiftRepository) List(ctx context.Context, page, limit int) ([]model.Shift, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM shifts`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, opened_by, closed_by, opened_at, closed_at,
		        total_cash, total_terminal, total_online, total_debt_sales, status
		 FROM shifts ORDER BY opened_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shifts []model.Shift
	for rows.Next() {
		var s model.Shift
		if err := rows.Scan(&s.ID, &s.OpenedBy, &s.ClosedBy, &s.OpenedAt, &s.ClosedAt,
			&s.TotalCash, &s.TotalTerminal, &s.TotalOnline, &s.TotalDebtSales, &s.Status); err != nil {
			return nil, 0, err
		}
		shifts = append(shifts, s)
	}
	return shifts, total, rows.Err()
}

// AggregateSales sums payment fields for all sales in a shift.
func (r *ShiftRepository) AggregateSales(ctx context.Context, tx DBTX, shiftID uuid.UUID) (cash, terminal, online, debt decimal.Decimal, err error) {
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(paid_cash),0), COALESCE(SUM(paid_terminal),0),
		        COALESCE(SUM(paid_online),0), COALESCE(SUM(paid_debt),0)
		 FROM sales WHERE shift_id = $1`, shiftID,
	).Scan(&cash, &terminal, &online, &debt)
	return
}
```

- [ ] **Step 2: Create sale repository**

Write `familycotton-api/internal/repository/sale.go`:
```go
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type SaleRepository struct {
	db *pgxpool.Pool
}

func NewSaleRepository(db *pgxpool.Pool) *SaleRepository {
	return &SaleRepository{db: db}
}

func (r *SaleRepository) CreateSale(ctx context.Context, tx DBTX, s *model.Sale) error {
	return tx.QueryRow(ctx,
		`INSERT INTO sales (id, shift_id, client_id, total_amount, paid_cash, paid_terminal, paid_online, paid_debt, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at`,
		s.ID, s.ShiftID, s.ClientID, s.TotalAmount,
		s.PaidCash, s.PaidTerminal, s.PaidOnline, s.PaidDebt, s.CreatedBy,
	).Scan(&s.CreatedAt)
}

func (r *SaleRepository) CreateSaleItem(ctx context.Context, tx DBTX, item *model.SaleItem) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO sale_items (id, sale_id, product_id, quantity, unit_price, subtotal)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		item.ID, item.SaleID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal,
	)
	return err
}

func (r *SaleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Sale, error) {
	s := &model.Sale{}
	err := r.db.QueryRow(ctx,
		`SELECT id, shift_id, client_id, total_amount, paid_cash, paid_terminal,
		        paid_online, paid_debt, created_by, created_at
		 FROM sales WHERE id = $1`, id,
	).Scan(&s.ID, &s.ShiftID, &s.ClientID, &s.TotalAmount,
		&s.PaidCash, &s.PaidTerminal, &s.PaidOnline, &s.PaidDebt,
		&s.CreatedBy, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "sale not found")
	}
	if err != nil {
		return nil, err
	}

	// Load items.
	rows, err := r.db.Query(ctx,
		`SELECT id, sale_id, product_id, quantity, unit_price, subtotal
		 FROM sale_items WHERE sale_id = $1`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item model.SaleItem
		if err := rows.Scan(&item.ID, &item.SaleID, &item.ProductID,
			&item.Quantity, &item.UnitPrice, &item.Subtotal); err != nil {
			return nil, err
		}
		s.Items = append(s.Items, item)
	}
	return s, rows.Err()
}

func (r *SaleRepository) List(ctx context.Context, shiftID *uuid.UUID, clientID *uuid.UUID, page, limit int) ([]model.Sale, int, error) {
	where := ""
	var args []any
	idx := 1

	if shiftID != nil {
		where += fmt.Sprintf(" AND shift_id = $%d", idx)
		args = append(args, *shiftID)
		idx++
	}
	if clientID != nil {
		where += fmt.Sprintf(" AND client_id = $%d", idx)
		args = append(args, *clientID)
		idx++
	}

	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM sales WHERE 1=1 %s", where)
	err := r.db.QueryRow(ctx, countQ, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, (page-1)*limit)
	listQ := fmt.Sprintf(
		`SELECT id, shift_id, client_id, total_amount, paid_cash, paid_terminal,
		        paid_online, paid_debt, created_by, created_at
		 FROM sales WHERE 1=1 %s
		 ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1,
	)

	rows, err := r.db.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sales []model.Sale
	for rows.Next() {
		var s model.Sale
		if err := rows.Scan(&s.ID, &s.ShiftID, &s.ClientID, &s.TotalAmount,
			&s.PaidCash, &s.PaidTerminal, &s.PaidOnline, &s.PaidDebt,
			&s.CreatedBy, &s.CreatedAt); err != nil {
			return nil, 0, err
		}
		sales = append(sales, s)
	}
	return sales, total, rows.Err()
}

// GetSaleItemByID fetches a single sale_item.
func (r *SaleRepository) GetSaleItemByID(ctx context.Context, id uuid.UUID) (*model.SaleItem, error) {
	item := &model.SaleItem{}
	err := r.db.QueryRow(ctx,
		`SELECT id, sale_id, product_id, quantity, unit_price, subtotal
		 FROM sale_items WHERE id = $1`, id,
	).Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "sale item not found")
	}
	return item, err
}

// SumReturnedQty returns total already-returned quantity for a sale_item.
func (r *SaleRepository) SumReturnedQty(ctx context.Context, saleItemID uuid.UUID) (int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity), 0) FROM sale_returns WHERE sale_item_id = $1`, saleItemID,
	).Scan(&total)
	return total, err
}
```

**Note:** The `List` method uses `fmt.Sprintf` — add `"fmt"` to imports.

- [ ] **Step 3: Create safe_transaction repository**

Write `familycotton-api/internal/repository/safe_transaction.go`:
```go
package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
)

type SafeTransactionRepository struct{}

func NewSafeTransactionRepository() *SafeTransactionRepository {
	return &SafeTransactionRepository{}
}

func (r *SafeTransactionRepository) Create(ctx context.Context, tx DBTX, st *model.SafeTransaction) error {
	st.ID = uuid.New()
	_, err := tx.Exec(ctx,
		`INSERT INTO safe_transactions (id, type, source, balance_type, amount, description, reference_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		st.ID, st.Type, st.Source, st.BalanceType, st.Amount, st.Description, st.ReferenceID,
	)
	return err
}
```

- [ ] **Step 4: Create owner_debt repository**

Write `familycotton-api/internal/repository/owner_debt.go`:
```go
package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
)

type OwnerDebtRepository struct{}

func NewOwnerDebtRepository() *OwnerDebtRepository {
	return &OwnerDebtRepository{}
}

func (r *OwnerDebtRepository) Create(ctx context.Context, tx DBTX, od *model.OwnerDebt) error {
	od.ID = uuid.New()
	return tx.QueryRow(ctx,
		`INSERT INTO owner_debts (id, shift_id, amount) VALUES ($1, $2, $3)
		 RETURNING created_at`,
		od.ID, od.ShiftID, od.Amount,
	).Scan(&od.CreatedAt)
}
```

- [ ] **Step 5: Create sale_return repository**

Write `familycotton-api/internal/repository/sale_return.go`:
```go
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type SaleReturnRepository struct {
	db *pgxpool.Pool
}

func NewSaleReturnRepository(db *pgxpool.Pool) *SaleReturnRepository {
	return &SaleReturnRepository{db: db}
}

func (r *SaleReturnRepository) Create(ctx context.Context, tx DBTX, sr *model.SaleReturn) error {
	return tx.QueryRow(ctx,
		`INSERT INTO sale_returns (id, sale_id, sale_item_id, new_product_id, quantity, return_type, refund_amount, surcharge_amount, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at`,
		sr.ID, sr.SaleID, sr.SaleItemID, sr.NewProductID, sr.Quantity,
		sr.ReturnType, sr.RefundAmount, sr.SurchargeAmount, sr.CreatedBy,
	).Scan(&sr.CreatedAt)
}

func (r *SaleReturnRepository) List(ctx context.Context, saleID *uuid.UUID, page, limit int) ([]model.SaleReturn, int, error) {
	where := ""
	var args []any
	idx := 1

	if saleID != nil {
		where = fmt.Sprintf(" AND sale_id = $%d", idx)
		args = append(args, *saleID)
		idx++
	}

	var total int
	err := r.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM sale_returns WHERE 1=1 %s", where), args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(
			`SELECT id, sale_id, sale_item_id, new_product_id, quantity, return_type,
			        refund_amount, surcharge_amount, created_by, created_at
			 FROM sale_returns WHERE 1=1 %s
			 ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
			where, idx, idx+1,
		), args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var returns []model.SaleReturn
	for rows.Next() {
		var sr model.SaleReturn
		if err := rows.Scan(&sr.ID, &sr.SaleID, &sr.SaleItemID, &sr.NewProductID,
			&sr.Quantity, &sr.ReturnType, &sr.RefundAmount, &sr.SurchargeAmount,
			&sr.CreatedBy, &sr.CreatedAt); err != nil {
			return nil, 0, err
		}
		returns = append(returns, sr)
	}
	return returns, total, rows.Err()
}

// Unused import guard.
var _ = errors.Is
var _ = pgx.ErrNoRows
```

- [ ] **Step 6: Create client_payment repository**

Write `familycotton-api/internal/repository/client_payment.go`:
```go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type ClientPaymentRepository struct {
	db *pgxpool.Pool
}

func NewClientPaymentRepository(db *pgxpool.Pool) *ClientPaymentRepository {
	return &ClientPaymentRepository{db: db}
}

func (r *ClientPaymentRepository) Create(ctx context.Context, tx DBTX, cp *model.ClientPayment) error {
	return tx.QueryRow(ctx,
		`INSERT INTO client_payments (id, client_id, amount, payment_method, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		cp.ID, cp.ClientID, cp.Amount, cp.PaymentMethod, cp.CreatedBy,
	).Scan(&cp.CreatedAt)
}
```

- [ ] **Step 7: Add GetByIDForUpdate to product repository**

Add this method to `familycotton-api/internal/repository/product.go`:
```go
// GetByIDForUpdate locks the product row within a transaction.
func (r *ProductRepository) GetByIDForUpdate(ctx context.Context, tx DBTX, id uuid.UUID) (*model.Product, error) {
	p := &model.Product{}
	err := tx.QueryRow(ctx,
		`SELECT id, sku, name, brand, supplier_id, photo_url, cost_price, sell_price,
		        qty_shop, qty_warehouse, is_deleted, created_at, updated_at
		 FROM products WHERE id = $1 AND is_deleted = false
		 FOR UPDATE`, id,
	).Scan(&p.ID, &p.SKU, &p.Name, &p.Brand, &p.SupplierID, &p.PhotoURL,
		&p.CostPrice, &p.SellPrice, &p.QtyShop, &p.QtyWarehouse,
		&p.IsDeleted, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "product not found")
	}
	if err != nil {
		return nil, err
	}
	p.Margin = p.SellPrice.Sub(p.CostPrice)
	return p, nil
}

// UpdateStock updates qty_shop within a transaction.
func (r *ProductRepository) UpdateStock(ctx context.Context, tx DBTX, id uuid.UUID, qtyShop, qtyWarehouse int) error {
	_, err := tx.Exec(ctx,
		`UPDATE products SET qty_shop = $1, qty_warehouse = $2, updated_at = NOW()
		 WHERE id = $3 AND is_deleted = false`,
		qtyShop, qtyWarehouse, id,
	)
	return err
}
```

Also add the `DBTX` import usage — the methods use `DBTX` from `db.go` in the same package.

- [ ] **Step 8: Also add UpdateDebt to client repository**

Add this method to `familycotton-api/internal/repository/client.go`:
```go
// UpdateDebt adjusts client total_debt within a transaction.
func (r *ClientRepository) UpdateDebt(ctx context.Context, tx DBTX, clientID uuid.UUID, delta decimal.Decimal) error {
	_, err := tx.Exec(ctx,
		`UPDATE clients SET total_debt = total_debt + $1, updated_at = NOW()
		 WHERE id = $2 AND is_deleted = false`,
		delta, clientID,
	)
	return err
}
```

Add `"github.com/shopspring/decimal"` to client.go imports.

- [ ] **Step 9: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/repository/ && \
git commit -m "feat: add Phase 3 repositories (shift, sale, sale_return, client_payment, safe_transaction, owner_debt)"
```

---

### Task 4: Shift Service + Sale Service

**Files:**
- Create: `familycotton-api/internal/service/shift.go`
- Create: `familycotton-api/internal/service/sale.go`

- [ ] **Step 1: Create shift service**

Write `familycotton-api/internal/service/shift.go`:
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

type ShiftService struct {
	pool          *pgxpool.Pool
	shiftRepo     *repository.ShiftRepository
	safeRepo      *repository.SafeTransactionRepository
	ownerDebtRepo *repository.OwnerDebtRepository
}

func NewShiftService(
	pool *pgxpool.Pool,
	shiftRepo *repository.ShiftRepository,
	safeRepo *repository.SafeTransactionRepository,
	ownerDebtRepo *repository.OwnerDebtRepository,
) *ShiftService {
	return &ShiftService{
		pool:          pool,
		shiftRepo:     shiftRepo,
		safeRepo:      safeRepo,
		ownerDebtRepo: ownerDebtRepo,
	}
}

func (s *ShiftService) Open(ctx context.Context, userID uuid.UUID) (*model.Shift, error) {
	// Check no open shift exists.
	current, err := s.shiftRepo.GetCurrentOpen(ctx)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return nil, model.NewAppError(model.ErrValidation, "a shift is already open")
	}

	shift := &model.Shift{
		ID:       uuid.New(),
		OpenedBy: userID,
	}
	if err := s.shiftRepo.Create(ctx, shift); err != nil {
		return nil, err
	}
	return shift, nil
}

func (s *ShiftService) Close(ctx context.Context, userID uuid.UUID) (*model.Shift, error) {
	current, err := s.shiftRepo.GetCurrentOpen(ctx)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, model.NewAppError(model.ErrValidation, "no open shift to close")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Aggregate sales.
	cash, terminal, online, debt, err := s.shiftRepo.AggregateSales(ctx, tx, current.ID)
	if err != nil {
		return nil, err
	}

	current.ClosedBy = &userID
	current.TotalCash = cash
	current.TotalTerminal = terminal
	current.TotalOnline = online
	current.TotalDebtSales = debt

	if err := s.shiftRepo.CloseShift(ctx, tx, current); err != nil {
		return nil, err
	}

	// Safe transactions for cash.
	if cash.IsPositive() {
		desc := "Shift cash income"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "income", Source: "shift", BalanceType: "cash",
			Amount: cash, Description: &desc, ReferenceID: &current.ID,
		}); err != nil {
			return nil, err
		}
	}

	// Safe transactions for terminal.
	if terminal.IsPositive() {
		desc := "Shift terminal income"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "income", Source: "shift", BalanceType: "terminal",
			Amount: terminal, Description: &desc, ReferenceID: &current.ID,
		}); err != nil {
			return nil, err
		}
	}

	// Online: transfer to safe as cash + create owner debt.
	if online.IsPositive() {
		desc := "Online payments transferred as cash"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "income", Source: "online_owner_debt", BalanceType: "cash",
			Amount: online, Description: &desc, ReferenceID: &current.ID,
		}); err != nil {
			return nil, err
		}

		if err := s.ownerDebtRepo.Create(ctx, tx, &model.OwnerDebt{
			ShiftID: current.ID,
			Amount:  online,
		}); err != nil {
			return nil, err
		}
	}

	current.Status = "closed"
	return current, tx.Commit(ctx)
}

func (s *ShiftService) GetCurrent(ctx context.Context) (*model.Shift, error) {
	current, err := s.shiftRepo.GetCurrentOpen(ctx)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, model.NewAppError(model.ErrNotFound, "no open shift")
	}
	return current, nil
}

func (s *ShiftService) List(ctx context.Context, page, limit int) ([]model.Shift, int, error) {
	return s.shiftRepo.List(ctx, page, limit)
}
```

- [ ] **Step 2: Create sale service**

Write `familycotton-api/internal/service/sale.go`:
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

type SaleService struct {
	pool        *pgxpool.Pool
	saleRepo    *repository.SaleRepository
	shiftRepo   *repository.ShiftRepository
	productRepo *repository.ProductRepository
	clientRepo  *repository.ClientRepository
}

func NewSaleService(
	pool *pgxpool.Pool,
	saleRepo *repository.SaleRepository,
	shiftRepo *repository.ShiftRepository,
	productRepo *repository.ProductRepository,
	clientRepo *repository.ClientRepository,
) *SaleService {
	return &SaleService{
		pool:        pool,
		saleRepo:    saleRepo,
		shiftRepo:   shiftRepo,
		productRepo: productRepo,
		clientRepo:  clientRepo,
	}
}

func (s *SaleService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateSaleRequest) (*model.Sale, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify shift is open.
	shift, err := s.shiftRepo.GetCurrentOpen(ctx)
	if err != nil {
		return nil, err
	}
	if shift == nil {
		return nil, model.NewAppError(model.ErrValidation, "no open shift")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Build items, compute total, lock and deduct stock.
	var items []model.SaleItem
	totalAmount := decimal.Zero

	for _, ri := range req.Items {
		product, err := s.productRepo.GetByIDForUpdate(ctx, tx, ri.ProductID)
		if err != nil {
			return nil, err
		}
		if product.QtyShop < ri.Quantity {
			return nil, model.NewAppError(model.ErrValidation, "insufficient stock for product "+product.Name)
		}

		subtotal := ri.UnitPrice.Mul(decimal.NewFromInt(int64(ri.Quantity)))
		item := model.SaleItem{
			ID:        uuid.New(),
			ProductID: ri.ProductID,
			Quantity:  ri.Quantity,
			UnitPrice: ri.UnitPrice,
			Subtotal:  subtotal,
		}
		items = append(items, item)
		totalAmount = totalAmount.Add(subtotal)

		// Deduct stock.
		if err := s.productRepo.UpdateStock(ctx, tx, product.ID, product.QtyShop-ri.Quantity, product.QtyWarehouse); err != nil {
			return nil, err
		}
	}

	// Validate payment split.
	paidTotal := req.PaidCash.Add(req.PaidTerminal).Add(req.PaidOnline).Add(req.PaidDebt)
	if !paidTotal.Equal(totalAmount) {
		return nil, model.NewAppError(model.ErrValidation, "payment amounts must equal total_amount")
	}

	// Create sale.
	sale := &model.Sale{
		ID:           uuid.New(),
		ShiftID:      shift.ID,
		ClientID:     req.ClientID,
		TotalAmount:  totalAmount,
		PaidCash:     req.PaidCash,
		PaidTerminal: req.PaidTerminal,
		PaidOnline:   req.PaidOnline,
		PaidDebt:     req.PaidDebt,
		CreatedBy:    userID,
	}

	if err := s.saleRepo.CreateSale(ctx, tx, sale); err != nil {
		return nil, err
	}

	// Create sale items.
	for i := range items {
		items[i].SaleID = sale.ID
		if err := s.saleRepo.CreateSaleItem(ctx, tx, &items[i]); err != nil {
			return nil, err
		}
	}

	// Update client debt if paid_debt > 0.
	if req.PaidDebt.IsPositive() && req.ClientID != nil {
		if err := s.clientRepo.UpdateDebt(ctx, tx, *req.ClientID, req.PaidDebt); err != nil {
			return nil, err
		}
	}

	sale.Items = items
	return sale, tx.Commit(ctx)
}

func (s *SaleService) GetByID(ctx context.Context, id uuid.UUID) (*model.Sale, error) {
	return s.saleRepo.GetByID(ctx, id)
}

func (s *SaleService) List(ctx context.Context, shiftID, clientID *uuid.UUID, page, limit int) ([]model.Sale, int, error) {
	return s.saleRepo.List(ctx, shiftID, clientID, page, limit)
}
```

- [ ] **Step 3: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/service/shift.go internal/service/sale.go && \
git commit -m "feat: add shift and sale services with transactional business logic"
```

---

### Task 5: Sale Return Service + Client Payment Service

**Files:**
- Create: `familycotton-api/internal/service/sale_return.go`
- Create: `familycotton-api/internal/service/client_payment.go`

- [ ] **Step 1: Create sale_return service**

Write `familycotton-api/internal/service/sale_return.go`:
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
		return nil, model.NewAppError(model.ErrValidation, "return quantity exceeds available")
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
			return nil, model.NewAppError(model.ErrValidation, "insufficient stock for exchange product")
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
			return nil, model.NewAppError(model.ErrValidation, "insufficient stock for exchange product")
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
```

- [ ] **Step 2: Create client_payment service**

Write `familycotton-api/internal/service/client_payment.go`:
```go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type ClientPaymentService struct {
	pool        *pgxpool.Pool
	paymentRepo *repository.ClientPaymentRepository
	clientRepo  *repository.ClientRepository
	safeRepo    *repository.SafeTransactionRepository
}

func NewClientPaymentService(
	pool *pgxpool.Pool,
	paymentRepo *repository.ClientPaymentRepository,
	clientRepo *repository.ClientRepository,
	safeRepo *repository.SafeTransactionRepository,
) *ClientPaymentService {
	return &ClientPaymentService{
		pool: pool, paymentRepo: paymentRepo, clientRepo: clientRepo, safeRepo: safeRepo,
	}
}

func (s *ClientPaymentService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateClientPaymentRequest) (*model.ClientPayment, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify client exists.
	if _, err := s.clientRepo.GetByID(ctx, req.ClientID); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	cp := &model.ClientPayment{
		ID:            uuid.New(),
		ClientID:      req.ClientID,
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
		CreatedBy:     userID,
	}

	if err := s.paymentRepo.Create(ctx, tx, cp); err != nil {
		return nil, err
	}

	// Reduce client debt.
	if err := s.clientRepo.UpdateDebt(ctx, tx, req.ClientID, req.Amount.Neg()); err != nil {
		return nil, err
	}

	// Income to safe.
	desc := "Client debt payment"
	if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
		Type: "income", Source: "client_payment", BalanceType: req.PaymentMethod,
		Amount: req.Amount, Description: &desc, ReferenceID: &cp.ID,
	}); err != nil {
		return nil, err
	}

	return cp, tx.Commit(ctx)
}
```

- [ ] **Step 3: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/service/sale_return.go internal/service/client_payment.go && \
git commit -m "feat: add sale return and client payment services"
```

---

### Task 6: All Phase 3 Handlers

**Files:**
- Create: `familycotton-api/internal/handler/shift.go`
- Create: `familycotton-api/internal/handler/sale.go`
- Create: `familycotton-api/internal/handler/sale_return.go`
- Create: `familycotton-api/internal/handler/client_payment.go`

- [ ] **Step 1: Create shift handler**

Write `familycotton-api/internal/handler/shift.go`:
```go
package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/service"
)

type ShiftHandler struct {
	service *service.ShiftService
}

func NewShiftHandler(service *service.ShiftService) *ShiftHandler {
	return &ShiftHandler{service: service}
}

func (h *ShiftHandler) Open(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	shift, err := h.service.Open(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, shift)
}

func (h *ShiftHandler) Close(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	shift, err := h.service.Close(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, shift)
}

func (h *ShiftHandler) Current(w http.ResponseWriter, r *http.Request) {
	shift, err := h.service.GetCurrent(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, shift)
}

func (h *ShiftHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	shifts, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, shifts, page, limit, total)
}
```

- [ ] **Step 2: Create sale handler**

Write `familycotton-api/internal/handler/sale.go`:
```go
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type SaleHandler struct {
	service *service.SaleService
}

func NewSaleHandler(service *service.SaleService) *SaleHandler {
	return &SaleHandler{service: service}
}

func (h *SaleHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateSaleRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	sale, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, sale)
}

func (h *SaleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid sale id"))
		return
	}
	sale, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, sale)
}

func (h *SaleHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	var shiftID, clientID *uuid.UUID
	if sid := r.URL.Query().Get("shift_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			shiftID = &id
		}
	}
	if cid := r.URL.Query().Get("client_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			clientID = &id
		}
	}
	sales, total, err := h.service.List(r.Context(), shiftID, clientID, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, sales, page, limit, total)
}
```

- [ ] **Step 3: Create sale_return handler**

Write `familycotton-api/internal/handler/sale_return.go`:
```go
package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type SaleReturnHandler struct {
	service *service.SaleReturnService
}

func NewSaleReturnHandler(service *service.SaleReturnService) *SaleReturnHandler {
	return &SaleReturnHandler{service: service}
}

func (h *SaleReturnHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateSaleReturnRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	ret, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, ret)
}

func (h *SaleReturnHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	var saleID *uuid.UUID
	if sid := r.URL.Query().Get("sale_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			saleID = &id
		}
	}
	returns, total, err := h.service.List(r.Context(), saleID, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, returns, page, limit, total)
}
```

- [ ] **Step 4: Create client_payment handler**

Write `familycotton-api/internal/handler/client_payment.go`:
```go
package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type ClientPaymentHandler struct {
	service *service.ClientPaymentService
}

func NewClientPaymentHandler(service *service.ClientPaymentService) *ClientPaymentHandler {
	return &ClientPaymentHandler{service: service}
}

func (h *ClientPaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateClientPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	payment, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, payment)
}
```

- [ ] **Step 5: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/handler/shift.go internal/handler/sale.go internal/handler/sale_return.go internal/handler/client_payment.go && \
git commit -m "feat: add Phase 3 handlers (shift, sale, sale_return, client_payment)"
```

---

### Task 7: Wire Routes + DI

**Files:**
- Modify: `familycotton-api/internal/router/router.go`
- Modify: `familycotton-api/cmd/api/main.go`

- [ ] **Step 1: Update router.go**

Add these parameters to `router.New()`:
```go
shiftHandler *handler.ShiftHandler,
saleHandler *handler.SaleHandler,
saleReturnHandler *handler.SaleReturnHandler,
clientPaymentHandler *handler.ClientPaymentHandler,
```

Add these routes inside the protected group:
```go
// Shifts + Sales (employee + owner).
r.Post("/shifts/open", shiftHandler.Open)
r.Post("/shifts/close", shiftHandler.Close)
r.Get("/shifts/current", shiftHandler.Current)
r.Get("/shifts", shiftHandler.List)
r.Post("/sales", saleHandler.Create)
r.Get("/sales", saleHandler.List)
r.Get("/sales/{id}", saleHandler.GetByID)
r.Post("/sale-returns", saleReturnHandler.Create)
r.Get("/sale-returns", saleReturnHandler.List)
r.Post("/client-payments", clientPaymentHandler.Create)
```

- [ ] **Step 2: Update main.go DI**

Add after Phase 2 wiring:
```go
// Phase 3 repositories.
shiftRepo := repository.NewShiftRepository(pool)
saleRepo := repository.NewSaleRepository(pool)
saleReturnRepo := repository.NewSaleReturnRepository(pool)
clientPaymentRepo := repository.NewClientPaymentRepository(pool)
safeTransactionRepo := repository.NewSafeTransactionRepository()
ownerDebtRepo := repository.NewOwnerDebtRepository()

// Phase 3 services.
shiftService := service.NewShiftService(pool, shiftRepo, safeTransactionRepo, ownerDebtRepo)
saleService := service.NewSaleService(pool, saleRepo, shiftRepo, productRepo, clientRepo)
saleReturnService := service.NewSaleReturnService(pool, saleReturnRepo, saleRepo, productRepo, clientRepo, safeTransactionRepo)
clientPaymentService := service.NewClientPaymentService(pool, clientPaymentRepo, clientRepo, safeTransactionRepo)

// Phase 3 handlers.
shiftHandler := handler.NewShiftHandler(shiftService)
saleHandler := handler.NewSaleHandler(saleService)
saleReturnHandler := handler.NewSaleReturnHandler(saleReturnService)
clientPaymentHandler := handler.NewClientPaymentHandler(clientPaymentService)
```

Update `router.New(...)` to pass all Phase 3 handlers.

- [ ] **Step 3: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/router/router.go cmd/api/main.go && \
git commit -m "feat: wire Phase 3 routes and DI"
```

---

### Task 8: Integration Smoke Test

- [ ] **Step 1: Rebuild Docker**

```bash
cd familycotton-api && docker compose up -d --build
```

- [ ] **Step 2: Test shift open → sale → shift close → verify safe**

```bash
TOKEN=... # login as admin

# Open shift
curl -s -X POST http://localhost:8082/api/v1/shifts/open -H "Authorization: Bearer $TOKEN"

# Create product (need stock)
PRODUCT=$(curl -s -X POST http://localhost:8082/api/v1/products -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sku":"TEST-001","name":"Test Product","cost_price":"10000","sell_price":"15000"}')

# We need to add stock manually or via purchase (Phase 4). For smoke test, use SQL:
docker compose exec db psql -U familycotton -c "UPDATE products SET qty_shop = 10 WHERE sku = 'TEST-001'"

# Create sale
curl -s -X POST http://localhost:8082/api/v1/sales -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":"<PRODUCT_ID>","quantity":2,"unit_price":"15000"}],"paid_cash":"20000","paid_terminal":"10000","paid_online":"0","paid_debt":"0"}'

# Close shift
curl -s -X POST http://localhost:8082/api/v1/shifts/close -H "Authorization: Bearer $TOKEN"

# Verify safe_transactions were created
docker compose exec db psql -U familycotton -c "SELECT * FROM safe_transactions"
```

- [ ] **Step 3: Test client payment**

```bash
# Create client
CLIENT=$(curl -s -X POST http://localhost:8082/api/v1/clients -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" -d '{"name":"Test Client"}')

# Create sale with debt
# ... (use client_id, paid_debt > 0)

# Make payment
curl -s -X POST http://localhost:8082/api/v1/client-payments -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<CLIENT_ID>","amount":"5000","payment_method":"cash"}'
```

- [ ] **Step 4: Stop Docker**

```bash
docker compose down
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | DBTX interface | repository/db.go |
| 2 | Phase 3 models (6 files) | model/shift.go, sale.go, sale_return.go, client_payment.go, safe_transaction.go, owner_debt.go |
| 3 | Phase 3 repositories (6 new + 2 modified) | repository/shift.go, sale.go, sale_return.go, client_payment.go, safe_transaction.go, owner_debt.go + product.go, client.go |
| 4 | Shift + Sale services | service/shift.go, service/sale.go |
| 5 | Sale return + Client payment services | service/sale_return.go, service/client_payment.go |
| 6 | Phase 3 handlers (4 files) | handler/shift.go, sale.go, sale_return.go, client_payment.go |
| 7 | Wire routes + DI | router/router.go, cmd/api/main.go |
| 8 | Integration smoke test | manual |
