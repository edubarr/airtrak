package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/edubarr/airtrak/backend/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	JWTSecret      string
	DefaultLocale  string
	AccessTokenTTL time.Duration
}

type Service struct {
	pool           *pgxpool.Pool
	queries        *store.Queries
	tokens         *TokenManager
	defaultLocale  string
	accessTokenTTL time.Duration
}

func NewService(pool *pgxpool.Pool, queries *store.Queries, cfg Config) (*Service, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is required")
	}

	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}

	if cfg.AccessTokenTTL <= 0 {
		return nil, fmt.Errorf("access token ttl must be positive")
	}

	tokens, err := NewTokenManager(cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	defaultLocale := strings.TrimSpace(cfg.DefaultLocale)
	if defaultLocale == "" {
		defaultLocale = "pt-BR"
	}

	if !isSupportedLocale(defaultLocale) {
		return nil, fmt.Errorf("unsupported default locale: %s", defaultLocale)
	}

	return &Service{
		pool:           pool,
		queries:        queries,
		tokens:         tokens,
		accessTokenTTL: cfg.AccessTokenTTL,
		defaultLocale:  defaultLocale,
	}, nil
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return AuthResult{}, err
	}

	if len(input.Password) < 8 {
		return AuthResult{}, fmt.Errorf("%w: password must have at least 8 characters", ErrInvalidInput)
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = strings.Split(email, "@")[0]
	}

	locale, err := s.normalizeLocale(input.Locale)
	if err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	hash := string(passwordHash)
	now := time.Now().UTC()
	expiresAt := now.Add(s.accessTokenTTL)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AuthResult{}, fmt.Errorf("begin register transaction: %w", err)
	}
	defer rollback(ctx, tx)

	qtx := s.queries.WithTx(tx)

	storedUser, err := qtx.CreateUser(ctx, store.CreateUserParams{
		DisplayName: displayName,
		Locale:      locale,
	})
	if err != nil {
		return AuthResult{}, fmt.Errorf("create user: %w", err)
	}

	_, err = qtx.CreateCredential(ctx, store.CreateCredentialParams{
		UserID:         storedUser.ID,
		CredentialType: CredentialTypeEmailPassword,
		Identifier:     email,
		SecretHash:     &hash,
		VerifiedAt:     pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		if isUniqueViolation(err) {
			return AuthResult{}, ErrEmailAlreadyExists
		}

		return AuthResult{}, fmt.Errorf("create email credential: %w", err)
	}

	session, err := qtx.CreateSession(ctx, store.CreateSessionParams{
		UserID:    storedUser.ID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return AuthResult{}, fmt.Errorf("create session: %w", err)
	}

	accessToken, err := s.tokens.Generate(storedUser.ID, session.ID, now, expiresAt)
	if err != nil {
		return AuthResult{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return AuthResult{}, fmt.Errorf("commit register transaction: %w", err)
	}

	return AuthResult{ExpiresAt: expiresAt, AccessToken: accessToken, User: userFromStore(&storedUser)}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	credential, err := s.queries.GetCredentialByTypeIdentifier(ctx, store.GetCredentialByTypeIdentifierParams{
		CredentialType: CredentialTypeEmailPassword,
		Identifier:     email,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthResult{}, ErrInvalidCredentials
		}

		return AuthResult{}, fmt.Errorf("get email credential: %w", err)
	}

	if credential.SecretHash == nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(*credential.SecretHash), []byte(input.Password))
	if err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	storedUser, err := s.queries.GetUser(ctx, credential.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthResult{}, ErrInvalidCredentials
		}

		return AuthResult{}, fmt.Errorf("get login user: %w", err)
	}

	if storedUser.Status != UserStatusActive {
		return AuthResult{}, ErrUserDisabled
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.accessTokenTTL)

	session, err := s.queries.CreateSession(ctx, store.CreateSessionParams{
		UserID:    storedUser.ID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return AuthResult{}, fmt.Errorf("create session: %w", err)
	}

	accessToken, err := s.tokens.Generate(storedUser.ID, session.ID, now, expiresAt)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{ExpiresAt: expiresAt, AccessToken: accessToken, User: userFromStore(&storedUser)}, nil
}

func (s *Service) AuthenticateToken(ctx context.Context, rawToken string) (Principal, error) {
	parsed, err := s.tokens.Parse(rawToken)
	if err != nil {
		return Principal{}, ErrUnauthenticated
	}

	session, err := s.queries.GetActiveSession(ctx, parsed.SessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Principal{}, ErrUnauthenticated
		}

		return Principal{}, fmt.Errorf("get active session: %w", err)
	}

	if session.UserID != parsed.UserID {
		return Principal{}, ErrUnauthenticated
	}

	storedUser, err := s.queries.GetUser(ctx, parsed.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Principal{}, ErrUnauthenticated
		}

		return Principal{}, fmt.Errorf("get token user: %w", err)
	}

	if storedUser.Status != UserStatusActive {
		return Principal{}, ErrUnauthenticated
	}

	return Principal{User: userFromStore(&storedUser), SessionID: session.ID}, nil
}

func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.queries.RevokeSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}

func (s *Service) FindUserByCredential(ctx context.Context, credentialType, identifier string) (User, error) {
	storedUser, err := s.queries.FindUserByCredential(ctx, store.FindUserByCredentialParams{
		CredentialType: credentialType,
		Identifier:     strings.TrimSpace(identifier),
	})
	if err != nil {
		return User{}, err
	}

	return userFromStore(&storedUser), nil
}

func (s *Service) normalizeLocale(locale string) (string, error) {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = s.defaultLocale
	}

	if !isSupportedLocale(locale) {
		return "", fmt.Errorf("%w: unsupported locale", ErrInvalidInput)
	}

	return locale, nil
}

func normalizeEmail(email string) (string, error) {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" || strings.ContainsAny(trimmed, " <>") {
		return "", fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}

	address, err := mail.ParseAddress(trimmed)
	if err != nil || !strings.EqualFold(address.Address, trimmed) || !strings.Contains(address.Address, "@") {
		return "", fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}

	return strings.ToLower(address.Address), nil
}

func isSupportedLocale(locale string) bool {
	switch locale {
	case "pt-BR", "en":
		return true
	default:
		return false
	}
}

func userFromStore(user *store.User) User {
	return User{
		ID:          user.ID,
		DisplayName: user.DisplayName,
		Locale:      user.Locale,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
