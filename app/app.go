package app

import (
	"context"
	"go-bank-api/config"
	"go-bank-api/db"
	"go-bank-api/handler"
	"go-bank-api/logger"
	"go-bank-api/repository"
	"go-bank-api/router"
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

	userRepo := repository.NewUserRepository(database)
	userHandler := handler.NewUserHandler(userRepo)
	r := router.NewRouter(userHandler)

	port := config.AppConfig.Server.Port
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		logger.Log.Infof("Starting server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatalf("Error starting server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Log.Warn("Shutdown signal received, Starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Log.Info("Server exited properly")
}
