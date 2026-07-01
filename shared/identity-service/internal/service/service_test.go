package service

import (
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	password := "SuperSecret123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("HashPassword returned empty hash")
	}

	if err := ComparePassword(hash, password); err != nil {
		t.Fatalf("ComparePassword returned error for correct password: %v", err)
	}

	if err := ComparePassword(hash, "wrong-password"); err == nil {
		t.Fatal("ComparePassword accepted a wrong password")
	}
}

func TestJWTManagerGeneratesToken(t *testing.T) {
	manager := &JWTManager{}
	manager.privateKey = nil
	manager.publicKey = nil

	_, err := manager.GenerateToken("user-1", "alice", "user", "client", []string{"read:users"})
	if err == nil {
		t.Fatal("GenerateToken should fail without key material")
	}
}
