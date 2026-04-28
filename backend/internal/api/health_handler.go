package api

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/edubarr/airtrak/backend/gen/airtrak/health/v1"
)

const readinessTimeout = 2 * time.Second

type readinessChecker interface {
	Ping(context.Context) error
}

type HealthHandler struct {
	checker readinessChecker
}

func NewHealthHandler(checker readinessChecker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

func (h *HealthHandler) Healthz(
	context.Context,
	*connect.Request[healthv1.HealthzRequest],
) (*connect.Response[healthv1.HealthzResponse], error) {
	return connect.NewResponse(&healthv1.HealthzResponse{Status: "ok"}), nil
}

func (h *HealthHandler) Readyz(
	ctx context.Context,
	_ *connect.Request[healthv1.ReadyzRequest],
) (*connect.Response[healthv1.ReadyzResponse], error) {
	readyCtx, cancel := context.WithTimeout(ctx, readinessTimeout)
	defer cancel()

	if err := h.checker.Ping(readyCtx); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("database unavailable: %w", err))
	}

	return connect.NewResponse(&healthv1.ReadyzResponse{Status: "ready"}), nil
}
