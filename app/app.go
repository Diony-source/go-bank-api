package app

import (
	"go-bank-api/db"
	"go-bank-api/handler"
	"go-bank-api/logger"
	"go-bank-api/repository"
	"go-bank-api/router"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func Run() {
	_ = godotenv.Load()
	logger.Init()

	database, err := db.Connect()
	if err != nil {
		logger.Log.Fatalf("Error connecting to the database: %v", err)
	}
	defer database.Close()

	userRepo := repository.NewUserRepository(database)
	userHandler := handler.NewUserHandler(userRepo)
	r := router.NewRouter(userHandler)

	logger.Log.Infof("Server started at :%s", os.Getenv("PORT"))
	logger.Log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), r))
}
