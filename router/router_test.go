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
var authService *service.AuthService // A helper auth service instance for test setup

// TestMain sets up the test environment for the router package.
func TestMain(m *testing.M) {
	logger.Init()
	config.LoadConfig("../")

	// Instantiate a dummy authService for password hashing in test helpers.
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

// TestHealthCheck_Integration tests the health check endpoint.
func TestHealthCheck_Integration(t *testing.T) {
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	testApp.Router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	expectedBody := `{"status":"API is healthy and running"}`
	assert.JSONEq(t, expectedBody, rr.Body.String())
}

// TestRegister_Integration tests the user registration endpoint.
func TestRegister_Integration(t *testing.T) {
	// Setup
	requestBody := `{
		"username": "integration_test_user",
		"email": "integration@test.com",
		"password": "password123"
	}`
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Teardown: ensure the test user is removed after the test.
	defer testApp.DB.Exec("DELETE FROM users WHERE email = $1", "integration@test.com")

	// Execute
	testApp.Router.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Verify database state
	var username string
	err := testApp.DB.QueryRow("SELECT username FROM users WHERE email = $1", "integration@test.com").Scan(&username)
	assert.NoError(t, err)
	assert.Equal(t, "integration_test_user", username)
}

// TestLogin_Integration tests the user login endpoint.
func TestLogin_Integration(t *testing.T) {
	// Setup: Create a user directly in the database to test against.
	email := "login.test@example.com"
	password := "password123"
	hashedPassword, _ := authService.HashPassword(password)

	_, err := testApp.DB.Exec(`INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`, "login_test_user", email, hashedPassword)
	assert.NoError(t, err)

	defer testApp.DB.Exec("DELETE FROM users WHERE email = $1", email)

	t.Run("successful login", func(t *testing.T) {
		requestBody := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, email, password)
		req, _ := http.NewRequest("POST", "/login", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Execute
		testApp.Router.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]string
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["token"], "Token should not be empty on successful login")
	})

	t.Run("wrong password", func(t *testing.T) {
		requestBody := fmt.Sprintf(`{"email": "%s", "password": "wrongpassword"}`, email)
		req, _ := http.NewRequest("POST", "/login", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Execute
		testApp.Router.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

// TestCreateAccount_Integration tests the account creation endpoint.
func TestCreateAccount_Integration(t *testing.T) {
	// Setup: Create a user to own the account.
	email := "account.test@example.com"
	password := "password123"
	user := createUserForTest(t, "account_test_user", email, password)
	defer cleanupUser(t, user.Email) // Ensure user is deleted after tests.

	// Login to get a valid token.
	token := loginUserForTest(t, user.Email, password)

	t.Run("success", func(t *testing.T) {
		requestBody := `{"currency": "USD"}`
		req, _ := http.NewRequest("POST", "/api/accounts", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token) // Use the acquired token.
		rr := httptest.NewRecorder()

		// Execute
		testApp.Router.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Verify database state
		var currency string
		err := testApp.DB.QueryRow("SELECT currency FROM accounts WHERE user_id = $1", user.ID).Scan(&currency)
		assert.NoError(t, err, "Account should be created in the database")
		assert.Equal(t, "USD", currency)
	})

	t.Run("unauthorized no token", func(t *testing.T) {
		requestBody := `{"currency": "EUR"}`
		req, _ := http.NewRequest("POST", "/api/accounts", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header is sent.
		rr := httptest.NewRecorder()

		// Execute
		testApp.Router.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

// --- Test Helper Functions ---

// createUserForTest is a helper to create a user and return the model.
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

// loginUserForTest is a helper to log in and return a JWT token.
func loginUserForTest(t *testing.T, email, password string) string {
	requestBody := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, email, password)
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	testApp.Router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	token, ok := response["token"]
	assert.True(t, ok, "Token should be in the login response")
	return token
}

// cleanupUser is a helper to delete a user after a test.
func cleanupUser(t *testing.T, email string) {
	_, err := testApp.DB.Exec("DELETE FROM users WHERE email = $1", email)
	assert.NoError(t, err, "Failed to clean up user")
}

// TestTransfer_Integration tests the money transfer endpoint.
func TestTransfer_Integration(t *testing.T) {
	// --- Setup ---
	// 1. Create Sender and Receiver
	sender := createUserForTest(t, "sender", "sender@test.com", "password123")
	receiver := createUserForTest(t, "receiver", "receiver@test.com", "password123")
	defer cleanupUser(t, sender.Email)
	defer cleanupUser(t, receiver.Email)

	// 2. Create Accounts for both
	senderAccount := createAccountForTest(t, sender.ID, "TRY")
	receiverAccount := createAccountForTest(t, receiver.ID, "TRY")

	// 3. Fund the sender's account directly in the DB for the test
	_, err := testApp.DB.Exec("UPDATE accounts SET balance = 500 WHERE id = $1", senderAccount.ID)
	assert.NoError(t, err)

	// 4. Login as the Sender to get a token
	senderToken := loginUserForTest(t, sender.Email, "password123")

	t.Run("successful transfer", func(t *testing.T) {
		// --- Action ---
		amount := 150.75
		requestBody := fmt.Sprintf(`{"from_account_id": %d, "to_account_id": %d, "amount": %.2f}`, senderAccount.ID, receiverAccount.ID, amount)
		req, _ := http.NewRequest("POST", "/api/transfers", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+senderToken)
		rr := httptest.NewRecorder()

		testApp.Router.ServeHTTP(rr, req)

		// --- Assertions ---
		assert.Equal(t, http.StatusCreated, rr.Code)

		// --- Verification ---
		// Check sender's balance
		var senderBalance float64
		err = testApp.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", senderAccount.ID).Scan(&senderBalance)
		assert.NoError(t, err)
		assert.Equal(t, 349.25, senderBalance) // 500 - 150.75

		// Check receiver's balance
		var receiverBalance float64
		err = testApp.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", receiverAccount.ID).Scan(&receiverBalance)
		assert.NoError(t, err)
		assert.Equal(t, 150.75, receiverBalance) // 0 + 150.75

		// Check if transaction record was created
		var transactionCount int
		err = testApp.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE from_account_id = $1", senderAccount.ID).Scan(&transactionCount)
		assert.NoError(t, err)
		assert.Equal(t, 1, transactionCount)

		// Cleanup for this specific sub-test
		_, _ = testApp.DB.Exec("UPDATE accounts SET balance = 500 WHERE id = $1", senderAccount.ID)
		_, _ = testApp.DB.Exec("UPDATE accounts SET balance = 0 WHERE id = $1", receiverAccount.ID)
		_, _ = testApp.DB.Exec("DELETE FROM transactions")
	})

	t.Run("insufficient funds", func(t *testing.T) {
		// --- Action ---
		amount := 9999.0 // More than the sender has
		requestBody := fmt.Sprintf(`{"from_account_id": %d, "to_account_id": %d, "amount": %.2f}`, senderAccount.ID, receiverAccount.ID, amount)
		req, _ := http.NewRequest("POST", "/api/transfers", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+senderToken)
		rr := httptest.NewRecorder()

		testApp.Router.ServeHTTP(rr, req)

		// --- Assertion ---
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

// createAccountForTest is a new helper for creating accounts.
func createAccountForTest(t *testing.T, userID int, currency string) model.Account {
	accountService := service.NewAccountService(repository.NewAccountRepository(testApp.DB))
	account, err := accountService.CreateNewAccount(userID, currency)
	assert.NoError(t, err)
	return *account
}

// TestAdminRoutes_Integration tests the admin-only endpoints and middleware.
func TestAdminRoutes_Integration(t *testing.T) {
	// --- Setup ---
	// 1. Create an Admin User and a Regular User
	adminUser := createUserWithRoleForTest(t, "admin_user", "admin@test.com", "password123", model.RoleAdmin)
	regularUser := createUserWithRoleForTest(t, "regular_user", "user@test.com", "password123", model.RoleUser)
	defer cleanupUser(t, adminUser.Email)
	defer cleanupUser(t, regularUser.Email)

	// 2. Login as both to get their tokens
	adminToken := loginUserForTest(t, adminUser.Email, "password123")
	userToken := loginUserForTest(t, regularUser.Email, "password123")

	endpoint := "/api/admin/users"

	t.Run("admin can access admin routes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", endpoint, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rr := httptest.NewRecorder()

		// Execute
		testApp.Router.ServeHTTP(rr, req)

		// Assert: Admin should get 200 OK.
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("regular user is forbidden from admin routes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", endpoint, nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		rr := httptest.NewRecorder()

		// Execute
		testApp.Router.ServeHTTP(rr, req)

		// Assert: Regular user should get 403 Forbidden.
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

// --- New Test Helper ---

// createUserWithRoleForTest is a helper to create a user with a specific role.
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

// TestAuthFlows_Integration tests the refresh token and logout logic.
func TestAuthFlows_Integration(t *testing.T) {
	// --- Setup ---
	email := "authflow@test.com"
	password := "password123"
	user := createUserForTest(t, "authflow_user", email, password)
	defer cleanupUser(t, user.Email)

	// 1. Login to get initial token pair
	loginBody := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, email, password)
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(loginBody))
	rr := httptest.NewRecorder()
	testApp.Router.ServeHTTP(rr, req)

	var loginResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &loginResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, loginResponse.AccessToken)
	assert.NotEmpty(t, loginResponse.RefreshToken)

	initialAccessToken := loginResponse.AccessToken

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
		// Use the valid access token to logout
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
