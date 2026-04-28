package api

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/edubarr/airtrak/backend/gen/airtrak/auth/v1"
	"github.com/edubarr/airtrak/backend/internal/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthHandler struct {
	service *auth.Service
}

func NewAuthHandler(service *auth.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(
	ctx context.Context,
	req *connect.Request[authv1.RegisterRequest],
) (*connect.Response[authv1.RegisterResponse], error) {
	result, err := h.service.Register(ctx, auth.RegisterInput{
		Email:       req.Msg.Email,
		Password:    req.Msg.Password,
		DisplayName: req.Msg.DisplayName,
		Locale:      req.Msg.Locale,
	})
	if err != nil {
		return nil, mapAuthError(err)
	}

	return connect.NewResponse(&authv1.RegisterResponse{
		User:        toProtoUser(&result.User),
		AccessToken: result.AccessToken,
		ExpiresAt:   timestamppb.New(result.ExpiresAt),
	}), nil
}

func (h *AuthHandler) Login(
	ctx context.Context,
	req *connect.Request[authv1.LoginRequest],
) (*connect.Response[authv1.LoginResponse], error) {
	result, err := h.service.Login(ctx, auth.LoginInput{
		Email:    req.Msg.Email,
		Password: req.Msg.Password,
	})
	if err != nil {
		return nil, mapAuthError(err)
	}

	return connect.NewResponse(&authv1.LoginResponse{
		User:        toProtoUser(&result.User),
		AccessToken: result.AccessToken,
		ExpiresAt:   timestamppb.New(result.ExpiresAt),
	}), nil
}

func (h *AuthHandler) GetMe(
	ctx context.Context,
	_ *connect.Request[authv1.GetMeRequest],
) (*connect.Response[authv1.GetMeResponse], error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, auth.ErrUnauthenticated)
	}

	return connect.NewResponse(&authv1.GetMeResponse{User: toProtoUser(&principal.User)}), nil
}

func (h *AuthHandler) Logout(
	ctx context.Context,
	_ *connect.Request[authv1.LogoutRequest],
) (*connect.Response[authv1.LogoutResponse], error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, auth.ErrUnauthenticated)
	}

	if err := h.service.Logout(ctx, principal.SessionID); err != nil {
		return nil, mapAuthError(err)
	}

	return connect.NewResponse(&authv1.LogoutResponse{}), nil
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, auth.ErrEmailAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrUnauthenticated):
		return connect.NewError(connect.CodeUnauthenticated, auth.ErrUnauthenticated)
	case errors.Is(err, auth.ErrUserDisabled):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func toProtoUser(user *auth.User) *authv1.User {
	return &authv1.User{
		Id:          user.ID.String(),
		DisplayName: user.DisplayName,
		Locale:      user.Locale,
		Status:      user.Status,
		CreatedAt:   timestamppb.New(user.CreatedAt),
		UpdatedAt:   timestamppb.New(user.UpdatedAt),
	}
}
