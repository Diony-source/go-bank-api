// File: app/app.go
package app

import (
	"context"
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
)

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

	// --- Wiring All Layers Together ---
	// This section is crucial for dependency injection.
	// We create instances of our repositories, services, and handlers here.

	// Layers for User
	userRepo := repository.NewUserRepository(database)
	// NEW: Create the user service and pass the repository to it.
	userService := service.NewUserService(userRepo)
	// UPDATED: Pass both the repository and the new service to the user handler.
	userHandler := handler.NewUserHandler(userRepo, userService)

	// Layers for Account
	accountRepo := repository.NewAccountRepository(database)
	accountService := service.NewAccountService(accountRepo)
	accountHandler := handler.NewAccountHandler(accountService)

	// Start the router with all handlers
	r := router.NewRouter(userHandler, accountHandler)

	// --- Start the Server with Graceful Shutdown ---
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
