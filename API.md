# FamilyCotton API Documentation

**Base URL:** `http://localhost:8082/api/v1`

**Auth:** All endpoints require `Authorization: Bearer <access_token>` header, except `POST /auth/login` and `POST /auth/refresh`.

**Roles:** `owner` (full access), `employee` (limited access — see tables below).

**Pagination:** List endpoints accept `?page=1&limit=20`. Response includes `meta: {page, limit, total}`.

**Response format:**
```json
// Success
{"data": {...}, "meta": {"page": 1, "limit": 20, "total": 100}}

// Error
{"error": {"code": "VALIDATION_ERROR", "message": "description"}}
```

**HTTP codes:** 200 (OK), 201 (Created), 400 (Validation), 401 (Unauthorized), 403 (Forbidden), 404 (Not Found), 500 (Internal)

---

## Auth

| Method | URL | Role | Body | Response `data` |
|--------|-----|------|------|-----------------|
| POST | `/auth/login` | public | `{login: string, password: string}` | `{access_token: string, refresh_token: string}` |
| POST | `/auth/refresh` | public | `{refresh_token: string}` | `{access_token: string, refresh_token: string}` |
| POST | `/auth/logout` | auth | `{refresh_token: string}` | `{message: "logged out"}` |
| GET | `/auth/me` | auth | — | `{id, name, login, role, created_at, updated_at}` |

**Default credentials:** `admin` / `admin123` (role: owner)

---

## Users

| Method | URL | Role | Query | Body | Response `data` |
|--------|-----|------|-------|------|-----------------|
| GET | `/users` | owner | — | — | `[{id, name, login, role, created_at, updated_at}]` |
| POST | `/users` | owner | — | `{name: string, login: string, password: string, role: "owner"\|"employee"}` | `{id, name, login, role, created_at, updated_at}` |
| PUT | `/users/:id` | owner | — | `{name?, login?, password?, role?}` | `{id, name, login, role, created_at, updated_at}` |
| DELETE | `/users/:id` | owner | — | — | `{message: "user deleted"}` |

---

## Suppliers

| Method | URL | Role | Query | Body | Response `data` |
|--------|-----|------|-------|------|-----------------|
| GET | `/suppliers` | auth | `?page&limit` | — | `[{id, name, phone, notes, total_debt, created_at, updated_at}]` + `meta` |
| GET | `/suppliers/:id` | auth | — | — | `{id, name, phone, notes, total_debt, created_at, updated_at}` |
| POST | `/suppliers` | owner | — | `{name: string, phone?: string, notes?: string}` | `{id, name, phone, notes, total_debt, created_at, updated_at}` |
| PUT | `/suppliers/:id` | owner | — | `{name?, phone?, notes?}` | `{id, name, phone, notes, total_debt, created_at, updated_at}` |
| DELETE | `/suppliers/:id` | owner | — | — | `{message: "supplier deleted"}` |

---

## Clients

| Method | URL | Role | Query | Body | Response `data` |
|--------|-----|------|-------|------|-----------------|
| GET | `/clients` | auth | `?page&limit` | — | `[{id, name, phone, total_debt, created_at, updated_at}]` + `meta` |
| GET | `/clients/:id` | auth | — | — | `{id, name, phone, total_debt, created_at, updated_at}` |
| POST | `/clients` | auth | — | `{name: string, phone?: string}` | `{id, name, phone, total_debt, created_at, updated_at}` |
| PUT | `/clients/:id` | auth | — | `{name?, phone?}` | `{id, name, phone, total_debt, created_at, updated_at}` |
| DELETE | `/clients/:id` | owner | — | — | `{message: "client deleted"}` |

---

## Creditors

| Method | URL | Role | Query | Body | Response `data` |
|--------|-----|------|-------|------|-----------------|
| GET | `/creditors` | owner | `?page&limit` | — | `[{id, name, phone, notes, total_debt, created_at, updated_at}]` + `meta` |
| GET | `/creditors/:id` | owner | — | — | `{id, name, phone, notes, total_debt, created_at, updated_at}` |
| POST | `/creditors` | owner | — | `{name: string, phone?: string, notes?: string}` | `{id, name, phone, notes, total_debt, created_at, updated_at}` |
| PUT | `/creditors/:id` | owner | — | `{name?, phone?, notes?}` | `{id, name, phone, notes, total_debt, created_at, updated_at}` |
| DELETE | `/creditors/:id` | owner | — | — | `{message: "creditor deleted"}` |

---

## Products

| Method | URL | Role | Query | Body | Response `data` |
|--------|-----|------|-------|------|-----------------|
| GET | `/products` | auth | `?page&limit&search&supplier_id&brand` | — | `[{id, sku, name, brand, supplier_id, photo_url, cost_price, sell_price, margin, qty_shop, qty_warehouse, created_at, updated_at}]` + `meta` |
| GET | `/products/:id` | auth | — | — | `{id, sku, name, brand, supplier_id, photo_url, cost_price, sell_price, margin, qty_shop, qty_warehouse, created_at, updated_at}` |
| POST | `/products` | auth | — | `{sku: string, name: string, brand?: string, supplier_id?: uuid, photo_url?: string, cost_price: decimal, sell_price: decimal}` | `{id, sku, name, ..., margin, qty_shop: 0, qty_warehouse: 0, ...}` |
| PUT | `/products/:id` | owner | — | `{sku?, name?, brand?, supplier_id?, photo_url?, cost_price?, sell_price?}` | `{id, sku, name, ..., margin, ...}` |
| DELETE | `/products/:id` | owner | — | — | `{message: "product deleted"}` |

`margin` is computed: `sell_price - cost_price`. `search` filters by name or SKU (ILIKE). `brand` filters by brand (ILIKE).

---

## Shifts

| Method | URL | Role | Query | Body | Response `data` |
|--------|-----|------|-------|------|-----------------|
| POST | `/shifts/open` | auth | — | — | `{id, opened_by, opened_at, closed_by: null, closed_at: null, total_cash: "0", total_terminal: "0", total_online: "0", total_debt_sales: "0", status: "open"}` |
| POST | `/shifts/close` | auth | — | — | `{id, opened_by, closed_by, opened_at, closed_at, total_cash, total_terminal, total_online, total_debt_sales, status: "closed"}` |
| GET | `/shifts/current` | auth | — | — | `{id, ..., status: "open"}` |
| GET | `/shifts` | auth | `?page&limit` | — | `[{...shift}]` + `meta` |

Shift close: aggregates all sales, creates safe_transactions (cash, terminal, online), creates owner_debt for online amount.

---

## Sales

| Method | URL | Role | Query | Body | Response `data` |
|--------|-----|------|-------|------|-----------------|
| POST | `/sales` | auth | — | `{client_id?: uuid, items: [{product_id: uuid, quantity: int, unit_price: decimal}], paid_cash: decimal, paid_terminal: decimal, paid_online: decimal, paid_debt: decimal}` | `{id, shift_id, client_id, total_amount, paid_cash, paid_terminal, paid_online, paid_debt, created_by, created_at, items: [{id, sale_id, product_id, quantity, unit_price, subtotal}]}` |
| GET | `/sales` | auth | `?page&limit&shift_id&client_id` | — | `[{id, shift_id, client_id, total_amount, paid_cash, paid_terminal, paid_online, paid_debt, created_by, created_at}]` + `meta` |
| GET | `/sales/:id` | auth | — | — | `{..., items: [{id, sale_id, product_id, quantity, unit_price, subtotal}]}` |

Validation: `paid_cash + paid_terminal + paid_online + paid_debt = total_amount`. If `paid_debt > 0`, `client_id` is required. Stock deducted per item (`qty_shop`). Client `total_debt` incremented if debt.

---

## Sale Returns

| Method | URL | Role | Query | Body | Response `data` |
|--------|-----|------|-------|------|-----------------|
| POST | `/sale-returns` | auth | — | `{sale_id: uuid, sale_item_id: uuid, new_product_id?: uuid, quantity: int, return_type: "full"\|"exchange"\|"exchange_diff"}` | `{id, sale_id, sale_item_id, new_product_id, quantity, return_type, refund_amount, surcharge_amount, created_by, created_at}` |
| GET | `/sale-returns` | auth | `?page&limit&sale_id` | — | `[{...}]` + `meta` |

- **full**: product returns to stock, refund proportionally from safe, client debt reduced if applicable
- **exchange**: old product returns, new product deducted from stock (`new_product_id` required)
- **exchange_diff**: exchange + money movement (refund if new cheaper, surcharge if new more expensive)

---

## Client Payments

| Method | URL | Role | Body | Response `data` |
|--------|-----|------|------|-----------------|
| POST | `/client-payments` | auth | `{client_id: uuid, amount: decimal, payment_method: "cash"\|"terminal"\|"online"}` | `{id, client_id, amount, payment_method, created_by, created_at}` |

Reduces `client.total_debt`, creates safe_transaction income.

---

## Purchase Orders

| Method | URL | Role | Query | Body | Response `data` |
|--------|-----|------|-------|------|-----------------|
| GET | `/purchase-orders` | owner | `?page&limit&supplier_id&status` | — | `[{id, supplier_id, total_amount, paid_amount, status, created_by, created_at, updated_at}]` + `meta` |
| GET | `/purchase-orders/:id` | owner | — | — | `{..., items: [{id, purchase_order_id, product_id, quantity, unit_cost, destination}]}` |
| POST | `/purchase-orders` | owner | — | `{supplier_id: uuid, items: [{product_id: uuid, quantity: int, unit_cost: decimal, destination: "shop"\|"warehouse"}], paid_amount: decimal}` | `{id, supplier_id, total_amount, paid_amount, status, ..., items: [...]}` |

`status`: `paid` (paid_amount = total), `partial` (0 < paid < total), `unpaid` (paid = 0). Stock incremented per item destination. Safe expense for paid_amount. Supplier debt for remainder.

---

## Supplier Payments

| Method | URL | Role | Body | Response `data` |
|--------|-----|------|------|-----------------|
| POST | `/supplier-payments` | owner | `{supplier_id: uuid, purchase_order_id?: uuid, payment_type: "money"\|"product_return", amount: decimal, returned_product_id?: uuid, returned_qty?: int}` | `{id, supplier_id, purchase_order_id, payment_type, amount, returned_product_id, returned_qty, created_by, created_at}` |

- **money**: safe expense, reduce supplier debt, update PO paid_amount/status
- **product_return**: deduct stock, reduce debt by `cost_price * qty`, update PO

---

## Creditor Transactions

| Method | URL | Role | Body | Response `data` |
|--------|-----|------|------|-----------------|
| POST | `/creditor-transactions` | owner | `{creditor_id: uuid, type: "receive"\|"repay", currency: "UZS"\|"USD", amount: decimal, exchange_rate: decimal}` | `{id, creditor_id, type, currency, amount, exchange_rate, amount_uzs, created_by, created_at}` |

`amount_uzs = amount * exchange_rate`. **receive**: safe income (cash) + increment creditor debt. **repay**: safe expense (cash) + reduce creditor debt.

---

## Stock

| Method | URL | Role | Body | Response `data` |
|--------|-----|------|------|-----------------|
| POST | `/stock/transfer` | owner | `{product_id: uuid, direction: "warehouse_to_shop"\|"shop_to_warehouse", quantity: int}` | `{id, product_id, direction, quantity, created_by, created_at}` |

---

## Inventory Checks

| Method | URL | Role | Body | Response `data` |
|--------|-----|------|------|-----------------|
| POST | `/inventory-checks` | owner | `{location: "shop"\|"warehouse"}` | `{id, location, checked_by, status: "in_progress", created_at, items: [{id, inventory_check_id, product_id, expected_qty, actual_qty: null, difference: null}]}` |
| PUT | `/inventory-checks/:id` | owner | `{items: [{item_id: uuid, actual_qty: int}], status?: "completed"}` | `{id, location, checked_by, status, created_at, completed_at, items: [...]}` |

Auto-generates items for all products on creation. When `status: "completed"`: verifies all items have `actual_qty`, auto-corrects stock.

---

## Safe

| Method | URL | Role | Query | Response `data` |
|--------|-----|------|-------|-----------------|
| GET | `/safe/balance` | owner | — | `{cash: decimal, terminal: decimal, online: decimal}` |
| GET | `/safe/transactions` | owner | `?page&limit` | `[{id, type, source, balance_type, amount, description, reference_id, created_at}]` + `meta` |
| GET | `/safe/owner-debts` | owner | — | `[{id, shift_id, amount, is_settled, created_at, settled_at}]` |
| POST | `/safe/owner-deposit` | owner | `{amount: decimal}` | `{message: "deposit recorded"}` |

`source` values: `shift`, `creditor_receive`, `creditor_repay`, `client_payment`, `client_refund`, `supplier_payment`, `purchase_cash`, `online_owner_debt`, `owner_deposit`

---

## Dashboard

| Method | URL | Role | Query | Response `data` |
|--------|-----|------|-------|-----------------|
| GET | `/dashboard/revenue` | owner | `?from=YYYY-MM-DD&to=YYYY-MM-DD` | `{total_revenue, cash, terminal, online, debt}` |
| GET | `/dashboard/profit` | owner | `?from=YYYY-MM-DD&to=YYYY-MM-DD` | `{total_revenue, total_cost, gross_profit}` |
| GET | `/dashboard/stock-value` | owner | — | `{total_cost_value, total_sell_value, total_items}` |
| GET | `/dashboard/sales-by-supplier` | owner | `?from=YYYY-MM-DD&to=YYYY-MM-DD` | `[{supplier_id, supplier_name, total_sales, items_sold}]` |
| GET | `/dashboard/paid-vs-debt` | owner | — | `{total_paid, total_debt}` |

---

## Data Types

- **uuid**: `"550e8400-e29b-41d4-a716-446655440000"`
- **decimal**: `"15000"` (string, never float)
- **timestamps**: `"2026-03-18T12:00:00Z"` (ISO 8601)
- **role**: `"owner"` | `"employee"`
- **shift status**: `"open"` | `"closed"`
- **purchase order status**: `"paid"` | `"partial"` | `"unpaid"`
- **return type**: `"full"` | `"exchange"` | `"exchange_diff"`
- **payment method**: `"cash"` | `"terminal"` | `"online"`
- **payment type** (supplier): `"money"` | `"product_return"`
- **direction** (stock): `"warehouse_to_shop"` | `"shop_to_warehouse"`
- **location** (inventory): `"shop"` | `"warehouse"`
- **currency**: `"UZS"` | `"USD"`
- **creditor tx type**: `"receive"` | `"repay"`
