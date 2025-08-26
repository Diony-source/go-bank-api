// handler/main_test.go
package handler

import (
	"database/sql"
	"fmt"
	"go-bank-api/config"
	"go-bank-api/logger"
	"go-bank-api/repository"
	"go-bank-api/router"
	"go-bank-api/service"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// TestApp holds dependencies for a test instance of the application.
type TestApp struct {
	Router http.Handler
	DB     *sql.DB
}

var testApp *TestApp

// TestMain sets up the test environment for the handler package.
func TestMain(m *testing.M) {
	logger.Init()
	config.LoadConfig("../") // Load config from root directory

	testDbConnStr := fmt.Sprintf("host=localhost port=5434 user=%s password=%s dbname=%s sslmode=disable",
		config.AppConfig.Database.User,
		config.AppConfig.Database.Password,
		config.AppConfig.Database.Name+"_test",
	)

	db, err := sql.Open("postgres", testDbConnStr)
	if err != nil {
		log.Fatalf("could not connect to test database: %v", err)
	}

	// Retry connection
	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("database not ready: %v", err)
	}

	runMigrations(testDbConnStr)

	testApp = setupTestApp(db)

	exitCode := m.Run()

	db.Close()
	os.Exit(exitCode)
}

// setupTestApp initializes the application for testing without starting the server.
func setupTestApp(db *sql.DB) *TestApp {
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := NewUserHandler(userRepo, userService)

	accountRepo := repository.NewAccountRepository(db)
	accountService := service.NewAccountService(accountRepo)
	accountHandler := NewAccountHandler(accountService)

	transactionRepo := repository.NewTransactionRepository(db)
	transactionService := service.NewTransactionService(db, accountRepo, transactionRepo)
	transactionHandler := NewTransactionHandler(transactionService)

	r := router.NewRouter(userHandler, accountHandler, transactionHandler)

	return &TestApp{
		Router: r,
		DB:     db,
	}
}

func runMigrations(connStr string) {
	migrationPath := "file://../db/migrations"
	mig, err := migrate.New(migrationPath, connStr)
	if err != nil {
		log.Fatalf("cannot create migrate instance: %v", err)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run migrate up: %v", err)
	}
}
