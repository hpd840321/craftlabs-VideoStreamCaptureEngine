package auth

import (
	"testing"
	"time"
)

func TestGenerateToken_ValidToken(t *testing.T) {
	secret := []byte("test-secret-key")
	username := "admin"
	expiry := 1 * time.Hour

	token, err := GenerateToken(username, secret, expiry)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}
}

func TestValidateToken_ValidToken(t *testing.T) {
	secret := []byte("test-secret-key")
	username := "admin"

	token, _ := GenerateToken(username, secret, 1*time.Hour)
	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.Username != username {
		t.Errorf("claims.Username = %s, want %s", claims.Username, username)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	secret := []byte("test-secret-key")

	token, _ := GenerateToken("admin", secret, -1*time.Hour)
	_, err := ValidateToken(token, secret)
	if err == nil {
		t.Fatal("ValidateToken() should fail for expired token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, _ := GenerateToken("admin", []byte("correct-secret"), 1*time.Hour)
	_, err := ValidateToken(token, []byte("wrong-secret"))
	if err == nil {
		t.Fatal("ValidateToken() should fail with wrong secret")
	}
}

func TestValidateToken_InvalidTokenString(t *testing.T) {
	secret := []byte("test-secret-key")
	_, err := ValidateToken("invalid-token-string", secret)
	if err == nil {
		t.Fatal("ValidateToken() should fail for invalid token string")
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	secret := []byte("test-secret-key")
	tokenStr := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIn0.invalid"
	_, err := ValidateToken(tokenStr, secret)
	if err == nil {
		t.Fatal("ValidateToken() should fail for RS256 signed token")
	}
}

func TestHashPassword_ReturnsHash(t *testing.T) {
	hash, err := HashPassword("my-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}
	if len(hash) < 50 {
		t.Errorf("hash too short: len=%d", len(hash))
	}
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	password := "my-password"
	hash, _ := HashPassword(password)

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword() should return true for correct password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("correct-password")

	if CheckPassword("wrong-password", hash) {
		t.Error("CheckPassword() should return false for wrong password")
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	if CheckPassword("password", "not-a-valid-hash") {
		t.Error("CheckPassword() should return false for invalid hash")
	}
}
