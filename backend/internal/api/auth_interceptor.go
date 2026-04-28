package api

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/edubarr/airtrak/backend/gen/airtrak/auth/v1/authv1connect"
	"github.com/edubarr/airtrak/backend/internal/auth"
)

func AuthInterceptor(service *auth.Service) connect.UnaryInterceptorFunc {
	publicProcedures := map[string]struct{}{
		authv1connect.AuthServiceRegisterProcedure: {},
		authv1connect.AuthServiceLoginProcedure:    {},
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := publicProcedures[req.Spec().Procedure]; ok {
				return next(ctx, req)
			}

			token := bearerToken(req.Header().Get("Authorization"))
			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, auth.ErrUnauthenticated)
			}

			principal, err := service.AuthenticateToken(ctx, token)
			if err != nil {
				return nil, mapAuthError(err)
			}

			return next(auth.ContextWithPrincipal(ctx, &principal), req)
		}
	}
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}

	return strings.TrimSpace(token)
}
