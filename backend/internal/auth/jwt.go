package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenManager struct {
	secret []byte
}

type tokenClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type ParsedToken struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

func NewTokenManager(secret string) (*TokenManager, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("jwt secret is required")
	}

	return &TokenManager{secret: []byte(secret)}, nil
}

func (m *TokenManager) Generate(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error) {
	claims := tokenClaims{
		SessionID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return signed, nil
}

func (m *TokenManager) Parse(rawToken string) (ParsedToken, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return ParsedToken{}, ErrUnauthenticated
	}

	claims := &tokenClaims{}

	token, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected jwt signing method: %s", token.Method.Alg())
			}

			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || token == nil || !token.Valid {
		return ParsedToken{}, ErrUnauthenticated
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return ParsedToken{}, ErrUnauthenticated
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return ParsedToken{}, ErrUnauthenticated
	}

	return ParsedToken{UserID: userID, SessionID: sessionID}, nil
}
