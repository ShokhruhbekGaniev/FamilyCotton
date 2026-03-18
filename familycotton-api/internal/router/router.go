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
	shiftHandler *handler.ShiftHandler,
	saleHandler *handler.SaleHandler,
	saleReturnHandler *handler.SaleReturnHandler,
	clientPaymentHandler *handler.ClientPaymentHandler,
	purchaseOrderHandler *handler.PurchaseOrderHandler,
	supplierPaymentHandler *handler.SupplierPaymentHandler,
	creditorTransactionHandler *handler.CreditorTransactionHandler,
	stockTransferHandler *handler.StockTransferHandler,
	inventoryCheckHandler *handler.InventoryCheckHandler,
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

			// Clients (employee + owner, delete owner only).
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
		})
	})

	return r
}
