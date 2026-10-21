package svc

import (
	"context"

	"github.com/stackrox/rox/central/externalbackups/service/internal"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// ---- Correct: trusted cross-package variable ----

type svc1 struct{}

func (s *svc1) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, internal.Authorizer.Authorized(ctx, fullMethodName)
}

// ---- Fail: struct field (SelectorExpr not in the trusted allow-list) ----

type svc2 struct {
	auth authz.Authorizer
}

func (s *svc2) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `does not satisfy the required constraints`
	return ctx, s.auth.Authorized(ctx, fullMethodName)
}
