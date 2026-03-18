# Phase 2 — Catalogs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add CRUD for Products, Suppliers, Clients, and Creditors with offset-based pagination, search filters, and role-based access.

**Architecture:** Follows Phase 1 layered pattern (handler → service → repository). Each domain gets its own model, repository, service, and handler file. Pagination is offset-based (`?page=1&limit=20`). All list endpoints return `{data: [...], meta: {page, limit, total}}`.

**Tech Stack:** Go 1.25, chi v5, pgx v5, shopspring/decimal (new — for monetary fields)

**Spec:** `docs/superpowers/specs/2026-03-18-familycotton-backend-design.md`

---

## File Map

```
familycotton-api/internal/
├── model/
│   ├── supplier.go     # Supplier, CreateSupplierReq, UpdateSupplierReq
│   ├── creditor.go     # Creditor, CreateCreditorReq, UpdateCreditorReq
│   ├── client.go       # Client, CreateClientReq, UpdateClientReq
│   └── product.go      # Product, CreateProductReq, UpdateProductReq
├── repository/
│   ├── supplier.go     # SupplierRepository (CRUD + pagination)
│   ├── creditor.go     # CreditorRepository (CRUD + pagination)
│   ├── client.go       # ClientRepository (CRUD + pagination)
│   └── product.go      # ProductRepository (CRUD + search + pagination)
├── service/
│   ├── supplier.go     # SupplierService
│   ├── creditor.go     # CreditorService
│   ├── client.go       # ClientService
│   └── product.go      # ProductService (margin computation)
├── handler/
│   ├── supplier.go     # SupplierHandler
│   ├── creditor.go     # CreditorHandler
│   ├── client.go       # ClientHandler
│   └── product.go      # ProductHandler
├── router/router.go    # Add new routes (modify)
cmd/api/main.go         # Add new DI wiring (modify)
```

---

### Task 1: Add shopspring/decimal dependency

**Files:**
- Modify: `familycotton-api/go.mod`

- [ ] **Step 1: Add dependency**

```bash
cd familycotton-api && go get github.com/shopspring/decimal
```

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: add shopspring/decimal for monetary fields"
```

---

### Task 2: Supplier Model + Repository

**Files:**
- Create: `familycotton-api/internal/model/supplier.go`
- Create: `familycotton-api/internal/repository/supplier.go`

- [ ] **Step 1: Create supplier model**

Write `familycotton-api/internal/model/supplier.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Supplier struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Phone     *string         `json:"phone"`
	Notes     *string         `json:"notes"`
	TotalDebt decimal.Decimal `json:"total_debt"`
	IsDeleted bool            `json:"-"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateSupplierRequest struct {
	Name  string  `json:"name"`
	Phone *string `json:"phone"`
	Notes *string `json:"notes"`
}

func (r *CreateSupplierRequest) Validate() error {
	if r.Name == "" {
		return NewAppError(ErrValidation, "name is required")
	}
	return nil
}

type UpdateSupplierRequest struct {
	Name  *string `json:"name,omitempty"`
	Phone *string `json:"phone,omitempty"`
	Notes *string `json:"notes,omitempty"`
}

func (r *UpdateSupplierRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "name cannot be empty")
	}
	return nil
}
```

- [ ] **Step 2: Create supplier repository**

Write `familycotton-api/internal/repository/supplier.go`:
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

type SupplierRepository struct {
	db *pgxpool.Pool
}

func NewSupplierRepository(db *pgxpool.Pool) *SupplierRepository {
	return &SupplierRepository{db: db}
}

func (r *SupplierRepository) Create(ctx context.Context, s *model.Supplier) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO suppliers (id, name, phone, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		s.ID, s.Name, s.Phone, s.Notes,
	).Scan(&s.CreatedAt, &s.UpdatedAt)
}

func (r *SupplierRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Supplier, error) {
	s := &model.Supplier{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, phone, notes, total_debt, is_deleted, created_at, updated_at
		 FROM suppliers WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&s.ID, &s.Name, &s.Phone, &s.Notes, &s.TotalDebt, &s.IsDeleted, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "supplier not found")
	}
	return s, err
}

func (r *SupplierRepository) List(ctx context.Context, page, limit int) ([]model.Supplier, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM suppliers WHERE is_deleted = false`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, name, phone, notes, total_debt, is_deleted, created_at, updated_at
		 FROM suppliers WHERE is_deleted = false
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var suppliers []model.Supplier
	for rows.Next() {
		var s model.Supplier
		if err := rows.Scan(&s.ID, &s.Name, &s.Phone, &s.Notes, &s.TotalDebt, &s.IsDeleted, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, total, rows.Err()
}

func (r *SupplierRepository) Update(ctx context.Context, s *model.Supplier) error {
	err := r.db.QueryRow(ctx,
		`UPDATE suppliers SET name=$1, phone=$2, notes=$3, updated_at=NOW()
		 WHERE id=$4 AND is_deleted = false
		 RETURNING updated_at`,
		s.Name, s.Phone, s.Notes, s.ID,
	).Scan(&s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.NewAppError(model.ErrNotFound, "supplier not found")
	}
	return err
}

func (r *SupplierRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE suppliers SET is_deleted = true, updated_at = NOW()
		 WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "supplier not found")
	}
	return nil
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd familycotton-api && go build ./internal/model/ && go build ./internal/repository/
```

- [ ] **Step 4: Commit**

```bash
git add internal/model/supplier.go internal/repository/supplier.go
git commit -m "feat: add supplier model and repository"
```

---

### Task 3: Supplier Service + Handler

**Files:**
- Create: `familycotton-api/internal/service/supplier.go`
- Create: `familycotton-api/internal/handler/supplier.go`

- [ ] **Step 1: Create supplier service**

Write `familycotton-api/internal/service/supplier.go`:
```go
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type SupplierService struct {
	repo *repository.SupplierRepository
}

func NewSupplierService(repo *repository.SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}

func (s *SupplierService) Create(ctx context.Context, req *model.CreateSupplierRequest) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	supplier := &model.Supplier{
		ID:    uuid.New(),
		Name:  req.Name,
		Phone: req.Phone,
		Notes: req.Notes,
	}

	if err := s.repo.Create(ctx, supplier); err != nil {
		return nil, err
	}
	return supplier, nil
}

func (s *SupplierService) GetByID(ctx context.Context, id uuid.UUID) (*model.Supplier, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SupplierService) List(ctx context.Context, page, limit int) ([]model.Supplier, int, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *SupplierService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateSupplierRequest) (*model.Supplier, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	supplier, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		supplier.Name = *req.Name
	}
	if req.Phone != nil {
		supplier.Phone = req.Phone
	}
	if req.Notes != nil {
		supplier.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, supplier); err != nil {
		return nil, err
	}
	return supplier, nil
}

func (s *SupplierService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
```

- [ ] **Step 2: Create supplier handler**

Write `familycotton-api/internal/handler/supplier.go`:
```go
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type SupplierHandler struct {
	service *service.SupplierService
}

func NewSupplierHandler(service *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{service: service}
}

func (h *SupplierHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)

	suppliers, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}

	respondList(w, suppliers, page, limit, total)
}

func (h *SupplierHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid supplier id"))
		return
	}

	supplier, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, supplier)
}

func (h *SupplierHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateSupplierRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	supplier, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusCreated, supplier)
}

func (h *SupplierHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid supplier id"))
		return
	}

	var req model.UpdateSupplierRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	supplier, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, supplier)
}

func (h *SupplierHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid supplier id"))
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, map[string]string{"message": "supplier deleted"})
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd familycotton-api && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/service/supplier.go internal/handler/supplier.go
git commit -m "feat: add supplier service and handler"
```

---

### Task 4: Client Model + Repository + Service + Handler

**Files:**
- Create: `familycotton-api/internal/model/client.go`
- Create: `familycotton-api/internal/repository/client.go`
- Create: `familycotton-api/internal/service/client.go`
- Create: `familycotton-api/internal/handler/client.go`

- [ ] **Step 1: Create client model**

Write `familycotton-api/internal/model/client.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Client struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Phone     *string         `json:"phone"`
	TotalDebt decimal.Decimal `json:"total_debt"`
	IsDeleted bool            `json:"-"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateClientRequest struct {
	Name  string  `json:"name"`
	Phone *string `json:"phone"`
}

func (r *CreateClientRequest) Validate() error {
	if r.Name == "" {
		return NewAppError(ErrValidation, "name is required")
	}
	return nil
}

type UpdateClientRequest struct {
	Name  *string `json:"name,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

func (r *UpdateClientRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "name cannot be empty")
	}
	return nil
}
```

- [ ] **Step 2: Create client repository**

Write `familycotton-api/internal/repository/client.go`:
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

type ClientRepository struct {
	db *pgxpool.Pool
}

func NewClientRepository(db *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{db: db}
}

func (r *ClientRepository) Create(ctx context.Context, c *model.Client) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO clients (id, name, phone)
		 VALUES ($1, $2, $3)
		 RETURNING created_at, updated_at`,
		c.ID, c.Name, c.Phone,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *ClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Client, error) {
	c := &model.Client{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, phone, total_debt, is_deleted, created_at, updated_at
		 FROM clients WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&c.ID, &c.Name, &c.Phone, &c.TotalDebt, &c.IsDeleted, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "client not found")
	}
	return c, err
}

func (r *ClientRepository) List(ctx context.Context, page, limit int) ([]model.Client, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM clients WHERE is_deleted = false`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, name, phone, total_debt, is_deleted, created_at, updated_at
		 FROM clients WHERE is_deleted = false
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var clients []model.Client
	for rows.Next() {
		var c model.Client
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.TotalDebt, &c.IsDeleted, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		clients = append(clients, c)
	}
	return clients, total, rows.Err()
}

func (r *ClientRepository) Update(ctx context.Context, c *model.Client) error {
	err := r.db.QueryRow(ctx,
		`UPDATE clients SET name=$1, phone=$2, updated_at=NOW()
		 WHERE id=$3 AND is_deleted = false
		 RETURNING updated_at`,
		c.Name, c.Phone, c.ID,
	).Scan(&c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.NewAppError(model.ErrNotFound, "client not found")
	}
	return err
}

func (r *ClientRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE clients SET is_deleted = true, updated_at = NOW()
		 WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "client not found")
	}
	return nil
}
```

- [ ] **Step 3: Create client service**

Write `familycotton-api/internal/service/client.go`:
```go
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type ClientService struct {
	repo *repository.ClientRepository
}

func NewClientService(repo *repository.ClientRepository) *ClientService {
	return &ClientService{repo: repo}
}

func (s *ClientService) Create(ctx context.Context, req *model.CreateClientRequest) (*model.Client, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	client := &model.Client{
		ID:    uuid.New(),
		Name:  req.Name,
		Phone: req.Phone,
	}

	if err := s.repo.Create(ctx, client); err != nil {
		return nil, err
	}
	return client, nil
}

func (s *ClientService) GetByID(ctx context.Context, id uuid.UUID) (*model.Client, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ClientService) List(ctx context.Context, page, limit int) ([]model.Client, int, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *ClientService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateClientRequest) (*model.Client, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	client, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		client.Name = *req.Name
	}
	if req.Phone != nil {
		client.Phone = req.Phone
	}

	if err := s.repo.Update(ctx, client); err != nil {
		return nil, err
	}
	return client, nil
}

func (s *ClientService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
```

- [ ] **Step 4: Create client handler**

Write `familycotton-api/internal/handler/client.go`:
```go
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type ClientHandler struct {
	service *service.ClientService
}

func NewClientHandler(service *service.ClientService) *ClientHandler {
	return &ClientHandler{service: service}
}

func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)

	clients, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}

	respondList(w, clients, page, limit, total)
}

func (h *ClientHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid client id"))
		return
	}

	client, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, client)
}

func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateClientRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	client, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusCreated, client)
}

func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid client id"))
		return
	}

	var req model.UpdateClientRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	client, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, client)
}

func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid client id"))
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, map[string]string{"message": "client deleted"})
}
```

- [ ] **Step 5: Verify compilation**

```bash
cd familycotton-api && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/model/client.go internal/repository/client.go internal/service/client.go internal/handler/client.go
git commit -m "feat: add client model, repository, service, handler"
```

---

### Task 5: Creditor Model + Repository + Service + Handler

**Files:**
- Create: `familycotton-api/internal/model/creditor.go`
- Create: `familycotton-api/internal/repository/creditor.go`
- Create: `familycotton-api/internal/service/creditor.go`
- Create: `familycotton-api/internal/handler/creditor.go`

- [ ] **Step 1: Create creditor model**

Write `familycotton-api/internal/model/creditor.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Creditor struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Phone     *string         `json:"phone"`
	Notes     *string         `json:"notes"`
	TotalDebt decimal.Decimal `json:"total_debt"`
	IsDeleted bool            `json:"-"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateCreditorRequest struct {
	Name  string  `json:"name"`
	Phone *string `json:"phone"`
	Notes *string `json:"notes"`
}

func (r *CreateCreditorRequest) Validate() error {
	if r.Name == "" {
		return NewAppError(ErrValidation, "name is required")
	}
	return nil
}

type UpdateCreditorRequest struct {
	Name  *string `json:"name,omitempty"`
	Phone *string `json:"phone,omitempty"`
	Notes *string `json:"notes,omitempty"`
}

func (r *UpdateCreditorRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "name cannot be empty")
	}
	return nil
}
```

- [ ] **Step 2: Create creditor repository**

Write `familycotton-api/internal/repository/creditor.go`:
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

type CreditorRepository struct {
	db *pgxpool.Pool
}

func NewCreditorRepository(db *pgxpool.Pool) *CreditorRepository {
	return &CreditorRepository{db: db}
}

func (r *CreditorRepository) Create(ctx context.Context, c *model.Creditor) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO creditors (id, name, phone, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		c.ID, c.Name, c.Phone, c.Notes,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CreditorRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Creditor, error) {
	c := &model.Creditor{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, phone, notes, total_debt, is_deleted, created_at, updated_at
		 FROM creditors WHERE id = $1 AND is_deleted = false`, id,
	).Scan(&c.ID, &c.Name, &c.Phone, &c.Notes, &c.TotalDebt, &c.IsDeleted, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.NewAppError(model.ErrNotFound, "creditor not found")
	}
	return c, err
}

func (r *CreditorRepository) List(ctx context.Context, page, limit int) ([]model.Creditor, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM creditors WHERE is_deleted = false`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx,
		`SELECT id, name, phone, notes, total_debt, is_deleted, created_at, updated_at
		 FROM creditors WHERE is_deleted = false
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var creditors []model.Creditor
	for rows.Next() {
		var c model.Creditor
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Notes, &c.TotalDebt, &c.IsDeleted, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		creditors = append(creditors, c)
	}
	return creditors, total, rows.Err()
}

func (r *CreditorRepository) Update(ctx context.Context, c *model.Creditor) error {
	err := r.db.QueryRow(ctx,
		`UPDATE creditors SET name=$1, phone=$2, notes=$3, updated_at=NOW()
		 WHERE id=$4 AND is_deleted = false
		 RETURNING updated_at`,
		c.Name, c.Phone, c.Notes, c.ID,
	).Scan(&c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.NewAppError(model.ErrNotFound, "creditor not found")
	}
	return err
}

func (r *CreditorRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE creditors SET is_deleted = true, updated_at = NOW()
		 WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "creditor not found")
	}
	return nil
}
```

- [ ] **Step 3: Create creditor service**

Write `familycotton-api/internal/service/creditor.go`:
```go
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type CreditorService struct {
	repo *repository.CreditorRepository
}

func NewCreditorService(repo *repository.CreditorRepository) *CreditorService {
	return &CreditorService{repo: repo}
}

func (s *CreditorService) Create(ctx context.Context, req *model.CreateCreditorRequest) (*model.Creditor, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	creditor := &model.Creditor{
		ID:    uuid.New(),
		Name:  req.Name,
		Phone: req.Phone,
		Notes: req.Notes,
	}

	if err := s.repo.Create(ctx, creditor); err != nil {
		return nil, err
	}
	return creditor, nil
}

func (s *CreditorService) GetByID(ctx context.Context, id uuid.UUID) (*model.Creditor, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CreditorService) List(ctx context.Context, page, limit int) ([]model.Creditor, int, error) {
	return s.repo.List(ctx, page, limit)
}

func (s *CreditorService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateCreditorRequest) (*model.Creditor, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	creditor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		creditor.Name = *req.Name
	}
	if req.Phone != nil {
		creditor.Phone = req.Phone
	}
	if req.Notes != nil {
		creditor.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, creditor); err != nil {
		return nil, err
	}
	return creditor, nil
}

func (s *CreditorService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
```

- [ ] **Step 4: Create creditor handler**

Write `familycotton-api/internal/handler/creditor.go` — same pattern as supplier handler but with `CreditorService`, `model.Creditor`, `model.CreateCreditorRequest`, `model.UpdateCreditorRequest`, and "creditor" in error messages.

```go
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type CreditorHandler struct {
	service *service.CreditorService
}

func NewCreditorHandler(service *service.CreditorService) *CreditorHandler {
	return &CreditorHandler{service: service}
}

func (h *CreditorHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)
	creditors, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		respondError(w, err)
		return
	}
	respondList(w, creditors, page, limit, total)
}

func (h *CreditorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid creditor id"))
		return
	}
	creditor, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, creditor)
}

func (h *CreditorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateCreditorRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	creditor, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusCreated, creditor)
}

func (h *CreditorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid creditor id"))
		return
	}
	var req model.UpdateCreditorRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}
	creditor, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, creditor)
}

func (h *CreditorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid creditor id"))
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	respondSuccess(w, http.StatusOK, map[string]string{"message": "creditor deleted"})
}
```

- [ ] **Step 5: Verify compilation**

```bash
cd familycotton-api && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/model/creditor.go internal/repository/creditor.go internal/service/creditor.go internal/handler/creditor.go
git commit -m "feat: add creditor model, repository, service, handler"
```

---

### Task 6: Product Model + Repository + Service + Handler

**Files:**
- Create: `familycotton-api/internal/model/product.go`
- Create: `familycotton-api/internal/repository/product.go`
- Create: `familycotton-api/internal/service/product.go`
- Create: `familycotton-api/internal/handler/product.go`

Products are the most complex catalog entity — they have search filters (?search, ?supplier_id, ?brand), a computed margin field, and different RBAC rules (employee: read + create, owner: full CRUD).

- [ ] **Step 1: Create product model**

Write `familycotton-api/internal/model/product.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Product struct {
	ID           uuid.UUID       `json:"id"`
	SKU          string          `json:"sku"`
	Name         string          `json:"name"`
	Brand        *string         `json:"brand"`
	SupplierID   *uuid.UUID      `json:"supplier_id"`
	PhotoURL     *string         `json:"photo_url"`
	CostPrice    decimal.Decimal `json:"cost_price"`
	SellPrice    decimal.Decimal `json:"sell_price"`
	Margin       decimal.Decimal `json:"margin"`
	QtyShop      int             `json:"qty_shop"`
	QtyWarehouse int             `json:"qty_warehouse"`
	IsDeleted    bool            `json:"-"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CreateProductRequest struct {
	SKU        string          `json:"sku"`
	Name       string          `json:"name"`
	Brand      *string         `json:"brand"`
	SupplierID *uuid.UUID      `json:"supplier_id"`
	PhotoURL   *string         `json:"photo_url"`
	CostPrice  decimal.Decimal `json:"cost_price"`
	SellPrice  decimal.Decimal `json:"sell_price"`
}

func (r *CreateProductRequest) Validate() error {
	if r.SKU == "" {
		return NewAppError(ErrValidation, "sku is required")
	}
	if r.Name == "" {
		return NewAppError(ErrValidation, "name is required")
	}
	if r.CostPrice.IsNegative() {
		return NewAppError(ErrValidation, "cost_price cannot be negative")
	}
	if r.SellPrice.IsNegative() {
		return NewAppError(ErrValidation, "sell_price cannot be negative")
	}
	return nil
}

type UpdateProductRequest struct {
	SKU        *string          `json:"sku,omitempty"`
	Name       *string          `json:"name,omitempty"`
	Brand      *string          `json:"brand,omitempty"`
	SupplierID *uuid.UUID       `json:"supplier_id,omitempty"`
	PhotoURL   *string          `json:"photo_url,omitempty"`
	CostPrice  *decimal.Decimal `json:"cost_price,omitempty"`
	SellPrice  *decimal.Decimal `json:"sell_price,omitempty"`
}

func (r *UpdateProductRequest) Validate() error {
	if r.SKU != nil && *r.SKU == "" {
		return NewAppError(ErrValidation, "sku cannot be empty")
	}
	if r.Name != nil && *r.Name == "" {
		return NewAppError(ErrValidation, "name cannot be empty")
	}
	if r.CostPrice != nil && r.CostPrice.IsNegative() {
		return NewAppError(ErrValidation, "cost_price cannot be negative")
	}
	if r.SellPrice != nil && r.SellPrice.IsNegative() {
		return NewAppError(ErrValidation, "sell_price cannot be negative")
	}
	return nil
}

type ProductFilter struct {
	Search     string
	SupplierID *uuid.UUID
	Brand      string
}
```

- [ ] **Step 2: Create product repository**

Write `familycotton-api/internal/repository/product.go`:
```go
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/familycotton/api/internal/model"
)

type ProductRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, p *model.Product) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO products (id, sku, name, brand, supplier_id, photo_url, cost_price, sell_price)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at, updated_at`,
		p.ID, p.SKU, p.Name, p.Brand, p.SupplierID, p.PhotoURL, p.CostPrice, p.SellPrice,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if isDuplicateKey(err) {
			return model.NewAppError(model.ErrConflict, "sku already exists")
		}
		return err
	}
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	p := &model.Product{}
	err := r.db.QueryRow(ctx,
		`SELECT id, sku, name, brand, supplier_id, photo_url, cost_price, sell_price,
		        qty_shop, qty_warehouse, is_deleted, created_at, updated_at
		 FROM products WHERE id = $1 AND is_deleted = false`, id,
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

func (r *ProductRepository) List(ctx context.Context, filter model.ProductFilter, page, limit int) ([]model.Product, int, error) {
	where, args := buildProductFilter(filter)

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products WHERE is_deleted = false %s", where)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	args = append(args, limit, offset)
	listQuery := fmt.Sprintf(
		`SELECT id, sku, name, brand, supplier_id, photo_url, cost_price, sell_price,
		        qty_shop, qty_warehouse, is_deleted, created_at, updated_at
		 FROM products WHERE is_deleted = false %s
		 ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args),
	)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Brand, &p.SupplierID, &p.PhotoURL,
			&p.CostPrice, &p.SellPrice, &p.QtyShop, &p.QtyWarehouse,
			&p.IsDeleted, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		p.Margin = p.SellPrice.Sub(p.CostPrice)
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *ProductRepository) Update(ctx context.Context, p *model.Product) error {
	err := r.db.QueryRow(ctx,
		`UPDATE products SET sku=$1, name=$2, brand=$3, supplier_id=$4, photo_url=$5,
		        cost_price=$6, sell_price=$7, updated_at=NOW()
		 WHERE id=$8 AND is_deleted = false
		 RETURNING updated_at`,
		p.SKU, p.Name, p.Brand, p.SupplierID, p.PhotoURL, p.CostPrice, p.SellPrice, p.ID,
	).Scan(&p.UpdatedAt)
	if err != nil {
		if isDuplicateKey(err) {
			return model.NewAppError(model.ErrConflict, "sku already exists")
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return model.NewAppError(model.ErrNotFound, "product not found")
		}
		return err
	}
	return nil
}

func (r *ProductRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE products SET is_deleted = true, updated_at = NOW()
		 WHERE id = $1 AND is_deleted = false`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.NewAppError(model.ErrNotFound, "product not found")
	}
	return nil
}

func buildProductFilter(f model.ProductFilter) (string, []any) {
	var conditions []string
	var args []any
	idx := 1

	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR sku ILIKE $%d)", idx, idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	if f.SupplierID != nil {
		conditions = append(conditions, fmt.Sprintf("supplier_id = $%d", idx))
		args = append(args, *f.SupplierID)
		idx++
	}
	if f.Brand != "" {
		conditions = append(conditions, fmt.Sprintf("brand ILIKE $%d", idx))
		args = append(args, "%"+f.Brand+"%")
		idx++
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return "AND " + strings.Join(conditions, " AND "), args
}
```

- [ ] **Step 3: Create product service**

Write `familycotton-api/internal/service/product.go`:
```go
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
```

- [ ] **Step 4: Create product handler**

Write `familycotton-api/internal/handler/product.go`:
```go
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/service"
)

type ProductHandler struct {
	service *service.ProductService
}

func NewProductHandler(service *service.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationParams(r)

	filter := model.ProductFilter{
		Search: r.URL.Query().Get("search"),
		Brand:  r.URL.Query().Get("brand"),
	}
	if sid := r.URL.Query().Get("supplier_id"); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			filter.SupplierID = &id
		}
	}

	products, total, err := h.service.List(r.Context(), filter, page, limit)
	if err != nil {
		respondError(w, err)
		return
	}

	respondList(w, products, page, limit, total)
}

func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid product id"))
		return
	}

	product, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, product)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	product, err := h.service.Create(r.Context(), &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusCreated, product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid product id"))
		return
	}

	var req model.UpdateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	product, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, product)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, model.NewAppError(model.ErrValidation, "invalid product id"))
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, http.StatusOK, map[string]string{"message": "product deleted"})
}
```

- [ ] **Step 5: Verify compilation**

```bash
cd familycotton-api && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/model/product.go internal/repository/product.go internal/service/product.go internal/handler/product.go
git commit -m "feat: add product model, repository, service, handler with search filters"
```

---

### Task 7: Wire Routes + DI in Router and Main.go

**Files:**
- Modify: `familycotton-api/internal/router/router.go`
- Modify: `familycotton-api/cmd/api/main.go`

- [ ] **Step 1: Update router.go**

Add the new handler parameters and routes. The updated `router.go` should be:

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
	supplierHandler *handler.SupplierHandler,
	clientHandler *handler.ClientHandler,
	creditorHandler *handler.CreditorHandler,
	productHandler *handler.ProductHandler,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.CORS)
	r.Use(middleware.Logging)

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes.
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)

		// Protected routes.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authService))

			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/auth/me", authHandler.Me)

			// Users (owner only).
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequireRole("owner"))
				r.Get("/", userHandler.List)
				r.Post("/", userHandler.Create)
				r.Put("/{id}", userHandler.Update)
				r.Delete("/{id}", userHandler.Delete)
			})

			// Suppliers (employee: read only, owner: full CRUD).
			r.Route("/suppliers", func(r chi.Router) {
				r.Get("/", supplierHandler.List)
				r.Get("/{id}", supplierHandler.GetByID)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole("owner"))
					r.Post("/", supplierHandler.Create)
					r.Put("/{id}", supplierHandler.Update)
					r.Delete("/{id}", supplierHandler.Delete)
				})
			})

			// Clients (employee + owner).
			r.Route("/clients", func(r chi.Router) {
				r.Get("/", clientHandler.List)
				r.Get("/{id}", clientHandler.GetByID)
				r.Post("/", clientHandler.Create)
				r.Put("/", clientHandler.Update)
				r.Put("/{id}", clientHandler.Update)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole("owner"))
					r.Delete("/{id}", clientHandler.Delete)
				})
			})

			// Creditors (owner only).
			r.Route("/creditors", func(r chi.Router) {
				r.Use(middleware.RequireRole("owner"))
				r.Get("/", creditorHandler.List)
				r.Get("/{id}", creditorHandler.GetByID)
				r.Post("/", creditorHandler.Create)
				r.Put("/{id}", creditorHandler.Update)
				r.Delete("/{id}", creditorHandler.Delete)
			})

			// Products (employee: read + create, owner: full CRUD).
			r.Route("/products", func(r chi.Router) {
				r.Get("/", productHandler.List)
				r.Get("/{id}", productHandler.GetByID)
				r.Post("/", productHandler.Create)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole("owner"))
					r.Put("/{id}", productHandler.Update)
					r.Delete("/{id}", productHandler.Delete)
				})
			})
		})
	})

	return r
}
```

**Note:** In the clients route, there is a duplicate `r.Put("/", ...)` — remove it. The correct clients block should be:
```go
r.Route("/clients", func(r chi.Router) {
    r.Get("/", clientHandler.List)
    r.Get("/{id}", clientHandler.GetByID)
    r.Post("/", clientHandler.Create)
    r.Put("/{id}", clientHandler.Update)
    r.Group(func(r chi.Router) {
        r.Use(middleware.RequireRole("owner"))
        r.Delete("/{id}", clientHandler.Delete)
    })
})
```

- [ ] **Step 2: Update main.go DI wiring**

Add new repositories, services, and handlers in `cmd/api/main.go`. After the existing user wiring, add:

```go
// Repositories (add after tokenRepo).
supplierRepo := repository.NewSupplierRepository(pool)
clientRepo := repository.NewClientRepository(pool)
creditorRepo := repository.NewCreditorRepository(pool)
productRepo := repository.NewProductRepository(pool)

// Services (add after userService).
supplierService := service.NewSupplierService(supplierRepo)
clientService := service.NewClientService(clientRepo)
creditorService := service.NewCreditorService(creditorRepo)
productService := service.NewProductService(productRepo)

// Handlers (add after userHandler).
supplierHandler := handler.NewSupplierHandler(supplierService)
clientHandler := handler.NewClientHandler(clientService)
creditorHandler := handler.NewCreditorHandler(creditorService)
productHandler := handler.NewProductHandler(productService)
```

Update the `router.New(...)` call to pass all new handlers:
```go
r := router.New(authService, authHandler, userHandler,
    supplierHandler, clientHandler, creditorHandler, productHandler)
```

- [ ] **Step 3: Verify compilation**

```bash
cd familycotton-api && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/router/router.go cmd/api/main.go
git commit -m "feat: wire Phase 2 routes and DI for all catalog domains"
```

---

### Task 8: Integration Smoke Test

- [ ] **Step 1: Rebuild and start Docker**

```bash
cd familycotton-api && docker compose up -d --build
```

- [ ] **Step 2: Login and get token**

```bash
TOKEN=$(curl -s -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"login":"admin","password":"admin123"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
```

- [ ] **Step 3: Test Suppliers CRUD**

```bash
# Create
curl -s -X POST http://localhost:8082/api/v1/suppliers \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Cotton World","phone":"+998901234567","notes":"Main supplier"}'

# List with pagination
curl -s "http://localhost:8082/api/v1/suppliers?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"

# Get by ID (use ID from create response)
```

- [ ] **Step 4: Test Products CRUD with filters**

```bash
# Create product
curl -s -X POST http://localhost:8082/api/v1/products \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"sku":"FC-001","name":"Cotton T-Shirt","brand":"FamilyCotton","cost_price":"50000","sell_price":"80000"}'

# Search by name
curl -s "http://localhost:8082/api/v1/products?search=cotton" \
  -H "Authorization: Bearer $TOKEN"

# Filter by brand
curl -s "http://localhost:8082/api/v1/products?brand=Family" \
  -H "Authorization: Bearer $TOKEN"
```

- [ ] **Step 5: Test Clients and Creditors**

```bash
# Create client
curl -s -X POST http://localhost:8082/api/v1/clients \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"John Doe","phone":"+998909876543"}'

# Create creditor
curl -s -X POST http://localhost:8082/api/v1/creditors \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Bank ABC","phone":"+998901111111","notes":"Credit line"}'
```

- [ ] **Step 6: Test RBAC — employee read suppliers but cannot create**

```bash
# Login as employee
EMP_TOKEN=$(curl -s -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"login":"employee1","password":"pass123"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")

# Employee CAN read suppliers
curl -s "http://localhost:8082/api/v1/suppliers" -H "Authorization: Bearer $EMP_TOKEN"

# Employee CANNOT create supplier (expect 403)
curl -s -X POST http://localhost:8082/api/v1/suppliers \
  -H "Authorization: Bearer $EMP_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Blocked"}'

# Employee CAN create product
curl -s -X POST http://localhost:8082/api/v1/products \
  -H "Authorization: Bearer $EMP_TOKEN" -H "Content-Type: application/json" \
  -d '{"sku":"FC-002","name":"Cotton Pants","cost_price":"40000","sell_price":"70000"}'

# Employee CANNOT delete product (expect 403)
```

- [ ] **Step 7: Stop Docker**

```bash
docker compose down
```

- [ ] **Step 8: Commit if fixes needed**

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Add decimal dependency | go.mod |
| 2 | Supplier model + repository | model/supplier.go, repository/supplier.go |
| 3 | Supplier service + handler | service/supplier.go, handler/supplier.go |
| 4 | Client full stack | model/client.go, repository/client.go, service/client.go, handler/client.go |
| 5 | Creditor full stack | model/creditor.go, repository/creditor.go, service/creditor.go, handler/creditor.go |
| 6 | Product full stack (with search) | model/product.go, repository/product.go, service/product.go, handler/product.go |
| 7 | Wire routes + DI | router/router.go, cmd/api/main.go |
| 8 | Integration smoke test | manual curl tests |
