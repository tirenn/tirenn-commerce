package security

import (
	"testing"
)

func TestJWTGenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-12345"
	token, err := GenerateJWT(1, "user@example.com", "customer", "Test User", secret, 2)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	claims, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if claims.UserID != 1 || claims.Email != "user@example.com" || claims.Role != "customer" {
		t.Errorf("Unexpected claims: %+v", claims)
	}
}

func TestJWTInvalidSecret(t *testing.T) {
	token, _ := GenerateJWT(1, "user@example.com", "customer", "Test User", "secret1", 2)
	_, err := ValidateJWT(token, "secret2")
	if err == nil {
		t.Errorf("Expected error validating with wrong secret, got nil")
	}
}

func TestJWTExpired(t *testing.T) {
	token, _ := GenerateJWT(1, "user@example.com", "customer", "Test User", "secret", -1)
	_, err := ValidateJWT(token, "secret")
	if err == nil {
		t.Errorf("Expected error for expired token, got nil")
	}
}

func TestPasswordHashing(t *testing.T) {
	raw := "SecurePassword123!"
	hashed, err := HashPassword(raw)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !CheckPasswordHash(raw, hashed) {
		t.Errorf("CheckPasswordHash failed for valid password")
	}

	if CheckPasswordHash("WrongPassword", hashed) {
		t.Errorf("CheckPasswordHash should return false for wrong password")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Sony WH-1000XM5 Wireless Headphones", "sony-wh-1000xm5-wireless-headphones"},
		{"  Hello   World!  ", "hello-world"},
		{"MacBook Pro 16\" (M3 Max)", "macbook-pro-16-m3-max"},
	}

	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.expected {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
