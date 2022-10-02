package context

import "context"

type retryContextKey struct{}

// WithRetry returns a context with the retry context key
func WithRetry(ctx context.Context) context.Context {
	return context.WithValue(ctx, retryContextKey{}, true)
}

// HasRetry context checks if there is a retry context
func HasRetry(ctx context.Context) bool {
	return ctx.Value(retryContextKey{}) != nil
}
