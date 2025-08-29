// File: app/app.go
package app

import (
	"context"
	"database/sql"
	"go-bank-api/config"
	"go-bank-api/db"
	"go-bank-api/handler"
	"go-bank-api/logger"
	"go-bank-api/repository"
	"go-bank-api/router"
	"go-bank-api/service"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

// Run initializes and starts the application.
func Run() {
	config.LoadConfig(".")
	logger.Init()
	logger.Log.Info("Logger initialized")
	logger.Log.Info("Configuration loaded successfully")

	database, err := db.Connect()
	if err != nil {
		logger.Log.Fatalf("Error connecting to the database: %v", err)
	}
	defer database.Close()

	redisClient, err := db.ConnectRedis()
	if err != nil {
		logger.Log.Fatalf("Error connecting to Redis: %v", err)
	}
	defer redisClient.Close()

	// --- Dependency Injection ---
	userRepo := repository.NewUserRepository(database)
	tokenRepo := repository.NewTokenRepository(database)

	authService := service.NewAuthService(userRepo, tokenRepo)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userRepo, userService, authService)

	accountRepo := repository.NewAccountRepository(database)
	accountService := service.NewAccountService(accountRepo, redisClient)
	accountHandler := handler.NewAccountHandler(accountService)

	transactionRepo := repository.NewTransactionRepository(database)
	transactionService := service.NewTransactionService(database, accountRepo, transactionRepo)
	transactionHandler := handler.NewTransactionHandler(transactionService)

	r := router.NewRouter(userHandler, accountHandler, transactionHandler)

	// --- Server Initialization ---
	port := config.AppConfig.Server.Port
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		logger.Log.Infof("Server starting on port :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Warn("Shutdown signal received. Starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Log.Info("Server exited properly")
}

// TestApp holds dependencies for a test instance of the application.
type TestApp struct {
	Router      http.Handler
	DB          *sql.DB
	RedisClient *redis.Client // Mock client for test isolation.
}

// NewTestApp initializes the application for testing without starting the server.
// It accepts mockable dependencies for isolated testing.
func NewTestApp(db *sql.DB, redisClient *redis.Client) *TestApp {
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)

	authService := service.NewAuthService(userRepo, tokenRepo)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userRepo, userService, authService)

	accountRepo := repository.NewAccountRepository(db)
	// Inject the provided Redis client (real or mock) into the service layer.
	accountService := service.NewAccountService(accountRepo, redisClient)
	accountHandler := handler.NewAccountHandler(accountService)

	transactionRepo := repository.NewTransactionRepository(db)
	transactionService := service.NewTransactionService(db, accountRepo, transactionRepo)
	transactionHandler := handler.NewTransactionHandler(transactionService)

	r := router.NewRouter(userHandler, accountHandler, transactionHandler)

	return &TestApp{
		Router:      r,
		DB:          db,
		RedisClient: redisClient,
	}
}
