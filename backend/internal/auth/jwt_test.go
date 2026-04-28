package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenManagerRoundTrip(t *testing.T) {
	manager, err := NewTokenManager("test-secret")
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	userID := uuid.New()
	sessionID := uuid.New()
	expiresAt := time.Now().Add(time.Hour).UTC()

	token, err := manager.Generate(userID, sessionID, time.Now().UTC(), expiresAt)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	parsed, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.UserID != userID {
		t.Fatalf("parsed user id = %s, want %s", parsed.UserID, userID)
	}

	if parsed.SessionID != sessionID {
		t.Fatalf("parsed session id = %s, want %s", parsed.SessionID, sessionID)
	}
}

func TestTokenManagerRejectsWrongSecret(t *testing.T) {
	manager, err := NewTokenManager("test-secret")
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	otherManager, err := NewTokenManager("other-secret")
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	token, err := manager.Generate(uuid.New(), uuid.New(), time.Now().UTC(), time.Now().Add(time.Hour).UTC())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, err = otherManager.Parse(token)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Parse() error = %v, want %v", err, ErrUnauthenticated)
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	manager, err := NewTokenManager("test-secret")
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	token, err := manager.Generate(
		uuid.New(),
		uuid.New(),
		time.Now().Add(-2*time.Hour).UTC(),
		time.Now().Add(-time.Hour).UTC(),
	)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, err = manager.Parse(token)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Parse() error = %v, want %v", err, ErrUnauthenticated)
	}
}
