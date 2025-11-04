package retry

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GrpcRetryPolicy reports whether a gRPC error should be retried.
type GrpcRetryPolicy struct {
	retryableCodes map[codes.Code]bool
}

// NoCodesRetriedGrpcRetryPolicy creates an empty policy that retries no status codes
// until WithRetryableCodes is applied.
func NoCodesRetriedGrpcRetryPolicy() *GrpcRetryPolicy {
	return &GrpcRetryPolicy{retryableCodes: make(map[codes.Code]bool)}
}

// AllCodesRetriedGrpcRetryPolicy retries every gRPC code; callers can then remove
// codes via WithNonRetryableCodes to document which ones are intentionally
// excluded.
func AllCodesRetriedGrpcRetryPolicy() *GrpcRetryPolicy {
	retryable := make(map[codes.Code]bool)
	for i := codes.Code(0); i <= codes.Unauthenticated; i++ {
		retryable[i] = true
	}
	return &GrpcRetryPolicy{retryableCodes: retryable}
}

// DefaultGrpcRetryPolicy retries server or transient errors and skips obvious
// client errors (InvalidArgument, PermissionDenied, etc.).
func DefaultGrpcRetryPolicy() *GrpcRetryPolicy {
	// Start with all codes as retryable to express a "retry unless proven otherwise" stance.
	retryable := make(map[codes.Code]bool)

	for i := codes.Code(0); i <= codes.Unauthenticated; i++ {
		retryable[i] = true
	}

	nonRetryableCodes := []codes.Code{
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
	}

	for _, code := range nonRetryableCodes {
		retryable[code] = false
	}

	return &GrpcRetryPolicy{retryableCodes: retryable}
}

// WithRetryableCodes returns a copy of the policy with the provided codes marked
// as retryable.
func (p *GrpcRetryPolicy) WithRetryableCodes(statusCodes ...codes.Code) *GrpcRetryPolicy {
	// Create a new policy to avoid mutating the original
	newRetryable := make(map[codes.Code]bool, len(p.retryableCodes))
	for k, v := range p.retryableCodes {
		newRetryable[k] = v
	}

	for _, code := range statusCodes {
		newRetryable[code] = true
	}

	return &GrpcRetryPolicy{retryableCodes: newRetryable}
}

// WithNonRetryableCodes returns a copy of the policy with the provided codes
// marked as non-retryable.
func (p *GrpcRetryPolicy) WithNonRetryableCodes(statusCodes ...codes.Code) *GrpcRetryPolicy {
	// Create a new policy to avoid mutating the original
	newRetryable := make(map[codes.Code]bool, len(p.retryableCodes))
	for k, v := range p.retryableCodes {
		newRetryable[k] = v
	}

	for _, code := range statusCodes {
		newRetryable[code] = false
	}

	return &GrpcRetryPolicy{retryableCodes: newRetryable}
}

// ShouldRetry reports whether err maps to a retryable gRPC status code.
func (p *GrpcRetryPolicy) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	grpcStatus, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error, don't retry
		return false
	}

	return p.retryableCodes[grpcStatus.Code()]
}
