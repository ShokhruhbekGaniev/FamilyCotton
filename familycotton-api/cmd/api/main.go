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
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/familycotton/api/internal/config"
	"github.com/familycotton/api/internal/handler"
	"github.com/familycotton/api/internal/repository"
	"github.com/familycotton/api/internal/router"
	"github.com/familycotton/api/internal/service"
	"github.com/familycotton/api/migrations"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

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

	if err := runMigrations(cfg.DBURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewTokenRepository(pool)

	authService := service.NewAuthService(userRepo, tokenRepo, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	userService := service.NewUserService(userRepo)

	authHandler := handler.NewAuthHandler(authService, userService)
	userHandler := handler.NewUserHandler(userService)

	// Phase 2 repositories.
	supplierRepo := repository.NewSupplierRepository(pool)
	clientRepo := repository.NewClientRepository(pool)
	creditorRepo := repository.NewCreditorRepository(pool)
	productRepo := repository.NewProductRepository(pool)

	// Phase 2 services.
	supplierService := service.NewSupplierService(supplierRepo)
	clientService := service.NewClientService(clientRepo)
	creditorService := service.NewCreditorService(creditorRepo)
	productService := service.NewProductService(productRepo)

	// Phase 2 handlers.
	supplierHandler := handler.NewSupplierHandler(supplierService)
	clientHandler := handler.NewClientHandler(clientService)
	creditorHandler := handler.NewCreditorHandler(creditorService)
	productHandler := handler.NewProductHandler(productService)

	// Phase 3 repositories.
	shiftRepo := repository.NewShiftRepository(pool)
	saleRepo := repository.NewSaleRepository(pool)
	saleReturnRepo := repository.NewSaleReturnRepository(pool)
	clientPaymentRepo := repository.NewClientPaymentRepository(pool)
	safeTransactionRepo := repository.NewSafeTransactionRepository(pool)
	ownerDebtRepo := repository.NewOwnerDebtRepository(pool)

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

	// Phase 4 repositories.
	purchaseOrderRepo := repository.NewPurchaseOrderRepository(pool)
	supplierPaymentRepo := repository.NewSupplierPaymentRepository(pool)
	creditorTransactionRepo := repository.NewCreditorTransactionRepository(pool)
	stockTransferRepo := repository.NewStockTransferRepository(pool)
	inventoryCheckRepo := repository.NewInventoryCheckRepository(pool)

	// Phase 4 services.
	purchaseOrderService := service.NewPurchaseOrderService(pool, purchaseOrderRepo, productRepo, supplierRepo, safeTransactionRepo)
	supplierPaymentService := service.NewSupplierPaymentService(pool, supplierPaymentRepo, purchaseOrderRepo, productRepo, supplierRepo, safeTransactionRepo)
	creditorTransactionService := service.NewCreditorTransactionService(pool, creditorTransactionRepo, creditorRepo, safeTransactionRepo)
	stockTransferService := service.NewStockTransferService(pool, stockTransferRepo, productRepo)
	inventoryCheckService := service.NewInventoryCheckService(pool, inventoryCheckRepo, productRepo)

	// Phase 4 handlers.
	purchaseOrderHandler := handler.NewPurchaseOrderHandler(purchaseOrderService)
	supplierPaymentHandler := handler.NewSupplierPaymentHandler(supplierPaymentService)
	creditorTransactionHandler := handler.NewCreditorTransactionHandler(creditorTransactionService)
	stockTransferHandler := handler.NewStockTransferHandler(stockTransferService)
	inventoryCheckHandler := handler.NewInventoryCheckHandler(inventoryCheckService)

	// Phase 5 repositories.
	dashboardRepo := repository.NewDashboardRepository(pool)

	// Phase 5 services.
	safeService := service.NewSafeService(pool, safeTransactionRepo, ownerDebtRepo)
	dashboardService := service.NewDashboardService(dashboardRepo)

	// Phase 5 handlers.
	safeHandler := handler.NewSafeHandler(safeService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)

	r := router.New(authService, authHandler, userHandler,
		supplierHandler, clientHandler, creditorHandler, productHandler,
		shiftHandler, saleHandler, saleReturnHandler, clientPaymentHandler,
		purchaseOrderHandler, supplierPaymentHandler, creditorTransactionHandler,
		stockTransferHandler, inventoryCheckHandler,
		safeHandler, dashboardHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

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
