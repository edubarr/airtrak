package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/edubarr/airtrak/backend/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestServiceRegisterLoginAuthenticateLogout(t *testing.T) {
	databaseURL := os.Getenv("AIRTRAK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set AIRTRAK_TEST_DATABASE_URL to run PostgreSQL auth integration tests")
	}

	ctx := context.Background()

	applyMigrations(t, databaseURL)

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	defer pool.Close()

	service, err := NewService(pool, store.New(pool), Config{
		JWTSecret:      "integration-secret",
		AccessTokenTTL: time.Hour,
		DefaultLocale:  "pt-BR",
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	email := fmt.Sprintf("user-%d@example.com", time.Now().UnixNano())

	registered, err := service.Register(ctx, RegisterInput{
		Email:       email,
		Password:    "password123",
		DisplayName: "Test User",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if registered.User.ID.String() == "" {
		t.Fatal("registered user id is empty")
	}

	if registered.AccessToken == "" {
		t.Fatal("registered access token is empty")
	}

	_, err = service.Register(ctx, RegisterInput{Email: email, Password: "password123"})
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("duplicate Register() error = %v, want %v", err, ErrEmailAlreadyExists)
	}

	principal, err := service.AuthenticateToken(ctx, registered.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken(register token) error = %v", err)
	}

	if principal.User.ID != registered.User.ID {
		t.Fatalf("principal user id = %s, want %s", principal.User.ID, registered.User.ID)
	}

	loggedIn, err := service.Login(ctx, LoginInput{Email: email, Password: "password123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	loginPrincipal, err := service.AuthenticateToken(ctx, loggedIn.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken(login token) error = %v", err)
	}

	err = service.Logout(ctx, loginPrincipal.SessionID)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	_, err = service.AuthenticateToken(ctx, loggedIn.AccessToken)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("AuthenticateToken(logged out token) error = %v, want %v", err, ErrUnauthenticated)
	}
}

func applyMigrations(t *testing.T, databaseURL string) {
	t.Helper()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	defer func() { _ = db.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose.SetDialect() error = %v", err)
	}

	if err := goose.Up(db, "../../db/migrations"); err != nil {
		t.Fatalf("goose.Up() error = %v", err)
	}
}
