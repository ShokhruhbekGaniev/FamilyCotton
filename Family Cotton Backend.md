# FamilyCotton — Backend ТЗ

**Go + PostgreSQL 16 + Docker**
**v1.0 — Март 2026**

---

## 1. Технический стек

| Компонент | Технология |
|-----------|-----------|
| Язык | Go 1.22+ |
| Роутер | chi v5 |
| БД | PostgreSQL 16 (pgx v5) |
| Миграции | goose |
| Аутентификация | JWT (access + refresh tokens) |
| Деплой | Docker Compose на VPS |
| Порт | 8082 (рядом с другими проектами) |

---

## 2. Структура проекта

```
familycotton-api/
├── cmd/api/main.go
├── internal/
│   ├── config/        ← конфигурация (env)
│   ├── handler/       ← HTTP-хендлеры
│   ├── middleware/     ← auth, cors, logging
│   ├── model/         ← структуры данных
│   ├── repository/    ← работа с БД (pgx)
│   ├── service/       ← бизнес-логика
│   └── router/        ← маршруты chi
├── migrations/            ← goose SQL-миграции
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

---

## 3. База данных (PostgreSQL 16)

Все таблицы используют UUID как первичный ключ, TIMESTAMPTZ для дат, DECIMAL(15,2) для денежных сумм.

### 3.1 users

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | Генерируется авто |
| name | VARCHAR(255) | Имя пользователя |
| login | VARCHAR(100) UNIQUE | Логин |
| password_hash | VARCHAR(255) | bcrypt хеш |
| role | VARCHAR(20) | `owner` \| `employee` |
| created_at | TIMESTAMPTZ | Дата создания |

### 3.2 suppliers

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| name | VARCHAR(255) | Название / имя |
| phone | VARCHAR(20) | Телефон |
| notes | TEXT | Заметки |
| total_debt | DECIMAL(15,2) | Текущий долг (computed) |
| created_at | TIMESTAMPTZ | |

### 3.3 creditors

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| name | VARCHAR(255) | Имя кредитора |
| phone | VARCHAR(20) | Телефон |
| notes | TEXT | Заметки |
| total_debt | DECIMAL(15,2) | Остаток долга в UZS |
| created_at | TIMESTAMPTZ | |

### 3.4 clients

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| name | VARCHAR(255) | ФИО |
| phone | VARCHAR(20) | Телефон |
| total_debt | DECIMAL(15,2) | Долг клиента |
| created_at | TIMESTAMPTZ | |

### 3.5 products

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| sku | VARCHAR(100) UNIQUE | Артикул |
| name | VARCHAR(255) | Название |
| brand | VARCHAR(255) | Бренд |
| supplier_id | UUID FK → suppliers | Поставщик |
| photo_url | TEXT | URL фото |
| cost_price | DECIMAL(15,2) | Себестоимость |
| sell_price | DECIMAL(15,2) | Цена продажи |
| qty_shop | INTEGER DEFAULT 0 | Остаток в магазине |
| qty_warehouse | INTEGER DEFAULT 0 | Остаток на складе |
| created_at | TIMESTAMPTZ | |

> Маржа считается на бекенде: `margin = sell_price - cost_price`. Не хранится в БД.

### 3.6 shifts

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| opened_by | UUID FK → users | Кто открыл |
| closed_by | UUID FK → users NULL | Кто закрыл |
| opened_at | TIMESTAMPTZ | Время открытия |
| closed_at | TIMESTAMPTZ NULL | Время закрытия |
| total_cash | DECIMAL(15,2) | Итог наличные |
| total_terminal | DECIMAL(15,2) | Итог терминал |
| total_online | DECIMAL(15,2) | Итог онлайн |
| total_debt_sales | DECIMAL(15,2) | Итог в долг |
| status | VARCHAR(20) | `open` \| `closed` |

### 3.7 sales

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| shift_id | UUID FK → shifts | Смена |
| client_id | UUID FK → clients NULL | Клиент (если в долг) |
| product_id | UUID FK → products | Товар |
| quantity | INTEGER | Количество |
| unit_price | DECIMAL(15,2) | Цена за шт |
| total_amount | DECIMAL(15,2) | Общая сумма |
| paid_cash | DECIMAL(15,2) | Оплата наличными |
| paid_terminal | DECIMAL(15,2) | Оплата терминалом |
| paid_online | DECIMAL(15,2) | Оплата онлайн |
| paid_debt | DECIMAL(15,2) | В долг |
| created_at | TIMESTAMPTZ | |

> Валидация: `paid_cash + paid_terminal + paid_online + paid_debt = total_amount`

### 3.8 sale_returns

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| sale_id | UUID FK → sales | Оригинальная продажа |
| product_id | UUID FK → products | Возвращаемый товар |
| new_product_id | UUID FK → products NULL | Новый товар (обмен) |
| quantity | INTEGER | Кол-во возврата |
| return_type | VARCHAR(20) | `full` \| `exchange` \| `exchange_diff` |
| refund_amount | DECIMAL(15,2) | Сумма возврата клиенту |
| surcharge_amount | DECIMAL(15,2) | Доплата клиента |
| created_at | TIMESTAMPTZ | |

### 3.9 purchase_orders

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| supplier_id | UUID FK → suppliers | Поставщик |
| payment_type | VARCHAR(20) | `cash` \| `debt` |
| total_amount | DECIMAL(15,2) | Общая сумма закупки |
| paid_amount | DECIMAL(15,2) | Оплачено |
| status | VARCHAR(20) | `paid` \| `partial` \| `unpaid` |
| created_at | TIMESTAMPTZ | |

### 3.10 purchase_order_items

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| purchase_order_id | UUID FK | Закупка |
| product_id | UUID FK → products | Товар |
| quantity | INTEGER | Количество |
| unit_cost | DECIMAL(15,2) | Цена за шт |
| destination | VARCHAR(20) | `shop` \| `warehouse` |

### 3.11 supplier_payments

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| supplier_id | UUID FK | Поставщик |
| purchase_order_id | UUID FK NULL | К какой закупке |
| payment_type | VARCHAR(20) | `money` \| `product_return` |
| amount | DECIMAL(15,2) | Сумма |
| returned_product_id | UUID FK NULL | Товар (если возврат) |
| returned_qty | INTEGER NULL | Кол-во возврата |
| created_at | TIMESTAMPTZ | |

### 3.12 creditor_transactions

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| creditor_id | UUID FK | Кредитор |
| type | VARCHAR(20) | `receive` \| `repay` |
| currency | VARCHAR(3) | `UZS` \| `USD` |
| amount | DECIMAL(15,2) | Сумма в оригинальной валюте |
| exchange_rate | DECIMAL(15,4) | Курс на момент |
| amount_uzs | DECIMAL(15,2) | Сумма в UZS |
| created_at | TIMESTAMPTZ | |

### 3.13 client_payments

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| client_id | UUID FK | Клиент |
| amount | DECIMAL(15,2) | Сумма оплаты |
| payment_method | VARCHAR(20) | `cash` \| `terminal` \| `online` |
| created_at | TIMESTAMPTZ | |

### 3.14 stock_transfers

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| product_id | UUID FK | Товар |
| direction | VARCHAR(30) | `warehouse_to_shop` \| `shop_to_warehouse` |
| quantity | INTEGER | Количество |
| created_at | TIMESTAMPTZ | |

### 3.15 safe_transactions

Главный лог всех денежных движений в сейфе:

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| type | VARCHAR(20) | `income` \| `expense` \| `transfer` |
| source | VARCHAR(50) | Источник: `shift`, `creditor`, `client_payment`, `supplier_payment`, `creditor_repay`, `client_refund`, `online_owner_debt`, `owner_deposit` |
| balance_type | VARCHAR(20) | `cash` \| `terminal` \| `online` |
| amount | DECIMAL(15,2) | Сумма |
| description | TEXT | Описание |
| reference_id | UUID NULL | Ссылка на сущность |
| created_at | TIMESTAMPTZ | |

### 3.16 owner_debts

Долг бизнеса владельцу карты за онлайн-оплаты:

| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID PK | |
| shift_id | UUID FK | Смена |
| amount | DECIMAL(15,2) | Сумма онлайн-оплат |
| is_settled | BOOLEAN DEFAULT false | Погашен ли |
| created_at | TIMESTAMPTZ | |
| settled_at | TIMESTAMPTZ NULL | Дата погашения |

### 3.17 inventory_checks + inventory_check_items

Ревизия товаров. Структура: `location` (shop|warehouse), `checked_by`, `status` (in_progress|completed). Items: `expected_qty`, `actual_qty`, `difference`.

---

## 4. API эндпоинты (REST)

Базовый URL: `/api/v1`. Все эндпоинты защищены JWT (кроме `/auth/login`). Формат: JSON.

### 4.1 Auth

| Метод | URL | Описание |
|-------|-----|----------|
| POST | /auth/login | Вход (login + password) → JWT |
| POST | /auth/refresh | Обновить токен |
| GET | /auth/me | Текущий пользователь |

### 4.2 Users

| Метод | URL | Описание |
|-------|-----|----------|
| GET | /users | Список |
| POST | /users | Создать |
| PUT | /users/:id | Обновить |
| DELETE | /users/:id | Удалить |

### 4.3 Products

| Метод | URL | Описание |
|-------|-----|----------|
| GET | /products | Список (?search, ?supplier_id, ?brand) |
| GET | /products/:id | Детали |
| POST | /products | Создать |
| PUT | /products/:id | Обновить |
| DELETE | /products/:id | Удалить |

### 4.4 Suppliers

| Метод | URL | Описание |
|-------|-----|----------|
| GET | /suppliers | Список |
| GET | /suppliers/:id | Детали + взаиморасчёт |
| POST | /suppliers | Создать |
| PUT | /suppliers/:id | Обновить |
| DELETE | /suppliers/:id | Удалить |

### 4.5 Purchase Orders (закупки)

| Метод | URL | Описание |
|-------|-----|----------|
| GET | /purchase-orders | Список (?supplier_id, ?status) |
| POST | /purchase-orders | Создать (с items) |
| POST | /supplier-payments | Оплата поставщику (деньгами / товаром) |

### 4.6 Creditors

| Метод | URL | Описание |
|-------|-----|----------|
| GET | /creditors | Список |
| GET | /creditors/:id | Детали + история транзакций |
| POST | /creditors | Создать |
| POST | /creditor-transactions | Получить / вернуть деньги |

### 4.7 Clients

| Метод | URL | Описание |
|-------|-----|----------|
| GET | /clients | Список |
| GET | /clients/:id | Детали + покупки + долги |
| POST | /clients | Создать |
| POST | /client-payments | Оплата долга |

### 4.8 Shifts + Sales

| Метод | URL | Описание |
|-------|-----|----------|
| POST | /shifts/open | Открыть смену |
| POST | /shifts/close | Закрыть смену (подсчёт + перевод в сейф + долг владельцу) |
| GET | /shifts/current | Текущая смена |
| GET | /shifts | История смен |
| POST | /sales | Создать продажу (сплит-оплата) |
| POST | /sale-returns | Возврат товара |

### 4.9 Stock

| Метод | URL | Описание |
|-------|-----|----------|
| POST | /stock/transfer | Переместить товар (склад ↔ магазин) |
| GET | /stock/shop | Остатки магазина |
| GET | /stock/warehouse | Остатки склада |
| POST | /inventory-checks | Начать ревизию |
| PUT | /inventory-checks/:id | Обновить / завершить |

### 4.10 Safe (сейф)

| Метод | URL | Описание |
|-------|-----|----------|
| GET | /safe/balance | Балансы: нал / терминал / онлайн |
| GET | /safe/transactions | История движений |
| POST | /safe/owner-deposit | Внести наличные (погашение долга владельца) |

### 4.11 Dashboard

| Метод | URL | Описание |
|-------|-----|----------|
| GET | /dashboard/revenue?from&to | Выручка за период |
| GET | /dashboard/profit?from&to | Прибыль за период |
| GET | /dashboard/stock-value | Остаток товара (сумма) |
| GET | /dashboard/sales-by-supplier?from&to | Продажи по поставщикам |
| GET | /dashboard/paid-vs-debt | Оплачено vs в долг клиентам |

---

## 5. Ключевая бизнес-логика

### 5.1 Закрытие смены

1. Подсчитать суммы по видам оплаты за смену
2. Записать `total_cash`, `total_terminal`, `total_online`, `total_debt_sales` в shifts
3. Создать safe_transactions: наличные → cash, терминал → terminal
4. Онлайн: перевести в сейф как cash + создать `owner_debts` запись
5. Установить `status = closed`

### 5.2 Продажа товара

1. Проверить: смена открыта, товар в наличии (`qty_shop >= quantity`)
2. Валидация: `paid_cash + paid_terminal + paid_online + paid_debt = total_amount`
3. Списать `qty_shop`
4. Если `paid_debt > 0`: `client_id` обязателен, увеличить `client.total_debt`
5. Создать запись в sales

### 5.3 Возврат от клиента

1. **Полный**: вернуть товар на `qty_shop`, списать сумму из сейфа
2. **Обмен**: вернуть старый товар на `qty_shop`, списать новый товар с `qty_shop`
3. **Обмен с разницей**: то же + движение денег в сейфе
4. Если продажа была в долг: уменьшить `client.total_debt`

### 5.4 Закупка у поставщика

1. Создать `purchase_order` с items
2. Приходовать товары: `qty_shop` или `qty_warehouse` по `destination`
3. Если cash: списать из сейфа, `status = paid`
4. Если debt: увеличить `supplier.total_debt`, `status = unpaid`

### 5.5 Оплата поставщику

1. **Деньгами**: списать из сейфа, уменьшить `supplier.total_debt`, обновить `purchase_order.paid_amount`
2. **Товаром**: списать товар с остатков, уменьшить долг на `себестоимость × кол-во`
3. Пересчитать `purchase_order.status` (`paid` | `partial` | `unpaid`)

### 5.6 Кредиторы

1. **Получение**: фиксация валюты + курса, пополнить сейф (cash), увеличить `creditor.total_debt`
2. **Возврат**: фиксация курса, списать из сейфа, уменьшить `creditor.total_debt`

---

## 6. Docker Compose

Деплой на VPS рядом с другими проектами. PostgreSQL на отдельном порту (5434), API на 8082. Отдельная docker network. Nginx upstream для проксирования.
