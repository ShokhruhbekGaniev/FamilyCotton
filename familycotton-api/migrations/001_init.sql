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
    'Owner', 'admin', '$2a$10$yvQfOLtpdmm/PXACaiUcL.D94QCtgRPhQcBA274S3ON2qshFPlx2K', 'owner'
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
