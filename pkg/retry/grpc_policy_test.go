package retry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDefaultGrpcRetryPolicy(t *testing.T) {
	policy := DefaultGrpcRetryPolicy()

	// Test retryable codes (server errors and transient errors)
	retryableCodes := []codes.Code{
		codes.Unavailable,
		codes.Internal,
		codes.DeadlineExceeded,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.Unknown,
		codes.DataLoss,
	}

	for _, code := range retryableCodes {
		t.Run(code.String()+"_should_retry", func(t *testing.T) {
			err := status.Error(code, "test error")
			assert.True(t, policy.ShouldRetry(err), "Expected %s to be retryable", code)
		})
	}

	// Test non-retryable codes (client errors)
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
		t.Run(code.String()+"_should_not_retry", func(t *testing.T) {
			err := status.Error(code, "test error")
			assert.False(t, policy.ShouldRetry(err), "Expected %s to not be retryable", code)
		})
	}

	// Test edge cases
	t.Run("nil_error_should_not_retry", func(t *testing.T) {
		assert.False(t, policy.ShouldRetry(nil))
	})

	t.Run("non_grpc_error_should_not_retry", func(t *testing.T) {
		err := errors.New("not a grpc error")
		assert.False(t, policy.ShouldRetry(err))
	})
}

func TestNoCodesRetriedGrpcRetryPolicy(t *testing.T) {
	policy := NoCodesRetriedGrpcRetryPolicy().WithRetryableCodes(codes.Aborted, codes.Unavailable)

	err := status.Error(codes.Aborted, "aborted")
	assert.True(t, policy.ShouldRetry(err))

	err = status.Error(codes.Unavailable, "unavailable")
	assert.True(t, policy.ShouldRetry(err))

	err = status.Error(codes.Internal, "internal")
	assert.False(t, policy.ShouldRetry(err))
}
func TestGrpcRetryPolicy_WithRetryableCodes(t *testing.T) {
	t.Run("add_single_code", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy().WithRetryableCodes(codes.NotFound)

		// NotFound should now be retryable
		err := status.Error(codes.NotFound, "not found")
		assert.True(t, policy.ShouldRetry(err))

		// Other non-retryable codes should still not be retryable
		err = status.Error(codes.InvalidArgument, "invalid")
		assert.False(t, policy.ShouldRetry(err))

		// Retryable codes should still be retryable
		err = status.Error(codes.Unavailable, "unavailable")
		assert.True(t, policy.ShouldRetry(err))
	})

	t.Run("add_multiple_codes", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy().WithRetryableCodes(
			codes.NotFound,
			codes.Canceled,
		)

		err := status.Error(codes.NotFound, "not found")
		assert.True(t, policy.ShouldRetry(err))

		err = status.Error(codes.Canceled, "canceled")
		assert.True(t, policy.ShouldRetry(err))
	})

	t.Run("does_not_mutate_original_policy", func(t *testing.T) {
		original := DefaultGrpcRetryPolicy()
		modified := original.WithRetryableCodes(codes.NotFound)

		// Original should not retry NotFound
		err := status.Error(codes.NotFound, "not found")
		assert.False(t, original.ShouldRetry(err))

		// Modified should retry NotFound
		assert.True(t, modified.ShouldRetry(err))
	})
}

func TestGrpcRetryPolicy_WithNonRetryableCodes(t *testing.T) {
	t.Run("remove_single_code", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy().WithNonRetryableCodes(codes.Internal)

		// Internal should now not be retryable
		err := status.Error(codes.Internal, "internal error")
		assert.False(t, policy.ShouldRetry(err))

		// Other retryable codes should still be retryable
		err = status.Error(codes.Unavailable, "unavailable")
		assert.True(t, policy.ShouldRetry(err))

		// Non-retryable codes should still not be retryable
		err = status.Error(codes.InvalidArgument, "invalid")
		assert.False(t, policy.ShouldRetry(err))
	})

	t.Run("remove_multiple_codes", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy().WithNonRetryableCodes(
			codes.Internal,
			codes.Unavailable,
		)

		err := status.Error(codes.Internal, "internal")
		assert.False(t, policy.ShouldRetry(err))

		err = status.Error(codes.Unavailable, "unavailable")
		assert.False(t, policy.ShouldRetry(err))

		// DeadlineExceeded should still be retryable
		err = status.Error(codes.DeadlineExceeded, "deadline")
		assert.True(t, policy.ShouldRetry(err))
	})

	t.Run("does_not_mutate_original_policy", func(t *testing.T) {
		original := DefaultGrpcRetryPolicy()
		modified := original.WithNonRetryableCodes(codes.Internal)

		// Original should retry Internal
		err := status.Error(codes.Internal, "internal")
		assert.True(t, original.ShouldRetry(err))

		// Modified should not retry Internal
		assert.False(t, modified.ShouldRetry(err))
	})
}

func TestGrpcRetryPolicy_Chaining(t *testing.T) {
	t.Run("add_and_remove_codes", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy().
			WithRetryableCodes(codes.NotFound).
			WithNonRetryableCodes(codes.Internal)

		// NotFound should be retryable (added)
		err := status.Error(codes.NotFound, "not found")
		assert.True(t, policy.ShouldRetry(err))

		// Internal should not be retryable (removed)
		err = status.Error(codes.Internal, "internal")
		assert.False(t, policy.ShouldRetry(err))

		// Unavailable should still be retryable (default)
		err = status.Error(codes.Unavailable, "unavailable")
		assert.True(t, policy.ShouldRetry(err))

		// InvalidArgument should still not be retryable (default)
		err = status.Error(codes.InvalidArgument, "invalid")
		assert.False(t, policy.ShouldRetry(err))
	})

	t.Run("order_matters_last_wins", func(t *testing.T) {
		// Add NotFound, then remove it
		policy1 := DefaultGrpcRetryPolicy().
			WithRetryableCodes(codes.NotFound).
			WithNonRetryableCodes(codes.NotFound)

		err := status.Error(codes.NotFound, "not found")
		assert.False(t, policy1.ShouldRetry(err), "Last operation (remove) takes precedence")

		// Remove Internal, then add it back
		policy2 := DefaultGrpcRetryPolicy().
			WithNonRetryableCodes(codes.Internal).
			WithRetryableCodes(codes.Internal)

		err = status.Error(codes.Internal, "internal")
		assert.True(t, policy2.ShouldRetry(err), "Last operation (add) takes precedence")
	})
}

func TestGrpcRetryPolicy_Integration(t *testing.T) {
	t.Run("default_policy_retries_unavailable", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy()
		callCount := 0

		err := WithRetry(func() error {
			callCount++
			if callCount < 3 {
				err := status.Error(codes.Unavailable, "unavailable")
				if policy.ShouldRetry(err) {
					return MakeRetryable(err)
				}
				return err
			}
			return nil
		}, Tries(5), OnlyRetryableErrors())

		assert.NoError(t, err)
		assert.Equal(t, 3, callCount, "Should have retried until success")
	})

	t.Run("default_policy_does_not_retry_invalid_argument", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy()
		callCount := 0

		err := WithRetry(func() error {
			callCount++
			err := status.Error(codes.InvalidArgument, "invalid")
			if policy.ShouldRetry(err) {
				return MakeRetryable(err)
			}
			return err
		}, Tries(5), OnlyRetryableErrors())

		assert.Error(t, err)
		assert.Equal(t, 1, callCount, "Should not have retried")
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("custom_policy_retries_not_found", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy().WithRetryableCodes(codes.NotFound)
		callCount := 0

		err := WithRetry(func() error {
			callCount++
			if callCount < 3 {
				err := status.Error(codes.NotFound, "not found")
				if policy.ShouldRetry(err) {
					return MakeRetryable(err)
				}
				return err
			}
			return nil
		}, Tries(5), OnlyRetryableErrors())

		assert.NoError(t, err)
		assert.Equal(t, 3, callCount, "Should have retried until success")
	})

	t.Run("custom_policy_does_not_retry_internal", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy().WithNonRetryableCodes(codes.Internal)
		callCount := 0

		err := WithRetry(func() error {
			callCount++
			err := status.Error(codes.Internal, "internal")
			if policy.ShouldRetry(err) {
				return MakeRetryable(err)
			}
			return err
		}, Tries(5), OnlyRetryableErrors())

		assert.Error(t, err)
		assert.Equal(t, 1, callCount, "Should not have retried")
	})

	t.Run("policy_respects_nil_errors", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy()
		callCount := 0

		err := WithRetry(func() error {
			callCount++
			if policy.ShouldRetry(nil) {
				return MakeRetryable(errors.New("should not happen"))
			}
			return nil
		}, Tries(5), OnlyRetryableErrors())

		assert.NoError(t, err)
		assert.Equal(t, 1, callCount, "Should succeed on first try")
	})

	t.Run("policy_with_non_grpc_errors", func(t *testing.T) {
		policy := DefaultGrpcRetryPolicy()
		callCount := 0

		err := WithRetry(func() error {
			callCount++
			err := errors.New("regular error")
			if policy.ShouldRetry(err) {
				return MakeRetryable(err)
			}
			return err
		}, Tries(5), OnlyRetryableErrors())

		assert.Error(t, err)
		assert.Equal(t, 1, callCount, "Should not retry non-gRPC errors")
	})
}
