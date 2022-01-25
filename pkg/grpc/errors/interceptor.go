package errors

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"google.golang.org/grpc"
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
	return resp, ErrToGRPCStatus(err).Err()
}

// ErrorToGrpcCodeStreamInterceptor translates common errors defined in errorhelpers to GRPC codes.
func ErrorToGrpcCodeStreamInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return ErrToGRPCStatus(handler(srv, ss)).Err()
}
