package service

import (
	"testing"
)

// TestHashAndCheckPassword ensures that password hashing and verification work correctly.
func TestHashAndCheckPassword(t *testing.T) {
	// Define a sample password tto be used in the test.
	password := "mySecrettPassword123"

	// 1. Test Hashing
	hashedPassword, err := HashPassword(password)
	if err != nil {
		// If hashing fails, the test should fail immediaetly with a detailed error.
		t.Fatalf("HashPassword() returned an unexpected error: %v", err)
	}

	if hashedPassword == password {
		// The hashed password should never be the same as the original password.
		t.Errorf("Hashed password should not be the same as the original password.")
	}

	// 2. Test Successful Verification
	// The CheckPasswordHash function should return true for the correct password.
	match := CheckPasswordHash(password, hashedPassword)
	if !match {
		t.Errorf("CheckPasswordHash() should have returned true for a matching password, but got false.")
	}

	// 3. Test Failed Verification
	// The CheckPasswordHash function should return false for an incorrect password.
	wrongPassword := "notMyPassword"
	match = CheckPasswordHash(wrongPassword, hashedPassword)
	if match {
		t.Errorf("CheckPasswordHash() should have returned false for a non-matching password, but got true.")
	}
}
