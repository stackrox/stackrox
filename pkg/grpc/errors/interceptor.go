package errors

import (
	"context"
	"errors"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/pkg/errorhelpers"
	errox_grpc "github.com/stackrox/rox/pkg/errox/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PanicOnInvariantViolationUnaryInterceptor panics on ErrInvariantViolation.
// Note: this interceptor should ONLY be used in dev builds.
func PanicOnInvariantViolationUnaryInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if errors.Is(err, errorhelpers.ErrInvariantViolation) {
		panic(err)
	}
	return resp, err
}

// PanicOnInvariantViolationStreamInterceptor panics on ErrInvariantViolation.
// Note: this interceptor should ONLY be used in dev builds.
func PanicOnInvariantViolationStreamInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	if errors.Is(err, errorhelpers.ErrInvariantViolation) {
		panic(err)
	}
	return err
}

// ErrorToGrpcCodeInterceptor translates common errors defined in errorhelpers to GRPC codes.
func ErrorToGrpcCodeInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	return resp, ErrToGrpcStatus(err).Err()
}

// ErrorToGrpcCodeStreamInterceptor translates common errors defined in errorhelpers to GRPC codes.
func ErrorToGrpcCodeStreamInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	return ErrToGrpcStatus(err).Err()
}

// unwrapGRPCStatus unwraps the `err` chain to find an error
// implementing `GRPCStatus()`.
func unwrapGRPCStatus(err error) *status.Status {
	var se interface{ GRPCStatus() *status.Status }
	if errors.As(err, &se) {
		return se.GRPCStatus()
	}
	return nil
}

// ErrToHTTPStatus maps known internal and gRPC errors to the appropriate
// HTTP status code.
func ErrToHTTPStatus(err error) int {
	return runtime.HTTPStatusFromCode(ErrToGrpcStatus(err).Code())
}

// ErrToGrpcStatus wraps an error into a gRPC status with code.
func ErrToGrpcStatus(err error) *status.Status {
	if s, ok := status.FromError(err); ok {
		// `err` is either nil or status.Status.
		return s
	}
	var code codes.Code
	// `status.FromError()` doesn't unwrap the `err` chain, so unwrap it here.
	if s := unwrapGRPCStatus(err); s != nil {
		code = s.Code()
	} else {
		code = errox_grpc.RoxErrorToGRPCCode(err)
	}
	return status.New(code, err.Error())
}
