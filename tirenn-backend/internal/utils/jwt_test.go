package utils

import (
	"testing"
)

func TestJWT_GenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-12345"
	userID := uint(42)
	email := "hero@comic.com"
	role := "ADMIN"
	name := "Comic Hero"

	token, err := GenerateJWT(userID, email, role, name, secret, 2)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	if token == "" {
		t.Fatal("Token is empty")
	}

	claims, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate valid JWT: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("Expected Email %s, got %s", email, claims.Email)
	}
	if claims.Role != role {
		t.Errorf("Expected Role %s, got %s", role, claims.Role)
	}

	// Test invalid secret
	_, err = ValidateJWT(token, "wrong-secret-key")
	if err == nil {
		t.Error("Expected error validating with wrong secret, got nil")
	}
}

func TestPassword_HashAndCheck(t *testing.T) {
	password := "SecretPassword@2026"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("Password check failed for matching password")
	}

	if CheckPasswordHash("WrongPassword", hash) {
		t.Error("Password check should have failed for wrong password")
	}
}
