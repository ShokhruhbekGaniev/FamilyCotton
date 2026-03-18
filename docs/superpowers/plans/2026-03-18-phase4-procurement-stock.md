# Phase 4 — Procurement & Stock Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement purchase orders, supplier payments, creditor transactions, stock transfers, and inventory checks with auto-correction.

**Architecture:** Continues Phase 3 patterns — transactional services with DBTX, safe_transactions for monetary movements, debt tracking on suppliers/creditors. Inventory checks auto-generate items from current stock and auto-correct on completion.

**Tech Stack:** Go 1.25, chi v5, pgx v5, shopspring/decimal

**Spec:** `docs/superpowers/specs/2026-03-18-familycotton-backend-design.md` (Sections 6.4-6.7)

---

## File Map

```
familycotton-api/internal/
├── model/
│   ├── purchase_order.go      # NEW: PurchaseOrder, PurchaseOrderItem, CreatePurchaseOrderRequest
│   ├── supplier_payment.go    # NEW: SupplierPayment, CreateSupplierPaymentRequest
│   ├── creditor_transaction.go # NEW: CreditorTransaction, CreateCreditorTransactionRequest
│   ├── stock_transfer.go      # NEW: StockTransfer, CreateStockTransferRequest
│   └── inventory_check.go     # NEW: InventoryCheck, InventoryCheckItem, requests
├── repository/
│   ├── purchase_order.go      # NEW
│   ├── supplier_payment.go    # NEW
│   ├── creditor_transaction.go # NEW
│   ├── stock_transfer.go      # NEW
│   ├── inventory_check.go     # NEW
│   └── supplier.go            # MODIFY: add UpdateDebt method
├── service/
│   ├── purchase_order.go      # NEW: transactional purchase creation
│   ├── supplier_payment.go    # NEW: money + product return
│   ├── creditor_transaction.go # NEW: receive/repay with exchange rate
│   ├── stock_transfer.go      # NEW
│   └── inventory_check.go     # NEW: auto-generate items, auto-correct on completion
├── handler/
│   ├── purchase_order.go      # NEW
│   ├── supplier_payment.go    # NEW
│   ├── creditor_transaction.go # NEW
│   ├── stock_transfer.go      # NEW
│   └── inventory_check.go     # NEW
├── router/router.go           # MODIFY
cmd/api/main.go                # MODIFY
```

---

### Task 1: Models (5 files)

**Files:**
- Create: `internal/model/purchase_order.go`
- Create: `internal/model/supplier_payment.go`
- Create: `internal/model/creditor_transaction.go`
- Create: `internal/model/stock_transfer.go`
- Create: `internal/model/inventory_check.go`

- [ ] **Step 1: Create purchase_order model**

Write `internal/model/purchase_order.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PurchaseOrder struct {
	ID          uuid.UUID           `json:"id"`
	SupplierID  uuid.UUID           `json:"supplier_id"`
	TotalAmount decimal.Decimal     `json:"total_amount"`
	PaidAmount  decimal.Decimal     `json:"paid_amount"`
	Status      string              `json:"status"`
	CreatedBy   uuid.UUID           `json:"created_by"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Items       []PurchaseOrderItem `json:"items,omitempty"`
}

type PurchaseOrderItem struct {
	ID              uuid.UUID       `json:"id"`
	PurchaseOrderID uuid.UUID       `json:"purchase_order_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	Quantity        int             `json:"quantity"`
	UnitCost        decimal.Decimal `json:"unit_cost"`
	Destination     string          `json:"destination"`
}

type CreatePurchaseOrderItemRequest struct {
	ProductID   uuid.UUID       `json:"product_id"`
	Quantity    int             `json:"quantity"`
	UnitCost    decimal.Decimal `json:"unit_cost"`
	Destination string          `json:"destination"`
}

type CreatePurchaseOrderRequest struct {
	SupplierID uuid.UUID                        `json:"supplier_id"`
	Items      []CreatePurchaseOrderItemRequest  `json:"items"`
	PaidAmount decimal.Decimal                   `json:"paid_amount"`
}

func (r *CreatePurchaseOrderRequest) Validate() error {
	if len(r.Items) == 0 {
		return NewAppError(ErrValidation, "at least one item is required")
	}
	if r.PaidAmount.IsNegative() {
		return NewAppError(ErrValidation, "paid_amount cannot be negative")
	}
	for _, item := range r.Items {
		if item.Quantity <= 0 {
			return NewAppError(ErrValidation, "item quantity must be positive")
		}
		if item.UnitCost.IsNegative() {
			return NewAppError(ErrValidation, "unit_cost cannot be negative")
		}
		if item.Destination != "shop" && item.Destination != "warehouse" {
			return NewAppError(ErrValidation, "destination must be 'shop' or 'warehouse'")
		}
	}
	return nil
}
```

- [ ] **Step 2: Create supplier_payment model**

Write `internal/model/supplier_payment.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SupplierPayment struct {
	ID                uuid.UUID       `json:"id"`
	SupplierID        uuid.UUID       `json:"supplier_id"`
	PurchaseOrderID   *uuid.UUID      `json:"purchase_order_id"`
	PaymentType       string          `json:"payment_type"`
	Amount            decimal.Decimal `json:"amount"`
	ReturnedProductID *uuid.UUID      `json:"returned_product_id"`
	ReturnedQty       *int            `json:"returned_qty"`
	CreatedBy         uuid.UUID       `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
}

type CreateSupplierPaymentRequest struct {
	SupplierID        uuid.UUID       `json:"supplier_id"`
	PurchaseOrderID   *uuid.UUID      `json:"purchase_order_id"`
	PaymentType       string          `json:"payment_type"`
	Amount            decimal.Decimal `json:"amount"`
	ReturnedProductID *uuid.UUID      `json:"returned_product_id"`
	ReturnedQty       *int            `json:"returned_qty"`
}

func (r *CreateSupplierPaymentRequest) Validate() error {
	if r.PaymentType != "money" && r.PaymentType != "product_return" {
		return NewAppError(ErrValidation, "payment_type must be 'money' or 'product_return'")
	}
	if r.PaymentType == "money" && r.Amount.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "amount must be positive for money payment")
	}
	if r.PaymentType == "product_return" {
		if r.ReturnedProductID == nil || r.ReturnedQty == nil || *r.ReturnedQty <= 0 {
			return NewAppError(ErrValidation, "returned_product_id and returned_qty required for product_return")
		}
	}
	return nil
}
```

- [ ] **Step 3: Create creditor_transaction model**

Write `internal/model/creditor_transaction.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreditorTransaction struct {
	ID           uuid.UUID       `json:"id"`
	CreditorID   uuid.UUID       `json:"creditor_id"`
	Type         string          `json:"type"`
	Currency     string          `json:"currency"`
	Amount       decimal.Decimal `json:"amount"`
	ExchangeRate decimal.Decimal `json:"exchange_rate"`
	AmountUZS    decimal.Decimal `json:"amount_uzs"`
	CreatedBy    uuid.UUID       `json:"created_by"`
	CreatedAt    time.Time       `json:"created_at"`
}

type CreateCreditorTransactionRequest struct {
	CreditorID   uuid.UUID       `json:"creditor_id"`
	Type         string          `json:"type"`
	Currency     string          `json:"currency"`
	Amount       decimal.Decimal `json:"amount"`
	ExchangeRate decimal.Decimal `json:"exchange_rate"`
}

func (r *CreateCreditorTransactionRequest) Validate() error {
	if r.Type != "receive" && r.Type != "repay" {
		return NewAppError(ErrValidation, "type must be 'receive' or 'repay'")
	}
	if r.Currency != "UZS" && r.Currency != "USD" {
		return NewAppError(ErrValidation, "currency must be 'UZS' or 'USD'")
	}
	if r.Amount.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "amount must be positive")
	}
	if r.ExchangeRate.LessThanOrEqual(decimal.Zero) {
		return NewAppError(ErrValidation, "exchange_rate must be positive")
	}
	return nil
}
```

- [ ] **Step 4: Create stock_transfer model**

Write `internal/model/stock_transfer.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type StockTransfer struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	Direction string    `json:"direction"`
	Quantity  int       `json:"quantity"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateStockTransferRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Direction string    `json:"direction"`
	Quantity  int       `json:"quantity"`
}

func (r *CreateStockTransferRequest) Validate() error {
	if r.Direction != "warehouse_to_shop" && r.Direction != "shop_to_warehouse" {
		return NewAppError(ErrValidation, "direction must be 'warehouse_to_shop' or 'shop_to_warehouse'")
	}
	if r.Quantity <= 0 {
		return NewAppError(ErrValidation, "quantity must be positive")
	}
	return nil
}
```

- [ ] **Step 5: Create inventory_check model**

Write `internal/model/inventory_check.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type InventoryCheck struct {
	ID          uuid.UUID            `json:"id"`
	Location    string               `json:"location"`
	CheckedBy   uuid.UUID            `json:"checked_by"`
	Status      string               `json:"status"`
	CreatedAt   time.Time            `json:"created_at"`
	CompletedAt *time.Time           `json:"completed_at"`
	Items       []InventoryCheckItem `json:"items,omitempty"`
}

type InventoryCheckItem struct {
	ID               uuid.UUID `json:"id"`
	InventoryCheckID uuid.UUID `json:"inventory_check_id"`
	ProductID        uuid.UUID `json:"product_id"`
	ExpectedQty      int       `json:"expected_qty"`
	ActualQty        *int      `json:"actual_qty"`
	Difference       *int      `json:"difference"`
}

type CreateInventoryCheckRequest struct {
	Location string `json:"location"`
}

func (r *CreateInventoryCheckRequest) Validate() error {
	if r.Location != "shop" && r.Location != "warehouse" {
		return NewAppError(ErrValidation, "location must be 'shop' or 'warehouse'")
	}
	return nil
}

type UpdateInventoryCheckItemRequest struct {
	ItemID    uuid.UUID `json:"item_id"`
	ActualQty int       `json:"actual_qty"`
}

type UpdateInventoryCheckRequest struct {
	Items  []UpdateInventoryCheckItemRequest `json:"items"`
	Status *string                           `json:"status,omitempty"`
}
```

- [ ] **Step 6: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/model/purchase_order.go internal/model/supplier_payment.go \
  internal/model/creditor_transaction.go internal/model/stock_transfer.go \
  internal/model/inventory_check.go && \
git commit -m "feat: add Phase 4 models"
```

---

### Task 2: Repositories (5 new + 1 modified)

**Files:**
- Create: `internal/repository/purchase_order.go`
- Create: `internal/repository/supplier_payment.go`
- Create: `internal/repository/creditor_transaction.go`
- Create: `internal/repository/stock_transfer.go`
- Create: `internal/repository/inventory_check.go`
- Modify: `internal/repository/supplier.go` — add UpdateDebt

- [ ] **Step 1: Create purchase_order repository**

Write `internal/repository/purchase_order.go`:
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

type PurchaseOrderRepository struct {
	db *pgxpool.Pool
}

func NewPurchaseOrderRepository(db *pgxpool.Pool) *PurchaseOrderRepository {
	return &PurchaseOrderRepository{db: db}
}

func (r *PurchaseOrderRepository) Create(ctx context.Context, tx DBTX, po *model.PurchaseOrder) error {
	return tx.QueryRow(ctx,
		`INSERT INTO purchase_orders (id, supplier_id, total_amount, paid_amount, status, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at, updated_at`,
		po.ID, po.SupplierID, po.TotalAmount, po.PaidAmount, po.Status, po.CreatedBy,
	).Scan(&po.CreatedAt, &po.UpdatedAt)
}

func (r *PurchaseOrderRepository) CreateItem(ctx context.Context, tx DBTX, item *model.PurchaseOrderItem) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO purchase_order_items (id, purchase_order_id, product_id, quantity, unit_cost, destination)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		item.ID, item.PurchaseOrderID, item.ProductID, item.Quantity, item.UnitCost, item.Destination,
	)
	return err
}

func (r *PurchaseOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.PurchaseOrder, error) {
	po := &model.PurchaseOrder{}
	err := r.db.QueryRow(ctx,
		`SELECT id, supplier_id, total_amount, paid_amount, status, created_by, created_at, updated_at
		 FROM purchase_orders WHERE id = $1`, id,
	).Scan(&po.ID, &po.SupplierID, &po.TotalAmount, &po.PaidAmount, &po.Status,
		&po.CreatedBy, &po.CreatedAt, &po.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "purchase order not found")
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, purchase_order_id, product_id, quantity, unit_cost, destination
		 FROM purchase_order_items WHERE purchase_order_id = $1`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item model.PurchaseOrderItem
		if err := rows.Scan(&item.ID, &item.PurchaseOrderID, &item.ProductID,
			&item.Quantity, &item.UnitCost, &item.Destination); err != nil {
			return nil, err
		}
		po.Items = append(po.Items, item)
	}
	return po, rows.Err()
}

func (r *PurchaseOrderRepository) List(ctx context.Context, supplierID *uuid.UUID, status string, page, limit int) ([]model.PurchaseOrder, int, error) {
	where := ""
	var args []any
	idx := 1

	if supplierID != nil {
		where += fmt.Sprintf(" AND supplier_id = $%d", idx)
		args = append(args, *supplierID)
		idx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, status)
		idx++
	}

	var total int
	err := r.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM purchase_orders WHERE 1=1 %s", where), args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(
			`SELECT id, supplier_id, total_amount, paid_amount, status, created_by, created_at, updated_at
			 FROM purchase_orders WHERE 1=1 %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
			where, idx, idx+1,
		), args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []model.PurchaseOrder
	for rows.Next() {
		var po model.PurchaseOrder
		if err := rows.Scan(&po.ID, &po.SupplierID, &po.TotalAmount, &po.PaidAmount,
			&po.Status, &po.CreatedBy, &po.CreatedAt, &po.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, po)
	}
	return orders, total, rows.Err()
}

func (r *PurchaseOrderRepository) UpdatePaidAmount(ctx context.Context, tx DBTX, poID uuid.UUID, addAmount model.Decimal) error {
	_, err := tx.Exec(ctx,
		`UPDATE purchase_orders SET paid_amount = paid_amount + $1, updated_at = NOW() WHERE id = $2`,
		addAmount, poID,
	)
	return err
}

func (r *PurchaseOrderRepository) RecalcStatus(ctx context.Context, tx DBTX, poID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE purchase_orders SET status = CASE
		   WHEN paid_amount >= total_amount THEN 'paid'
		   WHEN paid_amount > 0 THEN 'partial'
		   ELSE 'unpaid'
		 END, updated_at = NOW()
		 WHERE id = $1`, poID,
	)
	return err
}
```

**Note:** `model.Decimal` should be `decimal.Decimal` — add `"github.com/shopspring/decimal"` import and fix the type. The `UpdatePaidAmount` parameter should be `addAmount decimal.Decimal`.

- [ ] **Step 2: Create supplier_payment repository**

Write `internal/repository/supplier_payment.go`:
```go
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type SupplierPaymentRepository struct {
	db *pgxpool.Pool
}

func NewSupplierPaymentRepository(db *pgxpool.Pool) *SupplierPaymentRepository {
	return &SupplierPaymentRepository{db: db}
}

func (r *SupplierPaymentRepository) Create(ctx context.Context, tx DBTX, sp *model.SupplierPayment) error {
	return tx.QueryRow(ctx,
		`INSERT INTO supplier_payments (id, supplier_id, purchase_order_id, payment_type, amount, returned_product_id, returned_qty, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at`,
		sp.ID, sp.SupplierID, sp.PurchaseOrderID, sp.PaymentType, sp.Amount,
		sp.ReturnedProductID, sp.ReturnedQty, sp.CreatedBy,
	).Scan(&sp.CreatedAt)
}
```

- [ ] **Step 3: Create creditor_transaction repository**

Write `internal/repository/creditor_transaction.go`:
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

type CreditorTransactionRepository struct {
	db *pgxpool.Pool
}

func NewCreditorTransactionRepository(db *pgxpool.Pool) *CreditorTransactionRepository {
	return &CreditorTransactionRepository{db: db}
}

func (r *CreditorTransactionRepository) Create(ctx context.Context, tx DBTX, ct *model.CreditorTransaction) error {
	return tx.QueryRow(ctx,
		`INSERT INTO creditor_transactions (id, creditor_id, type, currency, amount, exchange_rate, amount_uzs, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at`,
		ct.ID, ct.CreditorID, ct.Type, ct.Currency, ct.Amount, ct.ExchangeRate, ct.AmountUZS, ct.CreatedBy,
	).Scan(&ct.CreatedAt)
}

func (r *CreditorTransactionRepository) ListByCreditor(ctx context.Context, creditorID uuid.UUID, page, limit int) ([]model.CreditorTransaction, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM creditor_transactions WHERE creditor_id = $1`, creditorID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, creditor_id, type, currency, amount, exchange_rate, amount_uzs, created_by, created_at
		 FROM creditor_transactions WHERE creditor_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		creditorID, limit, (page-1)*limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txns []model.CreditorTransaction
	for rows.Next() {
		var ct model.CreditorTransaction
		if err := rows.Scan(&ct.ID, &ct.CreditorID, &ct.Type, &ct.Currency,
			&ct.Amount, &ct.ExchangeRate, &ct.AmountUZS, &ct.CreatedBy, &ct.CreatedAt); err != nil {
			return nil, 0, err
		}
		txns = append(txns, ct)
	}
	return txns, total, rows.Err()
}

// Unused import guards.
var _ = fmt.Sprintf
var _ = errors.Is
var _ = pgx.ErrNoRows
```

- [ ] **Step 4: Create stock_transfer repository**

Write `internal/repository/stock_transfer.go`:
```go
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type StockTransferRepository struct {
	db *pgxpool.Pool
}

func NewStockTransferRepository(db *pgxpool.Pool) *StockTransferRepository {
	return &StockTransferRepository{db: db}
}

func (r *StockTransferRepository) Create(ctx context.Context, tx DBTX, st *model.StockTransfer) error {
	return tx.QueryRow(ctx,
		`INSERT INTO stock_transfers (id, product_id, direction, quantity, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		st.ID, st.ProductID, st.Direction, st.Quantity, st.CreatedBy,
	).Scan(&st.CreatedAt)
}
```

- [ ] **Step 5: Create inventory_check repository**

Write `internal/repository/inventory_check.go`:
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

type InventoryCheckRepository struct {
	db *pgxpool.Pool
}

func NewInventoryCheckRepository(db *pgxpool.Pool) *InventoryCheckRepository {
	return &InventoryCheckRepository{db: db}
}

func (r *InventoryCheckRepository) Create(ctx context.Context, ic *model.InventoryCheck) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO inventory_checks (id, location, checked_by)
		 VALUES ($1, $2, $3)
		 RETURNING status, created_at`,
		ic.ID, ic.Location, ic.CheckedBy,
	).Scan(&ic.Status, &ic.CreatedAt)
}

func (r *InventoryCheckRepository) CreateItem(ctx context.Context, tx DBTX, item *model.InventoryCheckItem) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO inventory_check_items (id, inventory_check_id, product_id, expected_qty)
		 VALUES ($1, $2, $3, $4)`,
		item.ID, item.InventoryCheckID, item.ProductID, item.ExpectedQty,
	)
	return err
}

func (r *InventoryCheckRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.InventoryCheck, error) {
	ic := &model.InventoryCheck{}
	err := r.db.QueryRow(ctx,
		`SELECT id, location, checked_by, status, created_at, completed_at
		 FROM inventory_checks WHERE id = $1`, id,
	).Scan(&ic.ID, &ic.Location, &ic.CheckedBy, &ic.Status, &ic.CreatedAt, &ic.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "inventory check not found")
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, inventory_check_id, product_id, expected_qty, actual_qty, difference
		 FROM inventory_check_items WHERE inventory_check_id = $1`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item model.InventoryCheckItem
		if err := rows.Scan(&item.ID, &item.InventoryCheckID, &item.ProductID,
			&item.ExpectedQty, &item.ActualQty, &item.Difference); err != nil {
			return nil, err
		}
		ic.Items = append(ic.Items, item)
	}
	return ic, rows.Err()
}

func (r *InventoryCheckRepository) UpdateItemActualQty(ctx context.Context, tx DBTX, itemID uuid.UUID, actualQty int) error {
	diff := actualQty // Will be computed in SQL.
	_, err := tx.Exec(ctx,
		`UPDATE inventory_check_items SET actual_qty = $1, difference = $1 - expected_qty
		 WHERE id = $2`,
		actualQty, itemID,
	)
	_ = diff
	return err
}

func (r *InventoryCheckRepository) Complete(ctx context.Context, tx DBTX, id uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE inventory_checks SET status = 'completed', completed_at = NOW()
		 WHERE id = $1 AND status = 'in_progress'`,
		id,
	)
	return err
}

func (r *InventoryCheckRepository) GetItemsByCheckID(ctx context.Context, checkID uuid.UUID) ([]model.InventoryCheckItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, inventory_check_id, product_id, expected_qty, actual_qty, difference
		 FROM inventory_check_items WHERE inventory_check_id = $1`, checkID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.InventoryCheckItem
	for rows.Next() {
		var item model.InventoryCheckItem
		if err := rows.Scan(&item.ID, &item.InventoryCheckID, &item.ProductID,
			&item.ExpectedQty, &item.ActualQty, &item.Difference); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
```

- [ ] **Step 6: Add UpdateDebt to supplier repository**

Add to `internal/repository/supplier.go`:
```go
func (r *SupplierRepository) UpdateDebt(ctx context.Context, tx DBTX, supplierID uuid.UUID, delta decimal.Decimal) error {
	_, err := tx.Exec(ctx,
		`UPDATE suppliers SET total_debt = total_debt + $1, updated_at = NOW()
		 WHERE id = $2 AND is_deleted = false`,
		delta, supplierID,
	)
	return err
}
```

Add `"github.com/shopspring/decimal"` to supplier.go imports.

- [ ] **Step 7: Add UpdateDebt to creditor repository**

Add to `internal/repository/creditor.go`:
```go
func (r *CreditorRepository) UpdateDebt(ctx context.Context, tx DBTX, creditorID uuid.UUID, delta decimal.Decimal) error {
	_, err := tx.Exec(ctx,
		`UPDATE creditors SET total_debt = total_debt + $1, updated_at = NOW()
		 WHERE id = $2 AND is_deleted = false`,
		delta, creditorID,
	)
	return err
}
```

Add `"github.com/shopspring/decimal"` to creditor.go imports.

- [ ] **Step 8: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/repository/ && \
git commit -m "feat: add Phase 4 repositories"
```

---

### Task 3: Services (5 files)

**Files:**
- Create: `internal/service/purchase_order.go`
- Create: `internal/service/supplier_payment.go`
- Create: `internal/service/creditor_transaction.go`
- Create: `internal/service/stock_transfer.go`
- Create: `internal/service/inventory_check.go`

- [ ] **Step 1: Create purchase_order service**

Write `internal/service/purchase_order.go`:
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
	return &PurchaseOrderService{pool: pool, poRepo: poRepo, productRepo: productRepo, supplierRepo: supplierRepo, safeRepo: safeRepo}
}

func (s *PurchaseOrderService) Create(ctx context.Context, userID uuid.UUID, req *model.CreatePurchaseOrderRequest) (*model.PurchaseOrder, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify supplier exists.
	if _, err := s.supplierRepo.GetByID(ctx, req.SupplierID); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Compute total.
	totalAmount := decimal.Zero
	var items []model.PurchaseOrderItem
	for _, ri := range req.Items {
		subtotal := ri.UnitCost.Mul(decimal.NewFromInt(int64(ri.Quantity)))
		totalAmount = totalAmount.Add(subtotal)
		items = append(items, model.PurchaseOrderItem{
			ID:        uuid.New(),
			ProductID: ri.ProductID,
			Quantity:  ri.Quantity,
			UnitCost:  ri.UnitCost,
			Destination: ri.Destination,
		})
	}

	if req.PaidAmount.GreaterThan(totalAmount) {
		return nil, model.NewAppError(model.ErrValidation, "paid_amount cannot exceed total_amount")
	}

	// Determine status.
	status := "unpaid"
	if req.PaidAmount.Equal(totalAmount) {
		status = "paid"
	} else if req.PaidAmount.IsPositive() {
		status = "partial"
	}

	po := &model.PurchaseOrder{
		ID:          uuid.New(),
		SupplierID:  req.SupplierID,
		TotalAmount: totalAmount,
		PaidAmount:  req.PaidAmount,
		Status:      status,
		CreatedBy:   userID,
	}

	if err := s.poRepo.Create(ctx, tx, po); err != nil {
		return nil, err
	}

	// Create items + stock arrival.
	for i := range items {
		items[i].PurchaseOrderID = po.ID
		if err := s.poRepo.CreateItem(ctx, tx, &items[i]); err != nil {
			return nil, err
		}

		// Increment stock.
		product, err := s.productRepo.GetByIDForUpdate(ctx, tx, items[i].ProductID)
		if err != nil {
			return nil, err
		}
		newShop := product.QtyShop
		newWarehouse := product.QtyWarehouse
		if items[i].Destination == "shop" {
			newShop += items[i].Quantity
		} else {
			newWarehouse += items[i].Quantity
		}
		if err := s.productRepo.UpdateStock(ctx, tx, product.ID, newShop, newWarehouse); err != nil {
			return nil, err
		}
	}

	// Safe transactions for paid amount.
	if req.PaidAmount.IsPositive() {
		desc := "Purchase order payment"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "expense", Source: "purchase_cash", BalanceType: "cash",
			Amount: req.PaidAmount, Description: &desc, ReferenceID: &po.ID,
		}); err != nil {
			return nil, err
		}
	}

	// Supplier debt for unpaid portion.
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
```

- [ ] **Step 2: Create supplier_payment service**

Write `internal/service/supplier_payment.go`:
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

type SupplierPaymentService struct {
	pool         *pgxpool.Pool
	spRepo       *repository.SupplierPaymentRepository
	supplierRepo *repository.SupplierRepository
	productRepo  *repository.ProductRepository
	poRepo       *repository.PurchaseOrderRepository
	safeRepo     *repository.SafeTransactionRepository
}

func NewSupplierPaymentService(
	pool *pgxpool.Pool,
	spRepo *repository.SupplierPaymentRepository,
	supplierRepo *repository.SupplierRepository,
	productRepo *repository.ProductRepository,
	poRepo *repository.PurchaseOrderRepository,
	safeRepo *repository.SafeTransactionRepository,
) *SupplierPaymentService {
	return &SupplierPaymentService{pool: pool, spRepo: spRepo, supplierRepo: supplierRepo, productRepo: productRepo, poRepo: poRepo, safeRepo: safeRepo}
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
		ID:              uuid.New(),
		SupplierID:      req.SupplierID,
		PurchaseOrderID: req.PurchaseOrderID,
		PaymentType:     req.PaymentType,
		CreatedBy:       userID,
	}

	switch req.PaymentType {
	case "money":
		sp.Amount = req.Amount

		// Deduct from safe.
		desc := "Supplier payment"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "expense", Source: "supplier_payment", BalanceType: "cash",
			Amount: req.Amount, Description: &desc, ReferenceID: &sp.ID,
		}); err != nil {
			return nil, err
		}

		// Reduce supplier debt.
		if err := s.supplierRepo.UpdateDebt(ctx, tx, req.SupplierID, req.Amount.Neg()); err != nil {
			return nil, err
		}

		// Update purchase order if linked.
		if req.PurchaseOrderID != nil {
			if err := s.poRepo.UpdatePaidAmount(ctx, tx, *req.PurchaseOrderID, req.Amount); err != nil {
				return nil, err
			}
			if err := s.poRepo.RecalcStatus(ctx, tx, *req.PurchaseOrderID); err != nil {
				return nil, err
			}
		}

	case "product_return":
		sp.ReturnedProductID = req.ReturnedProductID
		sp.ReturnedQty = req.ReturnedQty

		// Lock and deduct product stock.
		product, err := s.productRepo.GetByIDForUpdate(ctx, tx, *req.ReturnedProductID)
		if err != nil {
			return nil, err
		}
		if product.QtyShop < *req.ReturnedQty {
			return nil, model.NewAppError(model.ErrValidation, "insufficient stock for product return")
		}
		if err := s.productRepo.UpdateStock(ctx, tx, product.ID, product.QtyShop-*req.ReturnedQty, product.QtyWarehouse); err != nil {
			return nil, err
		}

		// Calculate return value = cost_price * qty.
		returnValue := product.CostPrice.Mul(decimal.NewFromInt(int64(*req.ReturnedQty)))
		sp.Amount = returnValue

		// Reduce supplier debt.
		if err := s.supplierRepo.UpdateDebt(ctx, tx, req.SupplierID, returnValue.Neg()); err != nil {
			return nil, err
		}

		// Update purchase order if linked.
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
```

- [ ] **Step 3: Create creditor_transaction service**

Write `internal/service/creditor_transaction.go`:
```go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type CreditorTransactionService struct {
	pool         *pgxpool.Pool
	ctRepo       *repository.CreditorTransactionRepository
	creditorRepo *repository.CreditorRepository
	safeRepo     *repository.SafeTransactionRepository
}

func NewCreditorTransactionService(
	pool *pgxpool.Pool,
	ctRepo *repository.CreditorTransactionRepository,
	creditorRepo *repository.CreditorRepository,
	safeRepo *repository.SafeTransactionRepository,
) *CreditorTransactionService {
	return &CreditorTransactionService{pool: pool, ctRepo: ctRepo, creditorRepo: creditorRepo, safeRepo: safeRepo}
}

func (s *CreditorTransactionService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateCreditorTransactionRequest) (*model.CreditorTransaction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Verify creditor exists.
	if _, err := s.creditorRepo.GetByID(ctx, req.CreditorID); err != nil {
		return nil, err
	}

	amountUZS := req.Amount.Mul(req.ExchangeRate)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	ct := &model.CreditorTransaction{
		ID:           uuid.New(),
		CreditorID:   req.CreditorID,
		Type:         req.Type,
		Currency:     req.Currency,
		Amount:       req.Amount,
		ExchangeRate: req.ExchangeRate,
		AmountUZS:    amountUZS,
		CreatedBy:    userID,
	}

	if err := s.ctRepo.Create(ctx, tx, ct); err != nil {
		return nil, err
	}

	switch req.Type {
	case "receive":
		// Add to safe (cash), increment creditor debt.
		desc := "Creditor receive"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "income", Source: "creditor_receive", BalanceType: "cash",
			Amount: amountUZS, Description: &desc, ReferenceID: &ct.ID,
		}); err != nil {
			return nil, err
		}
		if err := s.creditorRepo.UpdateDebt(ctx, tx, req.CreditorID, amountUZS); err != nil {
			return nil, err
		}

	case "repay":
		// Deduct from safe (cash), reduce creditor debt.
		desc := "Creditor repay"
		if err := s.safeRepo.Create(ctx, tx, &model.SafeTransaction{
			Type: "expense", Source: "creditor_repay", BalanceType: "cash",
			Amount: amountUZS, Description: &desc, ReferenceID: &ct.ID,
		}); err != nil {
			return nil, err
		}
		if err := s.creditorRepo.UpdateDebt(ctx, tx, req.CreditorID, amountUZS.Neg()); err != nil {
			return nil, err
		}
	}

	return ct, tx.Commit(ctx)
}

func (s *CreditorTransactionService) ListByCreditor(ctx context.Context, creditorID uuid.UUID, page, limit int) ([]model.CreditorTransaction, int, error) {
	return s.ctRepo.ListByCreditor(ctx, creditorID, page, limit)
}
```

- [ ] **Step 4: Create stock_transfer service**

Write `internal/service/stock_transfer.go`:
```go
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

func NewStockTransferService(pool *pgxpool.Pool, stRepo *repository.StockTransferRepository, productRepo *repository.ProductRepository) *StockTransferService {
	return &StockTransferService{pool: pool, stRepo: stRepo, productRepo: productRepo}
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

	product, err := s.productRepo.GetByIDForUpdate(ctx, tx, req.ProductID)
	if err != nil {
		return nil, err
	}

	newShop := product.QtyShop
	newWarehouse := product.QtyWarehouse

	switch req.Direction {
	case "warehouse_to_shop":
		if product.QtyWarehouse < req.Quantity {
			return nil, model.NewAppError(model.ErrValidation, "insufficient warehouse stock")
		}
		newWarehouse -= req.Quantity
		newShop += req.Quantity
	case "shop_to_warehouse":
		if product.QtyShop < req.Quantity {
			return nil, model.NewAppError(model.ErrValidation, "insufficient shop stock")
		}
		newShop -= req.Quantity
		newWarehouse += req.Quantity
	}

	if err := s.productRepo.UpdateStock(ctx, tx, product.ID, newShop, newWarehouse); err != nil {
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
```

- [ ] **Step 5: Create inventory_check service**

Write `internal/service/inventory_check.go`:
```go
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

func NewInventoryCheckService(pool *pgxpool.Pool, icRepo *repository.InventoryCheckRepository, productRepo *repository.ProductRepository) *InventoryCheckService {
	return &InventoryCheckService{pool: pool, icRepo: icRepo, productRepo: productRepo}
}

func (s *InventoryCheckService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateInventoryCheckRequest) (*model.InventoryCheck, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	ic := &model.InventoryCheck{
		ID:        uuid.New(),
		Location:  req.Location,
		CheckedBy: userID,
	}

	if err := s.icRepo.Create(ctx, ic); err != nil {
		return nil, err
	}

	// Auto-generate items from all non-deleted products.
	products, _, err := s.productRepo.List(ctx, model.ProductFilter{}, 1, 10000)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	for _, p := range products {
		expectedQty := p.QtyShop
		if req.Location == "warehouse" {
			expectedQty = p.QtyWarehouse
		}
		item := &model.InventoryCheckItem{
			ID:               uuid.New(),
			InventoryCheckID: ic.ID,
			ProductID:        p.ID,
			ExpectedQty:      expectedQty,
		}
		if err := s.icRepo.CreateItem(ctx, tx, item); err != nil {
			return nil, err
		}
		ic.Items = append(ic.Items, *item)
	}

	return ic, tx.Commit(ctx)
}

func (s *InventoryCheckService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateInventoryCheckRequest) (*model.InventoryCheck, error) {
	ic, err := s.icRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ic.Status != "in_progress" {
		return nil, model.NewAppError(model.ErrValidation, "inventory check is already completed")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Update item quantities.
	for _, itemReq := range req.Items {
		if err := s.icRepo.UpdateItemActualQty(ctx, tx, itemReq.ItemID, itemReq.ActualQty); err != nil {
			return nil, err
		}
	}

	// If status is set to completed, auto-correct stock.
	if req.Status != nil && *req.Status == "completed" {
		// Re-read items to get updated actual_qty values.
		items, err := s.icRepo.GetItemsByCheckID(ctx, id)
		if err != nil {
			return nil, err
		}

		// Verify all items have actual_qty filled.
		for _, item := range items {
			if item.ActualQty == nil {
				return nil, model.NewAppError(model.ErrValidation, "all items must have actual_qty before completing")
			}
		}

		// Auto-correct stock.
		for _, item := range items {
			product, err := s.productRepo.GetByIDForUpdate(ctx, tx, item.ProductID)
			if err != nil {
				return nil, err
			}
			newShop := product.QtyShop
			newWarehouse := product.QtyWarehouse
			if ic.Location == "shop" {
				newShop = *item.ActualQty
			} else {
				newWarehouse = *item.ActualQty
			}
			if err := s.productRepo.UpdateStock(ctx, tx, product.ID, newShop, newWarehouse); err != nil {
				return nil, err
			}
		}

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
```

- [ ] **Step 6: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/service/ && \
git commit -m "feat: add Phase 4 services (purchase orders, supplier payments, creditor transactions, stock transfers, inventory checks)"
```

---

### Task 4: Handlers (5 files)

**Files:**
- Create: `internal/handler/purchase_order.go`
- Create: `internal/handler/supplier_payment.go`
- Create: `internal/handler/creditor_transaction.go`
- Create: `internal/handler/stock_transfer.go`
- Create: `internal/handler/inventory_check.go`

All handlers follow the established pattern. Each handler:
- Receives its service in constructor
- Uses `middleware.GetUserID(r.Context())` for userID
- Uses `decodeJSON`, `respondSuccess`, `respondError`, `respondList`, `paginationParams`
- Parses URL params with `chi.URLParam` and query params for filters

- [ ] **Step 1: Create purchase_order handler**

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

type PurchaseOrderHandler struct {
	service *service.PurchaseOrderService
}

func NewPurchaseOrderHandler(service *service.PurchaseOrderService) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{service: service}
}

func (h *PurchaseOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreatePurchaseOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	po, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, po)
}

func (h *PurchaseOrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid purchase order id"))
		return
	}
	po, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, po)
}

func (h *PurchaseOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	var supplierID *uuid.UUID
	if sid := r.URL.Query().Get("supplier_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			supplierID = &id
		}
	}
	status := r.URL.Query().Get("status")
	orders, total, err := h.service.List(r.Context(), supplierID, status, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, orders, page, limit, total)
}
```

- [ ] **Step 2: Create supplier_payment handler**

```go
package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type SupplierPaymentHandler struct {
	service *service.SupplierPaymentService
}

func NewSupplierPaymentHandler(service *service.SupplierPaymentService) *SupplierPaymentHandler {
	return &SupplierPaymentHandler{service: service}
}

func (h *SupplierPaymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateSupplierPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	sp, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, sp)
}
```

- [ ] **Step 3: Create creditor_transaction handler**

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

type CreditorTransactionHandler struct {
	service *service.CreditorTransactionService
}

func NewCreditorTransactionHandler(service *service.CreditorTransactionService) *CreditorTransactionHandler {
	return &CreditorTransactionHandler{service: service}
}

func (h *CreditorTransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateCreditorTransactionRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	ct, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, ct)
}
```

Note: The creditor detail endpoint (`GET /creditors/:id`) already exists from Phase 2. Creditor transaction history can be accessed via the creditor transaction service. If needed, add a list endpoint for creditor transactions here. For now, add it:

```go
func (h *CreditorTransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	creditorID, err := uuid.Parse(chi.URLParam(r, "creditorId"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid creditor id"))
		return
	}
	txns, total, err := h.service.ListByCreditor(r.Context(), creditorID, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, txns, page, limit, total)
}
```

- [ ] **Step 4: Create stock_transfer handler**

```go
package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type StockTransferHandler struct {
	service *service.StockTransferService
}

func NewStockTransferHandler(service *service.StockTransferService) *StockTransferHandler {
	return &StockTransferHandler{service: service}
}

func (h *StockTransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateStockTransferRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	st, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, st)
}
```

- [ ] **Step 5: Create inventory_check handler**

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

type InventoryCheckHandler struct {
	service *service.InventoryCheckService
}

func NewInventoryCheckHandler(service *service.InventoryCheckService) *InventoryCheckHandler {
	return &InventoryCheckHandler{service: service}
}

func (h *InventoryCheckHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req model.CreateInventoryCheckRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	ic, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, ic)
}

func (h *InventoryCheckHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid inventory check id"))
		return
	}
	var req model.UpdateInventoryCheckRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	ic, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, ic)
}
```

- [ ] **Step 6: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/handler/ && \
git commit -m "feat: add Phase 4 handlers"
```

---

### Task 5: Wire Routes + DI

**Files:**
- Modify: `internal/router/router.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Update router**

Add 5 new handler params to `router.New()`. Add these routes inside the protected group:

```go
// Purchase Orders (owner only).
r.Route("/purchase-orders", func(r chi.Router) {
	r.Use(middleware.RequireRole("owner"))
	r.Get("/", purchaseOrderHandler.List)
	r.Get("/{id}", purchaseOrderHandler.GetByID)
	r.Post("/", purchaseOrderHandler.Create)
})

// Supplier Payments (owner only).
r.Group(func(r chi.Router) {
	r.Use(middleware.RequireRole("owner"))
	r.Post("/supplier-payments", supplierPaymentHandler.Create)
})

// Creditor Transactions (owner only).
r.Group(func(r chi.Router) {
	r.Use(middleware.RequireRole("owner"))
	r.Post("/creditor-transactions", creditorTransactionHandler.Create)
})

// Stock (owner only).
r.Group(func(r chi.Router) {
	r.Use(middleware.RequireRole("owner"))
	r.Post("/stock/transfer", stockTransferHandler.Create)
	r.Post("/inventory-checks", inventoryCheckHandler.Create)
	r.Put("/inventory-checks/{id}", inventoryCheckHandler.Update)
})
```

- [ ] **Step 2: Update main.go DI**

Add Phase 4 repositories, services, handlers. Update `router.New(...)` call.

```go
// Phase 4 repositories.
purchaseOrderRepo := repository.NewPurchaseOrderRepository(pool)
supplierPaymentRepo := repository.NewSupplierPaymentRepository(pool)
creditorTransactionRepo := repository.NewCreditorTransactionRepository(pool)
stockTransferRepo := repository.NewStockTransferRepository(pool)
inventoryCheckRepo := repository.NewInventoryCheckRepository(pool)

// Phase 4 services.
purchaseOrderService := service.NewPurchaseOrderService(pool, purchaseOrderRepo, productRepo, supplierRepo, safeTransactionRepo)
supplierPaymentService := service.NewSupplierPaymentService(pool, supplierPaymentRepo, supplierRepo, productRepo, purchaseOrderRepo, safeTransactionRepo)
creditorTransactionService := service.NewCreditorTransactionService(pool, creditorTransactionRepo, creditorRepo, safeTransactionRepo)
stockTransferService := service.NewStockTransferService(pool, stockTransferRepo, productRepo)
inventoryCheckService := service.NewInventoryCheckService(pool, inventoryCheckRepo, productRepo)

// Phase 4 handlers.
purchaseOrderHandler := handler.NewPurchaseOrderHandler(purchaseOrderService)
supplierPaymentHandler := handler.NewSupplierPaymentHandler(supplierPaymentService)
creditorTransactionHandler := handler.NewCreditorTransactionHandler(creditorTransactionService)
stockTransferHandler := handler.NewStockTransferHandler(stockTransferService)
inventoryCheckHandler := handler.NewInventoryCheckHandler(inventoryCheckService)
```

- [ ] **Step 3: Verify and commit**

```bash
cd familycotton-api && go build ./... && \
git add internal/router/router.go cmd/api/main.go && \
git commit -m "feat: wire Phase 4 routes and DI"
```

---

### Task 6: Integration Smoke Test

- [ ] **Step 1: Rebuild Docker**

```bash
cd familycotton-api && docker compose up -d --build
```

- [ ] **Step 2: Test purchase order flow**

```bash
TOKEN=... # login as admin
# Create purchase order with items
# Verify stock increases
# Verify supplier debt
```

- [ ] **Step 3: Test stock transfer**

```bash
# Transfer warehouse → shop
# Verify stock quantities
```

- [ ] **Step 4: Test creditor transaction**

```bash
# Receive money from creditor (USD with exchange rate)
# Verify creditor debt, safe transaction
```

- [ ] **Step 5: Stop Docker**

```bash
docker compose down
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Phase 4 models (5 files) | model/purchase_order.go, supplier_payment.go, creditor_transaction.go, stock_transfer.go, inventory_check.go |
| 2 | Phase 4 repositories (5 new + 2 modified) | repository/* + supplier.go, creditor.go updates |
| 3 | Phase 4 services (5 files) | service/purchase_order.go, supplier_payment.go, creditor_transaction.go, stock_transfer.go, inventory_check.go |
| 4 | Phase 4 handlers (5 files) | handler/purchase_order.go, supplier_payment.go, creditor_transaction.go, stock_transfer.go, inventory_check.go |
| 5 | Wire routes + DI | router/router.go, cmd/api/main.go |
| 6 | Integration smoke test | manual |
