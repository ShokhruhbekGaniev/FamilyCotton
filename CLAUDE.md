# CLAUDE.md — FamilyCotton Backend

## Project Overview

FamilyCotton — REST API backend for a textile retail business. Manages sales, inventory, purchases, finances (safe/cash register), and analytics.

**Stack:** Go 1.25 | chi v5 | PostgreSQL 16 (pgx v5) | goose migrations | JWT auth | Docker Compose

**API runs on:** `http://localhost:8082/api/v1`
**DB port:** `5434` (mapped to 5432 inside container)

## Project Structure

```
familycotton-api/
├── cmd/api/main.go                  # Entry point, DI wiring, graceful shutdown
├── internal/
│   ├── config/config.go             # Env-based configuration
│   ├── model/                       # Data structs + request/response types (19 files)
│   ├── repository/                  # SQL queries via pgx (21 files, incl. db.go DBTX interface)
│   ├── service/                     # Business logic (19 files)
│   ├── handler/                     # HTTP handlers (20 files, incl. helpers.go)
│   ├── middleware/                   # auth.go, rbac.go, logging.go, cors.go
│   └── router/router.go            # All chi route definitions
├── migrations/
│   ├── migrations.go                # Embedded FS for goose
│   └── 001_init.sql                 # All 20 tables + indexes + seed data
├── docker-compose.yml               # api + postgres:16
├── Dockerfile                       # Multi-stage: golang:1.25-alpine → alpine:3.19
├── .env.example                     # Template for .env
└── go.mod
```

## Architecture

Classic layered: **handler → service → repository**. DI wired manually in `main.go`.

- **Models** (`internal/model/`): structs with JSON tags, `Validate()` on request types, pointers for optional/update fields, `json:"-"` for hidden fields (password, is_deleted)
- **Repositories** (`internal/repository/`): raw SQL via pgx. `RETURNING` for timestamps. `isDuplicateKey()` for unique violations. `is_deleted = false` filter on all reads. Pagination returns `(items, total, error)`.
- **Services** (`internal/service/`): business logic, validation, UUID generation. Complex ops use `pool.Begin(ctx)` + `defer tx.Rollback(ctx)` + `tx.Commit(ctx)`. `SELECT FOR UPDATE` on products for stock operations.
- **Handlers** (`internal/handler/`): thin HTTP layer. Use `decodeJSON()`, `respondSuccess()`, `respondError()`, `respondList()`, `paginationParams()` from `helpers.go`.
- **DBTX interface** (`repository/db.go`): satisfied by both `*pgxpool.Pool` and `pgx.Tx`, enabling repos to work in transactions.

## Key Patterns

- **Errors**: `AppError{Err: sentinel, Message: "user-facing"}`. Sentinels: `ErrNotFound`, `ErrValidation`, `ErrForbidden`, `ErrUnauthorized`, `ErrConflict`. Handler maps to HTTP codes.
- **Soft-delete**: `is_deleted` flag, never hard delete. Queries filter `WHERE is_deleted = false`.
- **Money**: `shopspring/decimal` in Go, `DECIMAL(15,2)` in PG. Never use float.
- **Margin**: computed `sell_price - cost_price`, never stored.
- **Pagination**: offset-based `?page=1&limit=20`. Response: `{data: [...], meta: {page, limit, total}}`.
- **RBAC**: `middleware.RequireRole("owner")` / `middleware.RequireRole("owner", "employee")`. Role from JWT claims.
- **Transactions**: services receive `*pgxpool.Pool`, call `pool.Begin(ctx)`, pass `tx` (DBTX) to repo methods.

## Roles

- **owner**: full access to everything
- **employee**: shifts, sales, returns, products (read+create), clients (read+create+update), suppliers (read only)
- Employee CANNOT: dashboard, safe, creditors, purchase orders, supplier payments, stock transfers, inventory checks, user management, product update/delete

## Running

```bash
cd familycotton-api
cp .env.example .env        # First time only
docker compose up -d --build
# API: http://localhost:8082/api/v1
# Login: admin / admin123
```

## Database

20 tables defined in `migrations/001_init.sql`. Key tables:
- `users`, `refresh_tokens` — auth
- `suppliers`, `creditors`, `clients`, `products` — catalogs
- `shifts`, `sales`, `sale_items`, `sale_returns` — sales operations
- `purchase_orders`, `purchase_order_items`, `supplier_payments` — procurement
- `creditor_transactions`, `client_payments` — payments
- `stock_transfers`, `inventory_checks`, `inventory_check_items` — stock
- `safe_transactions`, `owner_debts` — safe/finance

## API Documentation

Full endpoint reference: see `API.md` in project root.

## Important Rules

- Always keep `README.md` in the project root up to date when making changes
- Never use float for money — always `decimal.Decimal`
- All list endpoints must return paginated responses with `meta`
- New domains follow the same layered pattern: model → repository → service → handler → router → main.go
- Transactional operations must use `pool.Begin()` + `defer tx.Rollback()` + `tx.Commit()`
- Product stock changes must use `SELECT FOR UPDATE` to prevent races
- Soft-delete for entities referenced by foreign keys
- `created_by` (FK → users) on transactional tables for audit
