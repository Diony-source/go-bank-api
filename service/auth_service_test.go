// file: service/auth_service_test.go

package service

import (
	"testing"
)

// TestAuthService_HashAndCheckPassword ensures that password hashing and verification methods work correctly.
func TestAuthService_HashAndCheckPassword(t *testing.T) {
	// Since HashPassword and CheckPasswordHash don't use any repository dependencies,
	// we can instantiate AuthService with nil repositories for this specific test.
	authService := NewAuthService(nil, nil)
	password := "mySecretPassword123"

	// 1. Test Hashing
	hashedPassword, err := authService.HashPassword(password)
	if err != nil {
		t.Fatalf("authService.HashPassword() returned an unexpected error: %v", err)
	}

	if hashedPassword == password {
		t.Errorf("Hashed password should not be the same as the original password.")
	}

	// 2. Test Successful Verification
	match := authService.CheckPasswordHash(password, hashedPassword)
	if !match {
		t.Errorf("authService.CheckPasswordHash() should have returned true for a matching password, but got false.")
	}

	// 3. Test Failed Verification
	wrongPassword := "notMyPassword"
	match = authService.CheckPasswordHash(wrongPassword, hashedPassword)
	if match {
		t.Errorf("authService.CheckPasswordHash() should have returned false for a non-matching password, but got true.")
	}
}
