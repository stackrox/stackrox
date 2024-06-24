package errors

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	errox_grpc "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()
)

// PanicOnInvariantViolationUnaryInterceptor panics on ErrInvariantViolation.
// Note: this interceptor should ONLY be used in dev builds.
func PanicOnInvariantViolationUnaryInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if errors.Is(err, errox.InvariantViolation) {
		panic(err)
	}
	return resp, err
}

// PanicOnInvariantViolationStreamInterceptor panics on ErrInvariantViolation.
// Note: this interceptor should ONLY be used in dev builds.
func PanicOnInvariantViolationStreamInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	if errors.Is(err, errox.InvariantViolation) {
		panic(err)
	}
	return err
}

// LogInternalErrorInterceptor logs internal errors.
// Note: this interceptor should ONLY be used in dev builds because Internal is the default code,
// so all errors that have not been assigned a class will appear in the log.
func LogInternalErrorInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	logErrorIfInternal(err)
	return resp, err
}

// LogInternalErrorStreamInterceptor logs internal errors.
// Note: this interceptor should ONLY be used in dev builds because Internal is the default code,
// so all errors that have not been assigned a class will appear in the log.
func LogInternalErrorStreamInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	logErrorIfInternal(err)
	return err
}

func logErrorIfInternal(err error) {
	if grpcStatus := ErrToGrpcStatus(err); grpcStatus.Code() == codes.Internal {
		log.Errorf("Internal error occurred: %v", err)
	}
}

// concealError returns a new error, based on the type or code of the given one,
// with the message replaced with public messages found in the error chain, or,
// if none found, with the generic message for the given error type or code.
func concealError(err error) error {
	userMessage := errox.GetUserMessage(err)

	if s, ok := status.FromError(err); ok {
		if userMessage == "" {
			userMessage = s.Code().String()
		}
		return status.Error(s.Code(), userMessage)
	}
	sentinel := errox.GetBaseSentinelError(err)
	if userMessage != "" {
		return sentinel.New(userMessage)
	}
	return sentinel
}

func ConcealErrorInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	return resp, concealError(err)
}

func ConcealErrorStreamInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return concealError(handler(srv, ss))
}

// ErrorToGrpcCodeInterceptor translates common errors defined in errorhelpers to GRPC codes.
func ErrorToGrpcCodeInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	grpcStatus := ErrToGrpcStatus(err)
	return resp, grpcStatus.Err()
}

// ErrorToGrpcCodeStreamInterceptor translates common errors defined in errorhelpers to GRPC codes.
func ErrorToGrpcCodeStreamInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	grpcStatus := ErrToGrpcStatus(err)
	return grpcStatus.Err()
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
