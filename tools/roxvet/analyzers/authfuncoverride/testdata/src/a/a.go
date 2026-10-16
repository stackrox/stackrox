package a

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
)

type myAuth struct{}

func (m *myAuth) Authorized(_ context.Context, _ string) error { return nil }

var goodAuth authz.Authorizer = &myAuth{}

// ---- Correct implementation ----

type correctSvc struct{}

func (s *correctSvc) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, goodAuth.Authorized(ctx, fullMethodName)
}

// ---- Wrong signature: too few parameters ----

type wrongParamsSvc struct{}

func (s *wrongParamsSvc) AuthFuncOverride(ctx context.Context) (context.Context, error) { // want `AuthFuncOverride must have signature`
	return ctx, nil
}

// ---- Wrong signature: wrong first parameter type ----

type wrongFirstParamSvc struct{}

func (s *wrongFirstParamSvc) AuthFuncOverride(ctx string, fullMethodName string) (context.Context, error) { // want `AuthFuncOverride must have signature`
	return context.Background(), nil
}

// ---- Wrong signature: wrong return types ----

type wrongResultSvc struct{}

func (s *wrongResultSvc) AuthFuncOverride(ctx context.Context, fullMethodName string) error { // want `AuthFuncOverride must have signature`
	return nil
}

// ---- Wrong signature: value receiver ----

type valueReceiverSvc struct{}

func (s valueReceiverSvc) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `AuthFuncOverride must have signature`
	return ctx, goodAuth.Authorized(ctx, fullMethodName)
}

// ---- Wrong body: more than one statement ----

type multiStmtSvc struct{}

func (s *multiStmtSvc) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `AuthFuncOverride body must be exactly`
	_ = fullMethodName
	return ctx, goodAuth.Authorized(ctx, fullMethodName)
}

// ---- Wrong body: modified context returned ----

type modifiedCtxSvc struct{}

func (s *modifiedCtxSvc) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `AuthFuncOverride body must be exactly`
	return context.WithValue(ctx, struct{}{}, nil), goodAuth.Authorized(ctx, fullMethodName)
}

// ---- Wrong body: hardcoded method name in Authorized call ----

type hardcodedMethodSvc struct{}

func (s *hardcodedMethodSvc) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `AuthFuncOverride body must be exactly`
	return ctx, goodAuth.Authorized(ctx, "/hardcoded.Method")
}

// ---- Wrong body: wraps error instead of returning directly ----

type wrappedErrSvc struct{}

func (s *wrappedErrSvc) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `AuthFuncOverride body must be exactly`
	err := goodAuth.Authorized(ctx, fullMethodName)
	return ctx, err
}

// ---- Wrong body: method name is "Authorize" not "Authorized" ----

type wrongMethodSvc struct{}

type fakeAuth struct{}

func (f *fakeAuth) Authorize(_ context.Context, _ string) error { return nil }

var altAuth = &fakeAuth{}

func (s *wrongMethodSvc) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) { // want `AuthFuncOverride body must be exactly`
	return ctx, altAuth.Authorize(ctx, fullMethodName)
}

// ---- Non-method function named AuthFuncOverride (must be ignored) ----

func AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, goodAuth.Authorized(ctx, fullMethodName)
}
