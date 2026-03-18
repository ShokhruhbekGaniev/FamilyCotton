# Phase 1 — Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Set up the project foundation — scaffolding, Docker, database migrations, JWT auth, users CRUD, middleware, unified error handling.

**Architecture:** Classic layered (handler → service → repository). DI wired in main.go. PostgreSQL via pgx v5. Chi v5 router. JWT access (1h) + refresh (30d) tokens with DB-backed refresh token storage.

**Tech Stack:** Go 1.22+, chi v5, pgx v5, pgconn, goose, bcrypt, golang-jwt/jwt/v5, slog (shopspring/decimal introduced in Phase 2)

**Spec:** `docs/superpowers/specs/2026-03-18-familycotton-backend-design.md`

---

## File Map

```
familycotton-api/
├── cmd/api/main.go                    # Entry point, DI wiring, server start
├── internal/
│   ├── config/config.go               # Env config struct + loader
│   ├── model/
│   │   ├── user.go                    # User struct, CreateUserReq, UpdateUserReq
│   │   ├── auth.go                    # LoginReq, TokenPair, RefreshReq
│   │   └── response.go               # SuccessResponse, ErrorResponse, Meta
│   ├── repository/
│   │   ├── user.go                    # UserRepository (CRUD + soft-delete)
│   │   └── token.go                   # TokenRepository (refresh tokens)
│   ├── service/
│   │   ├── auth.go                    # AuthService (login, refresh, logout)
│   │   └── user.go                    # UserService (CRUD + password hashing)
│   ├── handler/
│   │   ├── auth.go                    # Auth handlers (login, refresh, logout, me)
│   │   ├── user.go                    # User handlers (CRUD)
│   │   └── helpers.go                 # JSON response helpers, error mapping
│   ├── middleware/
│   │   ├── auth.go                    # JWT validation, context user
│   │   ├── rbac.go                    # RequireRole middleware
│   │   ├── logging.go                 # Request logging (slog)
│   │   └── cors.go                    # CORS headers
│   └── router/router.go              # Chi route definitions
├── migrations/
│   ├── migrations.go                  # Embedded FS for goose (//go:embed *.sql)
│   └── 001_init.sql                   # All 20 tables + indexes
├── docker-compose.yml
├── Dockerfile
├── .env.example
├── .gitignore
└── go.mod
```

---

### Task 1: Project Scaffolding + Go Module

**Files:**
- Create: `familycotton-api/go.mod`
- Create: `familycotton-api/cmd/api/main.go` (stub)
- Create: `familycotton-api/.gitignore`
- Create: `familycotton-api/.env.example`

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p familycotton-api/cmd/api
mkdir -p familycotton-api/internal/{config,model,repository,service,handler,middleware,router}
mkdir -p familycotton-api/migrations
```

- [ ] **Step 2: Initialize Go module**

```bash
cd familycotton-api
go mod init github.com/familycotton/api
```

- [ ] **Step 3: Create .gitignore**

Write `familycotton-api/.gitignore`:
```gitignore
# Binary
/api
/tmp

# Environment
.env

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
```

- [ ] **Step 4: Create .env.example**

Write `familycotton-api/.env.example`:
```env
DB_HOST=db
DB_PORT=5432
DB_USER=familycotton
DB_PASSWORD=familycotton_secret
DB_NAME=familycotton
JWT_SECRET=change-me-in-production
JWT_ACCESS_TTL=1h
JWT_REFRESH_TTL=720h
SERVER_PORT=8082
```

- [ ] **Step 5: Create stub main.go**

Write `familycotton-api/cmd/api/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("FamilyCotton API starting...")
}
```

- [ ] **Step 6: Verify it compiles**

```bash
cd familycotton-api && go build ./cmd/api/
```

Expected: no errors, binary created.

- [ ] **Step 7: Commit**

```bash
git add familycotton-api/
git commit -m "feat: scaffold project structure and go module"
```

---

### Task 2: Docker Compose + Dockerfile

**Files:**
- Create: `familycotton-api/Dockerfile`
- Create: `familycotton-api/docker-compose.yml`

- [ ] **Step 1: Create Dockerfile**

Write `familycotton-api/Dockerfile`:
```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /api .
COPY migrations/ ./migrations/

EXPOSE 8082
CMD ["./api"]
```

- [ ] **Step 2: Create docker-compose.yml**

Write `familycotton-api/docker-compose.yml`:
```yaml
version: "3.8"

services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: familycotton
      POSTGRES_PASSWORD: familycotton_secret
      POSTGRES_DB: familycotton
    ports:
      - "5434:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    networks:
      - familycotton-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U familycotton"]
      interval: 5s
      timeout: 3s
      retries: 5

  api:
    build: .
    ports:
      - "8082:8082"
    env_file:
      - .env
    depends_on:
      db:
        condition: service_healthy
    networks:
      - familycotton-net

volumes:
  pgdata:

networks:
  familycotton-net:
    driver: bridge
```

- [ ] **Step 3: Create .env for local Docker**

```bash
cp familycotton-api/.env.example familycotton-api/.env
```

- [ ] **Step 4: Verify docker-compose config is valid**

```bash
cd familycotton-api && docker compose config --quiet
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add familycotton-api/Dockerfile familycotton-api/docker-compose.yml
git commit -m "feat: add Docker Compose and Dockerfile"
```

---

### Task 3: Configuration

**Files:**
- Create: `familycotton-api/internal/config/config.go`

- [ ] **Step 1: Create config.go**

Write `familycotton-api/internal/config/config.go`:
```go
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBURL          string
	JWTSecret      string
	JWTAccessTTL   time.Duration
	JWTRefreshTTL  time.Duration
	ServerPort     string
}

func Load() (*Config, error) {
	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "1h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5434")
	user := getEnv("DB_USER", "familycotton")
	password := getEnv("DB_PASSWORD", "familycotton_secret")
	dbName := getEnv("DB_NAME", "familycotton")

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbName)

	return &Config{
		DBHost:        host,
		DBPort:        port,
		DBUser:        user,
		DBPassword:    password,
		DBName:        dbName,
		DBURL:         dbURL,
		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-production"),
		JWTAccessTTL:  accessTTL,
		JWTRefreshTTL: refreshTTL,
		ServerPort:    getEnv("SERVER_PORT", "8082"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd familycotton-api && go build ./internal/config/
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add familycotton-api/internal/config/
git commit -m "feat: add env configuration loading"
```

---

### Task 4: Database Migrations (All 20 Tables)

**Files:**
- Create: `familycotton-api/migrations/migrations.go`
- Create: `familycotton-api/migrations/001_init.sql`

- [ ] **Step 1: Create migrations embed file**

Write `familycotton-api/migrations/migrations.go`:
```go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

- [ ] **Step 2: Write the migration**

Write `familycotton-api/migrations/001_init.sql`:
```sql
-- +goose Up

-- 1. users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    login VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('owner', 'employee')),
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. refresh_tokens
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- 3. suppliers
CREATE TABLE suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    notes TEXT,
    total_debt DECIMAL(15,2) NOT NULL DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. creditors
CREATE TABLE creditors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    notes TEXT,
    total_debt DECIMAL(15,2) NOT NULL DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. clients
CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    total_debt DECIMAL(15,2) NOT NULL DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. products
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    brand VARCHAR(255),
    supplier_id UUID REFERENCES suppliers(id),
    photo_url TEXT,
    cost_price DECIMAL(15,2) NOT NULL DEFAULT 0,
    sell_price DECIMAL(15,2) NOT NULL DEFAULT 0,
    qty_shop INTEGER NOT NULL DEFAULT 0 CHECK (qty_shop >= 0),
    qty_warehouse INTEGER NOT NULL DEFAULT 0 CHECK (qty_warehouse >= 0),
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_products_supplier_id ON products(supplier_id);

-- 7. shifts
CREATE TABLE shifts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opened_by UUID NOT NULL REFERENCES users(id),
    closed_by UUID REFERENCES users(id),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    total_cash DECIMAL(15,2) NOT NULL DEFAULT 0,
    total_terminal DECIMAL(15,2) NOT NULL DEFAULT 0,
    total_online DECIMAL(15,2) NOT NULL DEFAULT 0,
    total_debt_sales DECIMAL(15,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed'))
);

-- 8. sales
CREATE TABLE sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shift_id UUID NOT NULL REFERENCES shifts(id),
    client_id UUID REFERENCES clients(id),
    total_amount DECIMAL(15,2) NOT NULL,
    paid_cash DECIMAL(15,2) NOT NULL DEFAULT 0,
    paid_terminal DECIMAL(15,2) NOT NULL DEFAULT 0,
    paid_online DECIMAL(15,2) NOT NULL DEFAULT 0,
    paid_debt DECIMAL(15,2) NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sales_shift_id ON sales(shift_id);
CREATE INDEX idx_sales_client_id ON sales(client_id);

-- 9. sale_items
CREATE TABLE sale_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id UUID NOT NULL REFERENCES sales(id),
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(15,2) NOT NULL,
    subtotal DECIMAL(15,2) NOT NULL
);
CREATE INDEX idx_sale_items_sale_id ON sale_items(sale_id);
CREATE INDEX idx_sale_items_product_id ON sale_items(product_id);

-- 10. sale_returns
CREATE TABLE sale_returns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id UUID NOT NULL REFERENCES sales(id),
    sale_item_id UUID NOT NULL REFERENCES sale_items(id),
    new_product_id UUID REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    return_type VARCHAR(20) NOT NULL CHECK (return_type IN ('full', 'exchange', 'exchange_diff')),
    refund_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    surcharge_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sale_returns_sale_id ON sale_returns(sale_id);
CREATE INDEX idx_sale_returns_sale_item_id ON sale_returns(sale_item_id);

-- 11. purchase_orders
CREATE TABLE purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    total_amount DECIMAL(15,2) NOT NULL,
    paid_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'unpaid' CHECK (status IN ('paid', 'partial', 'unpaid')),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 12. purchase_order_items
CREATE TABLE purchase_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id),
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_cost DECIMAL(15,2) NOT NULL,
    destination VARCHAR(20) NOT NULL CHECK (destination IN ('shop', 'warehouse'))
);
CREATE INDEX idx_purchase_order_items_order_id ON purchase_order_items(purchase_order_id);

-- 13. supplier_payments
CREATE TABLE supplier_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    purchase_order_id UUID REFERENCES purchase_orders(id),
    payment_type VARCHAR(20) NOT NULL CHECK (payment_type IN ('money', 'product_return')),
    amount DECIMAL(15,2) NOT NULL,
    returned_product_id UUID REFERENCES products(id),
    returned_qty INTEGER,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_supplier_payments_supplier_id ON supplier_payments(supplier_id);
CREATE INDEX idx_supplier_payments_order_id ON supplier_payments(purchase_order_id);

-- 14. creditor_transactions
CREATE TABLE creditor_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creditor_id UUID NOT NULL REFERENCES creditors(id),
    type VARCHAR(20) NOT NULL CHECK (type IN ('receive', 'repay')),
    currency VARCHAR(3) NOT NULL CHECK (currency IN ('UZS', 'USD')),
    amount DECIMAL(15,2) NOT NULL,
    exchange_rate DECIMAL(15,4) NOT NULL DEFAULT 1,
    amount_uzs DECIMAL(15,2) NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_creditor_transactions_creditor_id ON creditor_transactions(creditor_id);

-- 15. client_payments
CREATE TABLE client_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    amount DECIMAL(15,2) NOT NULL,
    payment_method VARCHAR(20) NOT NULL CHECK (payment_method IN ('cash', 'terminal', 'online')),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_client_payments_client_id ON client_payments(client_id);

-- 16. stock_transfers
CREATE TABLE stock_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    direction VARCHAR(30) NOT NULL CHECK (direction IN ('warehouse_to_shop', 'shop_to_warehouse')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 17. safe_transactions
CREATE TABLE safe_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('income', 'expense', 'transfer')),
    source VARCHAR(50) NOT NULL CHECK (source IN (
        'shift', 'creditor_receive', 'creditor_repay', 'client_payment',
        'client_refund', 'supplier_payment', 'purchase_cash',
        'online_owner_debt', 'owner_deposit'
    )),
    balance_type VARCHAR(20) NOT NULL CHECK (balance_type IN ('cash', 'terminal', 'online')),
    amount DECIMAL(15,2) NOT NULL,
    description TEXT,
    reference_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_safe_transactions_created_at ON safe_transactions(created_at);
CREATE INDEX idx_safe_transactions_source ON safe_transactions(source);

-- 18. owner_debts
CREATE TABLE owner_debts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shift_id UUID NOT NULL REFERENCES shifts(id),
    amount DECIMAL(15,2) NOT NULL,
    is_settled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settled_at TIMESTAMPTZ
);

-- 19. inventory_checks
CREATE TABLE inventory_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location VARCHAR(20) NOT NULL CHECK (location IN ('shop', 'warehouse')),
    checked_by UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- 20. inventory_check_items
CREATE TABLE inventory_check_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inventory_check_id UUID NOT NULL REFERENCES inventory_checks(id),
    product_id UUID NOT NULL REFERENCES products(id),
    expected_qty INTEGER NOT NULL,
    actual_qty INTEGER,
    difference INTEGER
);
CREATE INDEX idx_inventory_check_items_check_id ON inventory_check_items(inventory_check_id);

-- Seed default owner account (password: admin123)
INSERT INTO users (name, login, password_hash, role) VALUES (
    'Owner', 'admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'owner'
);

-- +goose Down
DROP TABLE IF EXISTS inventory_check_items CASCADE;
DROP TABLE IF EXISTS inventory_checks CASCADE;
DROP TABLE IF EXISTS owner_debts CASCADE;
DROP TABLE IF EXISTS safe_transactions CASCADE;
DROP TABLE IF EXISTS stock_transfers CASCADE;
DROP TABLE IF EXISTS client_payments CASCADE;
DROP TABLE IF EXISTS creditor_transactions CASCADE;
DROP TABLE IF EXISTS supplier_payments CASCADE;
DROP TABLE IF EXISTS purchase_order_items CASCADE;
DROP TABLE IF EXISTS purchase_orders CASCADE;
DROP TABLE IF EXISTS sale_returns CASCADE;
DROP TABLE IF EXISTS sale_items CASCADE;
DROP TABLE IF EXISTS sales CASCADE;
DROP TABLE IF EXISTS shifts CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS clients CASCADE;
DROP TABLE IF EXISTS creditors CASCADE;
DROP TABLE IF EXISTS suppliers CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS users CASCADE;
```

- [ ] **Step 3: Add goose dependency and verify migration syntax**

```bash
cd familycotton-api && go get github.com/pressly/goose/v3
```

- [ ] **Step 4: Commit**

```bash
git add familycotton-api/migrations/ familycotton-api/go.mod familycotton-api/go.sum
git commit -m "feat: add database migration with all 20 tables"
```

---

### Task 5: Response Models + Error Handling

**Files:**
- Create: `familycotton-api/internal/model/response.go`
- Create: `familycotton-api/internal/handler/helpers.go`

- [ ] **Step 1: Create response models**

Write `familycotton-api/internal/model/response.go`:
```go
package model

import "fmt"

// Meta holds pagination metadata.
type Meta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// SuccessResponse wraps successful API responses.
type SuccessResponse struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

// ErrorBody is the error payload.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse wraps error API responses.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// Domain errors.
var (
	ErrNotFound    = fmt.Errorf("not found")
	ErrValidation  = fmt.Errorf("validation error")
	ErrForbidden   = fmt.Errorf("forbidden")
	ErrUnauthorized = fmt.Errorf("unauthorized")
	ErrConflict    = fmt.Errorf("conflict")
)

// AppError wraps a sentinel with a user-facing message.
type AppError struct {
	Err     error
	Message string
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

func NewAppError(sentinel error, msg string) *AppError {
	return &AppError{Err: sentinel, Message: msg}
}
```

- [ ] **Step 2: Create handler helpers**

Write `familycotton-api/internal/handler/helpers.go`:
```go
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/familycotton/api/internal/model"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func respondSuccess(w http.ResponseWriter, status int, data any) {
	respondJSON(w, status, model.SuccessResponse{Data: data})
}

func respondList(w http.ResponseWriter, data any, page, limit, total int) {
	respondJSON(w, http.StatusOK, model.SuccessResponse{
		Data: data,
		Meta: &model.Meta{Page: page, Limit: limit, Total: total},
	})
}

func respondError(w http.ResponseWriter, err error) {
	var appErr *model.AppError
	if errors.As(err, &appErr) {
		status := mapErrToStatus(appErr.Unwrap())
		code := mapErrToCode(appErr.Unwrap())
		respondJSON(w, status, model.ErrorResponse{
			Error: model.ErrorBody{Code: code, Message: appErr.Message},
		})
		return
	}

	// Fallback for unknown errors — don't expose internals.
	slog.Error("unhandled error", "error", err)
	respondJSON(w, http.StatusInternalServerError, model.ErrorResponse{
		Error: model.ErrorBody{Code: "INTERNAL_ERROR", Message: "internal server error"},
	})
}

func mapErrToStatus(err error) int {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, model.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, model.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, model.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func mapErrToCode(err error) string {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, model.ErrValidation):
		return "VALIDATION_ERROR"
	case errors.Is(err, model.ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, model.ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, model.ErrConflict):
		return "CONFLICT"
	default:
		return "INTERNAL_ERROR"
	}
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return model.NewAppError(model.ErrValidation, "invalid JSON body")
	}
	return nil
}

func paginationParams(r *http.Request) (page, limit int) {
	page = 1
	limit = 20
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	return
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd familycotton-api && go build ./internal/model/ && go build ./internal/handler/
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add familycotton-api/internal/model/response.go familycotton-api/internal/handler/helpers.go
git commit -m "feat: add response models and handler helpers"
```

---

### Task 6: User Model + Repository

**Files:**
- Create: `familycotton-api/internal/model/user.go`
- Create: `familycotton-api/internal/repository/user.go`

- [ ] **Step 1: Create user model**

Write `familycotton-api/internal/model/user.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	IsDeleted    bool      `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateUserRequest struct {
	Name     string `json:"name"`
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (r *CreateUserRequest) Validate() error {
	if r.Name == "" {
		return NewAppError(ErrValidation, "name is required")
	}
	if r.Login == "" {
		return NewAppError(ErrValidation, "login is required")
	}
	if len(r.Password) < 6 {
		return NewAppError(ErrValidation, "password must be at least 6 characters")
	}
	if r.Role != "owner" && r.Role != "employee" {
		return NewAppError(ErrValidation, "role must be 'owner' or 'employee'")
	}
	return nil
}

type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty"`
	Login    *string `json:"login,omitempty"`
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"`
}

func (r *UpdateUserRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "name cannot be empty")
	}
	if r.Login != nil && *r.Login == "" {
		return NewAppError(ErrValidation, "login cannot be empty")
	}
	if r.Password != nil && len(*r.Password) < 6 {
		return NewAppError(ErrValidation, "password must be at least 6 characters")
	}
	if r.Role != nil && *r.Role != "owner" && *r.Role != "employee" {
		return NewAppError(ErrValidation, "role must be 'owner' or 'employee'")
	}
	return nil
}
```

- [ ] **Step 2: Create user repository**

Write `familycotton-api/internal/repository/user.go`:
```go
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, name, login, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Name, u.Login, u.PasswordHash, u.Role,
	)
	if err != nil {
		if isDuplicateKey(err) {
			return model.NewAppError(model.ErrConflict, "login already exists")
		}
		return err
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, login, password_hash, role, is_deleted, created_at, updated_at
		 FROM users WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&u.ID, &u.Name, &u.Login, &u.PasswordHash, &u.Role, &u.IsDeleted, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "user not found")
	}
	return u, err
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, login, password_hash, role, is_deleted, created_at, updated_at
		 FROM users WHERE login = $1 AND is_deleted = false`, login,
	).Scan(&u.ID, &u.Name, &u.Login, &u.PasswordHash, &u.Role, &u.IsDeleted, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "user not found")
	}
	return u, err
}

func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, login, password_hash, role, is_deleted, created_at, updated_at
		 FROM users WHERE is_deleted = false ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Login, &u.PasswordHash, &u.Role, &u.IsDeleted, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, u *model.User) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET name=$1, login=$2, password_hash=$3, role=$4, updated_at=NOW()
		 WHERE id=$5 AND is_deleted = false`,
		u.Name, u.Login, u.PasswordHash, u.Role, u.ID,
	)
	if err != nil && isDuplicateKey(err) {
		return model.NewAppError(model.ErrConflict, "login already exists")
	}
	return err
}

func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE users SET is_deleted = true, updated_at = NOW() WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "user not found")
	}
	return nil
}

// isDuplicateKey checks for PostgreSQL unique violation (23505).
func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
```

- [ ] **Step 3: Add dependencies**

```bash
cd familycotton-api && go get github.com/google/uuid github.com/jackc/pgx/v5
```

- [ ] **Step 4: Verify compilation**

```bash
cd familycotton-api && go build ./internal/model/ && go build ./internal/repository/
```

- [ ] **Step 5: Commit**

```bash
git add familycotton-api/internal/model/user.go familycotton-api/internal/repository/user.go familycotton-api/go.mod familycotton-api/go.sum
git commit -m "feat: add user model and repository"
```

---

### Task 7: Auth Model + Token Repository

**Files:**
- Create: `familycotton-api/internal/model/auth.go`
- Create: `familycotton-api/internal/repository/token.go`

- [ ] **Step 1: Create auth model**

Write `familycotton-api/internal/model/auth.go`:
```go
package model

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	if r.Login == "" {
		return NewAppError(ErrValidation, "login is required")
	}
	if r.Password == "" {
		return NewAppError(ErrValidation, "password is required")
	}
	return nil
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r *RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return NewAppError(ErrValidation, "refresh_token is required")
	}
	return nil
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
```

- [ ] **Step 2: Create token repository**

Write `familycotton-api/internal/repository/token.go`:
```go
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository struct {
	db *pgxpool.Pool
}

func NewTokenRepository(db *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *TokenRepository) ExistsByHash(ctx context.Context, tokenHash string) (bool, uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT user_id FROM refresh_tokens WHERE token_hash = $1 AND expires_at > NOW()`,
		tokenHash,
	).Scan(&userID)
	if err != nil {
		return false, uuid.Nil, nil // not found or expired
	}
	return true, userID, nil
}

func (r *TokenRepository) DeleteByHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *TokenRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}

func (r *TokenRepository) CleanExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < NOW()`)
	return err
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd familycotton-api && go build ./internal/model/ && go build ./internal/repository/
```

- [ ] **Step 4: Commit**

```bash
git add familycotton-api/internal/model/auth.go familycotton-api/internal/repository/token.go
git commit -m "feat: add auth model and token repository"
```

---

### Task 8: Auth Service (Login, Refresh, Logout)

**Files:**
- Create: `familycotton-api/internal/service/auth.go`

- [ ] **Step 1: Create auth service**

Write `familycotton-api/internal/service/auth.go`:
```go
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	tokenRepo *repository.TokenRepository
	jwtSecret []byte
	accessTTL time.Duration
	refreshTTL time.Duration
}

func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtSecret:  []byte(jwtSecret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.TokenPair, error) {
	user, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err != nil {
		return nil, model.NewAppError(model.ErrUnauthorized, "invalid login or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, model.NewAppError(model.ErrUnauthorized, "invalid login or password")
	}

	return s.generateTokenPair(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, req *model.RefreshRequest) (*model.TokenPair, error) {
	hash := hashToken(req.RefreshToken)

	exists, userID, err := s.tokenRepo.ExistsByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, model.NewAppError(model.ErrUnauthorized, "invalid or expired refresh token")
	}

	// Delete old refresh token (rotation).
	if err := s.tokenRepo.DeleteByHash(ctx, hash); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, model.NewAppError(model.ErrUnauthorized, "user not found")
	}

	return s.generateTokenPair(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash := hashToken(refreshToken)
	return s.tokenRepo.DeleteByHash(ctx, hash)
}

func (s *AuthService) generateTokenPair(ctx context.Context, user *model.User) (*model.TokenPair, error) {
	now := time.Now()

	// Access token.
	accessClaims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"role": user.Role,
		"exp":  now.Add(s.accessTTL).Unix(),
		"iat":  now.Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	// Refresh token — random UUID, store hash in DB.
	refreshRaw := uuid.New().String()
	refreshHash := hashToken(refreshRaw)
	expiresAt := now.Add(s.refreshTTL)

	if err := s.tokenRepo.Create(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return nil, err
	}

	return &model.TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshRaw,
	}, nil
}

// ParseAccessToken validates an access token and returns (userID, role).
func (s *AuthService) ParseAccessToken(tokenStr string) (uuid.UUID, string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, model.NewAppError(model.ErrUnauthorized, "invalid token signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return uuid.Nil, "", model.NewAppError(model.ErrUnauthorized, "invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, "", model.NewAppError(model.ErrUnauthorized, "invalid token")
	}

	sub, _ := claims.GetSubject()
	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, "", model.NewAppError(model.ErrUnauthorized, "invalid token subject")
	}

	role, _ := claims["role"].(string)
	return userID, role, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
```

- [ ] **Step 2: Add dependencies**

```bash
cd familycotton-api && go get github.com/golang-jwt/jwt/v5 golang.org/x/crypto
```

- [ ] **Step 3: Verify compilation**

```bash
cd familycotton-api && go build ./internal/service/
```

- [ ] **Step 4: Commit**

```bash
git add familycotton-api/internal/service/auth.go familycotton-api/go.mod familycotton-api/go.sum
git commit -m "feat: add auth service with login, refresh, logout"
```

---

### Task 9: User Service

**Files:**
- Create: `familycotton-api/internal/service/user.go`

- [ ] **Step 1: Create user service**

Write `familycotton-api/internal/service/user.go`:
```go
package service

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Create(ctx context.Context, req *model.CreateUserRequest) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:           uuid.New(),
		Name:         req.Name,
		Login:        req.Login,
		PasswordHash: string(hash),
		Role:         req.Role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context) ([]model.User, error) {
	return s.repo.List(ctx)
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateUserRequest) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Login != nil {
		user.Login = *req.Login
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hash)
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd familycotton-api && go build ./internal/service/
```

- [ ] **Step 3: Commit**

```bash
git add familycotton-api/internal/service/user.go
git commit -m "feat: add user service with CRUD and password hashing"
```

---

### Task 10: Middleware (Auth, RBAC, Logging, CORS)

**Files:**
- Create: `familycotton-api/internal/middleware/auth.go`
- Create: `familycotton-api/internal/middleware/rbac.go`
- Create: `familycotton-api/internal/middleware/logging.go`
- Create: `familycotton-api/internal/middleware/cors.go`

- [ ] **Step 1: Create auth middleware**

Write `familycotton-api/internal/middleware/auth.go`:
```go
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/service"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RoleKey   contextKey = "role"
)

func GetUserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(UserIDKey).(uuid.UUID)
	return id
}

func GetRole(ctx context.Context) string {
	role, _ := ctx.Value(RoleKey).(string)
	return role
}

func jsonError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"code":"%s","message":"%s"}}`, code, message)
}

func Auth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				jsonError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				jsonError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization format")
				return
			}

			userID, role, err := authService.ParseAccessToken(parts[1])
			if err != nil {
				jsonError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, RoleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

- [ ] **Step 2: Create RBAC middleware**

Write `familycotton-api/internal/middleware/rbac.go`:
```go
package middleware

import (
	"net/http"
)

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r.Context())
			if !allowed[role] {
				jsonError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 3: Create logging middleware**

Write `familycotton-api/internal/middleware/logging.go`:
```go
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
```

- [ ] **Step 4: Create CORS middleware**

Write `familycotton-api/internal/middleware/cors.go`:
```go
package middleware

import "net/http"

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 5: Verify compilation**

```bash
cd familycotton-api && go build ./internal/middleware/
```

- [ ] **Step 6: Commit**

```bash
git add familycotton-api/internal/middleware/
git commit -m "feat: add auth, rbac, logging, cors middleware"
```

---

### Task 11: Auth Handler

**Files:**
- Create: `familycotton-api/internal/handler/auth.go`

- [ ] **Step 1: Create auth handler**

Write `familycotton-api/internal/handler/auth.go`:
```go
package handler

import (
	"net/http"

	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	userService *service.UserService
}

func NewAuthHandler(authService *service.AuthService, userService *service.UserService) *AuthHandler {
	return &AuthHandler{authService: authService, userService: userService}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, err)
		return
	}

	tokens, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, err)
		return
	}

	tokens, err := h.authService.Refresh(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if err := h.authService.Logout(r.Context(), req.RefreshToken); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, user)
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd familycotton-api && go build ./internal/handler/
```

- [ ] **Step 3: Commit**

```bash
git add familycotton-api/internal/handler/auth.go
git commit -m "feat: add auth handler (login, refresh, logout, me)"
```

---

### Task 12: User Handler

**Files:**
- Create: `familycotton-api/internal/handler/user.go`

- [ ] **Step 1: Create user handler**

Write `familycotton-api/internal/handler/user.go`:
```go
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, users)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	user, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusCreated, user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid user id"))
		return
	}

	var req model.UpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	user, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, user)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid user id"))
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
```

- [ ] **Step 2: Add chi dependency**

```bash
cd familycotton-api && go get github.com/go-chi/chi/v5
```

- [ ] **Step 3: Verify compilation**

```bash
cd familycotton-api && go build ./internal/handler/
```

- [ ] **Step 4: Commit**

```bash
git add familycotton-api/internal/handler/user.go familycotton-api/go.mod familycotton-api/go.sum
git commit -m "feat: add user handler with CRUD endpoints"
```

---

### Task 13: Router

**Files:**
- Create: `familycotton-api/internal/router/router.go`

- [ ] **Step 1: Create router**

Write `familycotton-api/internal/router/router.go`:
```go
package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/familycotton/api/internal/handler"
	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/service"
)

func New(
	authService *service.AuthService,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(middleware.CORS)
	r.Use(middleware.Logging)

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes (no auth required).
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Post("/auth/logout", authHandler.Logout)

		// Protected routes.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authService))

			// Auth.
			r.Get("/auth/me", authHandler.Me)

			// Users (owner only).
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequireRole("owner"))
				r.Get("/", userHandler.List)
				r.Post("/", userHandler.Create)
				r.Put("/{id}", userHandler.Update)
				r.Delete("/{id}", userHandler.Delete)
			})
		})
	})

	return r
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd familycotton-api && go build ./internal/router/
```

- [ ] **Step 3: Commit**

```bash
git add familycotton-api/internal/router/
git commit -m "feat: add chi router with auth and user routes"
```

---

### Task 14: Main.go — Wire Everything Together

**Files:**
- Modify: `familycotton-api/cmd/api/main.go`

- [ ] **Step 1: Write the full main.go**

Write `familycotton-api/cmd/api/main.go`:
```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // register pgx as database/sql driver
	"github.com/pressly/goose/v3"

	"github.com/familycotton/api/internal/config"
	"github.com/familycotton/api/internal/handler"
	"github.com/familycotton/api/internal/repository"
	"github.com/familycotton/api/internal/router"
	"github.com/familycotton/api/internal/service"
	"github.com/familycotton/api/migrations"
)

func main() {
	// Structured JSON logging.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Database connection.
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to database")

	// Run migrations.
	if err := runMigrations(cfg.DBURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Repositories.
	userRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewTokenRepository(pool)

	// Services.
	authService := service.NewAuthService(userRepo, tokenRepo, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	userService := service.NewUserService(userRepo)

	// Handlers.
	authHandler := handler.NewAuthHandler(authService, userService)
	userHandler := handler.NewUserHandler(userService)

	// Router.
	r := router.New(authService, authHandler, userHandler)

	// Server.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		slog.Info("server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	slog.Info("server stopped")
}

func runMigrations(dbURL string) error {
	goose.SetBaseFS(migrations.FS)

	db, err := goose.OpenDBWithDriver("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("migrations applied successfully")
	return nil
}
```

- [ ] **Step 2: Tidy modules**

```bash
cd familycotton-api && go mod tidy
```

- [ ] **Step 3: Verify compilation**

```bash
cd familycotton-api && go build ./cmd/api/
```

Expected: binary builds successfully.

- [ ] **Step 4: Commit**

```bash
git add familycotton-api/cmd/api/main.go familycotton-api/go.mod familycotton-api/go.sum
git commit -m "feat: wire main.go with DI, migrations, graceful shutdown"
```

---

### Task 15: Integration Test — Boot and Smoke

**Files:**
- No new files — validates the full stack works end-to-end.

- [ ] **Step 1: Start Docker Compose**

```bash
cd familycotton-api && docker compose up -d --build
```

Expected: both `db` and `api` containers start without errors.

- [ ] **Step 2: Wait for API to be ready and test health**

```bash
sleep 5 && curl -s http://localhost:8082/api/v1/auth/me | jq .
```

Expected: `401 Unauthorized` response (no token) — confirms server is running and router works.

- [ ] **Step 3: Test login with seed user**

```bash
curl -s -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"login":"admin","password":"admin123"}' | jq .
```

Expected: JSON with `access_token` and `refresh_token`.

**Note:** The seed password hash in migration must match `admin123`. If the bcrypt hash in the migration is incorrect, generate a new one:
```bash
htpasswd -nbBC 10 "" admin123 | tr -d ':\n' | sed 's/$2y/$2a/'
```
Then update the migration's INSERT.

- [ ] **Step 4: Test authenticated endpoints**

```bash
# Save token from login response.
TOKEN=$(curl -s -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"login":"admin","password":"admin123"}' | jq -r '.data.access_token')

# Test /auth/me
curl -s http://localhost:8082/api/v1/auth/me \
  -H "Authorization: Bearer $TOKEN" | jq .

# Test GET /users
curl -s http://localhost:8082/api/v1/users \
  -H "Authorization: Bearer $TOKEN" | jq .

# Test POST /users (create employee)
curl -s -X POST http://localhost:8082/api/v1/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Test Employee","login":"employee1","password":"pass123","role":"employee"}' | jq .
```

Expected: all return valid JSON with `data` wrapper.

- [ ] **Step 5: Test RBAC — employee cannot access /users**

```bash
# Login as employee.
EMP_TOKEN=$(curl -s -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"login":"employee1","password":"pass123"}' | jq -r '.data.access_token')

# Try to list users — should be 403.
curl -s http://localhost:8082/api/v1/users \
  -H "Authorization: Bearer $EMP_TOKEN" | jq .
```

Expected: `403 Forbidden` response.

- [ ] **Step 6: Verify database tables were created**

```bash
docker compose exec db psql -U familycotton -c "\dt" | head -30
```

Expected: all 20 tables listed.

- [ ] **Step 7: Stop Docker Compose**

```bash
cd familycotton-api && docker compose down
```

- [ ] **Step 8: Commit (if any fixes were needed)**

```bash
git add -A && git commit -m "fix: integration test fixes for phase 1"
```

Only commit if changes were made. If everything passed cleanly, skip this step.

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Project scaffolding | go.mod, main.go stub, .gitignore, .env.example |
| 2 | Docker Compose + Dockerfile | docker-compose.yml, Dockerfile |
| 3 | Configuration | config/config.go |
| 4 | Database migrations | migrations/001_init.sql (all 20 tables) |
| 5 | Response models + error handling | model/response.go, handler/helpers.go |
| 6 | User model + repository | model/user.go, repository/user.go |
| 7 | Auth model + token repository | model/auth.go, repository/token.go |
| 8 | Auth service | service/auth.go |
| 9 | User service | service/user.go |
| 10 | Middleware | middleware/auth.go, rbac.go, logging.go, cors.go |
| 11 | Auth handler | handler/auth.go |
| 12 | User handler | handler/user.go |
| 13 | Router | router/router.go |
| 14 | Main.go wiring | cmd/api/main.go |
| 15 | Integration smoke test | (manual curl tests) |
