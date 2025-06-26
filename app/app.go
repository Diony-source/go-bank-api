package app

import (
	"go-bank-api/config"
	"go-bank-api/db"
	"go-bank-api/handler"
	"go-bank-api/logger"
	"go-bank-api/repository"
	"go-bank-api/router"
	"net/http"
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

	port := config.AppConfig.Server.Port // Konfigürasyonu buradan al
	logger.Log.Infof("Server starting on port :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Log.Fatalf("Failed to start server: %v", err)
	}
}
