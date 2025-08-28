// file: router/router_test.go

package router_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-bank-api/app"
	"go-bank-api/config"
	"go-bank-api/logger"
	"go-bank-api/model"
	"go-bank-api/repository"
	"go-bank-api/service"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var testApp *app.TestApp
var authService *service.AuthService

func TestMain(m *testing.M) {
	logger.Init()
	config.LoadConfig("../")
	authService = service.NewAuthService(nil, nil)

	testDbConnStr := fmt.Sprintf("postgres://%s:%s@localhost:5434/%s_test?sslmode=disable",
		config.AppConfig.Database.User,
		config.AppConfig.Database.Password,
		config.AppConfig.Database.Name,
	)

	db, err := sql.Open("postgres", testDbConnStr)
	if err != nil {
		log.Fatalf("could not connect to test database: %v", err)
	}

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
	testApp = app.NewTestApp(db)
	exitCode := m.Run()
	db.Close()
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

// --- Test Helper Functions ---

func createUserForTest(t *testing.T, username, email, password string) model.User {
	hashedPassword, _ := authService.HashPassword(password)
	user := model.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
	}
	err := testApp.DB.QueryRow(
		`INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id`,
		user.Username, user.Email, user.Password,
	).Scan(&user.ID)
	assert.NoError(t, err)
	return user
}

func createUserWithRoleForTest(t *testing.T, username, email, password string, role model.Role) model.User {
	hashedPassword, _ := authService.HashPassword(password)
	user := model.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
		Role:     string(role),
	}
	err := testApp.DB.QueryRow(
		`INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, $4) RETURNING id`,
		user.Username, user.Email, user.Password, user.Role,
	).Scan(&user.ID)
	assert.NoError(t, err)
	return user
}

// loginUserForTest is a helper to log in.
// It now correctly unmarshals the new TokenPair response and returns only the access token for use in subsequent API calls.
func loginUserForTest(t *testing.T, email, password string) string {
	requestBody := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, email, password)
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	testApp.Router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, "Login request should be successful")

	var response service.TokenPair // Use the actual TokenPair struct for unmarshaling.
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err, "Should be able to unmarshal login response")

	// We are now looking for "access_token", not "token".
	assert.NotEmpty(t, response.AccessToken, "Access Token should not be empty on successful login")
	return response.AccessToken
}

func cleanupUser(t *testing.T, email string) {
	_, err := testApp.DB.Exec("DELETE FROM users WHERE email = $1", email)
	assert.NoError(t, err, "Failed to clean up user")
}

func createAccountForTest(t *testing.T, userID int, currency string) model.Account {
	accountService := service.NewAccountService(repository.NewAccountRepository(testApp.DB))
	account, err := accountService.CreateNewAccount(userID, currency)
	assert.NoError(t, err)
	return *account
}

// --- Test Suites ---

func TestHealthCheck_Integration(t *testing.T) {
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	testApp.Router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	expectedBody := `{"status":"API is healthy and running"}`
	assert.JSONEq(t, expectedBody, rr.Body.String())
}

func TestRegister_Integration(t *testing.T) {
	requestBody := `{"username":"integration_test_user","email":"integration@test.com","password":"password123"}`
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(requestBody))
	rr := httptest.NewRecorder()
	defer cleanupUser(t, "integration@test.com")

	testApp.Router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var username string
	err := testApp.DB.QueryRow("SELECT username FROM users WHERE email = $1", "integration@test.com").Scan(&username)
	assert.NoError(t, err)
	assert.Equal(t, "integration_test_user", username)
}

func TestLogin_Integration(t *testing.T) {
	email := "login.test@example.com"
	password := "password123"
	createUserForTest(t, "login_test_user", email, password)
	defer cleanupUser(t, email)

	t.Run("successful login", func(t *testing.T) {
		requestBody := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, email, password)
		req, _ := http.NewRequest("POST", "/login", strings.NewReader(requestBody))
		rr := httptest.NewRecorder()
		testApp.Router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response service.TokenPair
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
	})

	t.Run("wrong password", func(t *testing.T) {
		requestBody := fmt.Sprintf(`{"email": "%s", "password": "wrongpassword"}`, email)
		req, _ := http.NewRequest("POST", "/login", strings.NewReader(requestBody))
		rr := httptest.NewRecorder()
		testApp.Router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestCreateAccount_Integration(t *testing.T) {
	email := "account.test@example.com"
	password := "password123"
	user := createUserForTest(t, "account_test_user", email, password)
	defer cleanupUser(t, user.Email)
	token := loginUserForTest(t, user.Email, password)

	t.Run("success", func(t *testing.T) {
		requestBody := `{"currency": "USD"}`
		req, _ := http.NewRequest("POST", "/api/accounts", strings.NewReader(requestBody))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		testApp.Router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)

		var currency string
		err := testApp.DB.QueryRow("SELECT currency FROM accounts WHERE user_id = $1", user.ID).Scan(&currency)
		assert.NoError(t, err, "Account should be created in the database")
		assert.Equal(t, "USD", currency)
	})
}

func TestTransfer_Integration(t *testing.T) {
	sender := createUserForTest(t, "sender", "sender@test.com", "password123")
	receiver := createUserForTest(t, "receiver", "receiver@test.com", "password123")
	defer cleanupUser(t, sender.Email)
	defer cleanupUser(t, receiver.Email)

	senderAccount := createAccountForTest(t, sender.ID, "TRY")
	receiverAccount := createAccountForTest(t, receiver.ID, "TRY")
	_, err := testApp.DB.Exec("UPDATE accounts SET balance = 500 WHERE id = $1", senderAccount.ID)
	assert.NoError(t, err)
	senderToken := loginUserForTest(t, sender.Email, "password123")

	t.Run("successful transfer", func(t *testing.T) {
		amount := 150.75
		requestBody := fmt.Sprintf(`{"from_account_id": %d, "to_account_id": %d, "amount": %.2f}`, senderAccount.ID, receiverAccount.ID, amount)
		req, _ := http.NewRequest("POST", "/api/transfers", strings.NewReader(requestBody))
		req.Header.Set("Authorization", "Bearer "+senderToken)
		rr := httptest.NewRecorder()
		testApp.Router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)
	})
}

func TestAdminRoutes_Integration(t *testing.T) {
	adminUser := createUserWithRoleForTest(t, "admin_user", "admin@test.com", "password123", model.RoleAdmin)
	regularUser := createUserWithRoleForTest(t, "regular_user", "user@test.com", "password123", model.RoleUser)
	defer cleanupUser(t, adminUser.Email)
	defer cleanupUser(t, regularUser.Email)
	adminToken := loginUserForTest(t, adminUser.Email, "password123")
	userToken := loginUserForTest(t, regularUser.Email, "password123")
	endpoint := "/api/admin/users"

	t.Run("admin can access admin routes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", endpoint, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rr := httptest.NewRecorder()
		testApp.Router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("regular user is forbidden from admin routes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", endpoint, nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		rr := httptest.NewRecorder()
		testApp.Router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

func TestAuthFlows_Integration(t *testing.T) {
	email := "authflow@test.com"
	password := "password123"
	user := createUserForTest(t, "authflow_user", email, password)
	defer cleanupUser(t, user.Email)

	// 1. Login to get initial token pair
	loginBody := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, email, password)
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(loginBody))
	rr := httptest.NewRecorder()
	testApp.Router.ServeHTTP(rr, req)
	var loginResponse service.TokenPair
	err := json.Unmarshal(rr.Body.Bytes(), &loginResponse)
	assert.NoError(t, err)
	initialAccessToken := loginResponse.AccessToken

	// --- To prevent flaky tests, wait for 1 second to ensure the next token's timestamp is different ---
	time.Sleep(1 * time.Second)

	// --- Test 1: Successful Token Refresh ---
	t.Run("successful token refresh", func(t *testing.T) {
		refreshBody := fmt.Sprintf(`{"refresh_token": "%s"}`, loginResponse.RefreshToken)
		req, _ := http.NewRequest("POST", "/api/token/refresh", strings.NewReader(refreshBody))
		rr := httptest.NewRecorder()
		testApp.Router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var refreshResponse struct {
			AccessToken string `json:"access_token"`
		}
		err := json.Unmarshal(rr.Body.Bytes(), &refreshResponse)
		assert.NoError(t, err)
		assert.NotEmpty(t, refreshResponse.AccessToken)
		assert.NotEqual(t, initialAccessToken, refreshResponse.AccessToken, "New access token should be different")
	})

	// --- Test 2: Logout ---
	t.Run("successful logout", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/logout", nil)
		req.Header.Set("Authorization", "Bearer "+initialAccessToken)
		rr := httptest.NewRecorder()
		testApp.Router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify refresh token is now invalid
		refreshBody := fmt.Sprintf(`{"refresh_token": "%s"}`, loginResponse.RefreshToken)
		req, _ = http.NewRequest("POST", "/api/token/refresh", strings.NewReader(refreshBody))
		rr = httptest.NewRecorder()
		testApp.Router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code, "Refresh token should be invalid after logout")
	})
}
