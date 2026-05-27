package auth

import (
	"testing"

	"xnauth/internal/config"
)

func TestManagerGenerateAndParse(t *testing.T) {
	manager := NewManager(config.JWTConfig{
		Secret:      "test-secret",
		ExpireHours: 1,
	})

	token, _, err := manager.Generate(1, "admin")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.AdminID != 1 || claims.Username != "admin" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestManagerRejectsTamperedToken(t *testing.T) {
	manager := NewManager(config.JWTConfig{
		Secret:      "test-secret",
		ExpireHours: 1,
	})

	token, _, err := manager.Generate(1, "admin")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if _, err := manager.Parse(token + "x"); err == nil {
		t.Fatal("tampered token should be rejected")
	}
}
