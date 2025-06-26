package db

import (
	"database/sql"
	"fmt"
	"go-bank-api/config"
	"go-bank-api/logger"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	cfg := config.AppConfig.Database

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)

	safeConnStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Name)

	logger.Log.WithField("connection", safeConnStr).Info("Attempting to connect to the database")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to open database connection")
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err = db.Ping(); err != nil {
		logger.Log.WithError(err).Error("Failed to ping database")
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Log.Info("Database connection established successfully")
	return db, nil
}
