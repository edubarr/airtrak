package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	CredentialTypeEmailPassword = "email_password"
	CredentialTypeWhatsappJID   = "whatsapp_jid"

	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

type User struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DisplayName string
	Locale      string
	Status      string
	ID          uuid.UUID
}

type RegisterInput struct {
	Email       string
	Password    string
	DisplayName string
	Locale      string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	ExpiresAt   time.Time
	AccessToken string
	User        User
}

type Principal struct {
	User      User
	SessionID uuid.UUID
}

type principalContextKey struct{}

func ContextWithPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(*Principal)
	if !ok || principal == nil {
		return nil, false
	}

	return principal, true
}
