// handler/main_test.go
package handler

import (
	"database/sql"
	"fmt"
	"go-bank-api/config"
	"go-bank-api/logger"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

var testDB *sql.DB

// TestMain sets up the test database for the handler package.
func TestMain(m *testing.M) {
	logger.Init()
	config.LoadConfig("../")

	testDbConnStr := fmt.Sprintf("postgres://%s:%s@localhost:5434/%s_test?sslmode=disable",
		config.AppConfig.Database.User,
		config.AppConfig.Database.Password,
		config.AppConfig.Database.Name,
	)

	var err error
	testDB, err = sql.Open("postgres", testDbConnStr)
	if err != nil {
		log.Fatalf("could not connect to test database: %v", err)
	}

	for i := 0; i < 5; i++ {
		err = testDB.Ping()
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("database not ready: %v", err)
	}

	runMigrations(testDbConnStr)

	exitCode := m.Run()

	testDB.Close()
	os.Exit(exitCode)
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
