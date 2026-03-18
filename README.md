# FamilyCotton API

REST API backend for a textile retail business management system. Handles sales, inventory, purchases, supplier/creditor finances, and business analytics.

## Tech Stack

- **Go 1.25** with [chi v5](https://github.com/go-chi/chi) router
- **PostgreSQL 16** via [pgx v5](https://github.com/jackc/pgx)
- **JWT** authentication (access + refresh tokens)
- **Docker Compose** deployment
- Database migrations via [goose v3](https://github.com/pressly/goose)

## Quick Start

```bash
cd familycotton-api
cp .env.example .env
docker compose up -d --build
```

API will be available at `http://localhost:8082/api/v1`

**Default login:** `admin` / `admin123` (role: owner)

## Project Structure

```
familycotton-api/
├── cmd/api/main.go          # Entry point, dependency injection
├── internal/
│   ├── config/              # Environment configuration
│   ├── model/               # Data structures, request/response types
│   ├── repository/          # Database queries (pgx)
│   ├── service/             # Business logic
│   ├── handler/             # HTTP handlers
│   ├── middleware/           # Auth, RBAC, logging, CORS
│   └── router/              # Route definitions
├── migrations/              # SQL migrations (goose)
├── docker-compose.yml       # API + PostgreSQL
├── Dockerfile               # Multi-stage build
└── .env.example             # Environment template
```

## Architecture

Classic layered architecture: **Handler → Service → Repository**

- **Handlers** parse HTTP requests and return JSON responses
- **Services** contain business logic and manage database transactions
- **Repositories** execute SQL queries via pgx

Dependency injection is wired manually in `main.go`.

## Database

PostgreSQL 16 with 20 tables:

| Group | Tables |
|-------|--------|
| Auth | `users`, `refresh_tokens` |
| Catalogs | `suppliers`, `creditors`, `clients`, `products` |
| Sales | `shifts`, `sales`, `sale_items`, `sale_returns` |
| Procurement | `purchase_orders`, `purchase_order_items`, `supplier_payments` |
| Payments | `creditor_transactions`, `client_payments` |
| Stock | `stock_transfers`, `inventory_checks`, `inventory_check_items` |
| Finance | `safe_transactions`, `owner_debts` |

All tables use UUID primary keys, TIMESTAMPTZ for dates, DECIMAL(15,2) for money.

## API Endpoints

Full API documentation with request/response formats: **[API.md](API.md)**

### Summary (45 endpoints)

| Group | Endpoints | Access |
|-------|-----------|--------|
| Auth | Login, refresh, logout, me | Public / Auth |
| Users | CRUD | Owner |
| Suppliers | CRUD + pagination | Read: all, Write: owner |
| Clients | CRUD + pagination | All (delete: owner) |
| Creditors | CRUD + pagination | Owner |
| Products | CRUD + search + pagination | Read+create: all, Update/delete: owner |
| Shifts | Open, close, current, list | All |
| Sales | Create (split payment), list, details | All |
| Sale Returns | Full, exchange, exchange with diff | All |
| Client Payments | Debt repayment | All |
| Purchase Orders | Create with items, list, details | Owner |
| Supplier Payments | Money or product return | Owner |
| Creditor Transactions | Receive/repay with exchange rate | Owner |
| Stock Transfers | Warehouse ↔ Shop | Owner |
| Inventory Checks | Create, update, auto-correct | Owner |
| Safe | Balance, transactions, owner debts, deposit | Owner |
| Dashboard | Revenue, profit, stock value, analytics | Owner |

## Roles

| Feature | Owner | Employee |
|---------|-------|----------|
| Dashboard & Analytics | Yes | No |
| Safe & Finance | Yes | No |
| User Management | Yes | No |
| Suppliers (write) | Yes | No |
| Creditors | Yes | No |
| Purchase Orders | Yes | No |
| Stock Transfers | Yes | No |
| Inventory Checks | Yes | No |
| Products (update/delete) | Yes | No |
| Products (read/create) | Yes | Yes |
| Clients | Yes | Yes (no delete) |
| Suppliers (read) | Yes | Yes |
| Shifts & Sales | Yes | Yes |
| Sale Returns | Yes | Yes |
| Client Payments | Yes | Yes |

## Key Business Logic

### Sales Flow
1. Open shift → Create sales (split payment: cash/terminal/online/debt) → Close shift
2. Shift close aggregates payments, creates safe transactions, records owner debt for online payments
3. Stock deducted atomically with `SELECT FOR UPDATE` to prevent races

### Purchase Flow
1. Create purchase order with items → Stock arrives at shop/warehouse
2. Partial/full payment → Supplier debt tracked
3. Supplier payments: money (safe deduction) or product return (stock deduction at cost price)

### Returns
- **Full**: product returns to stock, proportional refund from safe
- **Exchange**: swap products
- **Exchange with diff**: swap + money movement for price difference

### Finance
- All monetary movements logged in `safe_transactions`
- Safe balance computed from transaction history (income - expense per type)
- Online payments create owner debt (owner's card receives payment, business owes owner)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `db` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `familycotton` | Database user |
| `DB_PASSWORD` | `familycotton_secret` | Database password |
| `DB_NAME` | `familycotton` | Database name |
| `JWT_SECRET` | `change-me-in-production` | JWT signing secret |
| `JWT_ACCESS_TTL` | `1h` | Access token lifetime |
| `JWT_REFRESH_TTL` | `720h` | Refresh token lifetime (30 days) |
| `SERVER_PORT` | `8082` | API server port |

## Docker

```bash
# Start
docker compose up -d --build

# Stop
docker compose down

# Logs
docker compose logs api -f

# Database shell
docker compose exec db psql -U familycotton
```

- API: `localhost:8082`
- PostgreSQL: `localhost:5434`
