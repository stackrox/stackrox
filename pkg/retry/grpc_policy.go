package retry

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcPolicy interface {
	Policy
	WithRetryableCodes(...codes.Code) GrpcPolicy
	WithNonRetryableCodes(...codes.Code) GrpcPolicy
}

// grpcRetryPolicy reports whether a gRPC error should be retried.
type grpcRetryPolicy struct {
	retryableCodes map[codes.Code]bool
}

// NoGrpcCodesRetriedPolicy creates an empty policy that retries no status codes
// until WithRetryableCodes is applied.
func NoGrpcCodesRetriedPolicy() GrpcPolicy {
	return &grpcRetryPolicy{retryableCodes: make(map[codes.Code]bool)}
}

// AllGrpcCodesRetriedPolicy retries every gRPC code; callers can then remove
// codes via WithNonRetryableCodes to document which ones are intentionally
// excluded.
func AllGrpcCodesRetriedPolicy() GrpcPolicy {
	retryable := make(map[codes.Code]bool)
	for i := codes.Code(0); i <= codes.Unauthenticated; i++ {
		retryable[i] = true
	}
	return &grpcRetryPolicy{retryableCodes: retryable}
}

// DefaultGrpcPolicy retries server or transient errors and skips obvious
// client errors (InvalidArgument, PermissionDenied, etc.).
func DefaultGrpcPolicy() GrpcPolicy {
	return AllGrpcCodesRetriedPolicy().WithNonRetryableCodes(
		codes.OK,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.FailedPrecondition,
		codes.OutOfRange,
		codes.Unimplemented,
		codes.Canceled,
	)
}

// WithRetryableCodes marks the provided codes as retryable and returns the policy
// for chaining. Since policies are created via constructors, this mutates the policy
// in place.
func (p *grpcRetryPolicy) WithRetryableCodes(statusCodes ...codes.Code) GrpcPolicy {
	for _, code := range statusCodes {
		p.retryableCodes[code] = true
	}
	return p
}

// WithNonRetryableCodes marks the provided codes as non-retryable and returns the
// policy for chaining. Since policies are created via constructors, this mutates
// the policy in place.
func (p *grpcRetryPolicy) WithNonRetryableCodes(statusCodes ...codes.Code) GrpcPolicy {
	for _, code := range statusCodes {
		delete(p.retryableCodes, code)
	}
	return p
}

// ShouldRetry reports whether err maps to a retryable gRPC status code.
func (p *grpcRetryPolicy) ShouldRetry(err error) bool {
	if p == nil || err == nil {
		return false
	}

	grpcStatus, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error, don't retry
		return false
	}

	return p.retryableCodes[grpcStatus.Code()]
}
