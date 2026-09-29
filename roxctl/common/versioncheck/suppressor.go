package versioncheck

import (
	"context"
)

type versionCheckerSuppressorKey struct{}

// ShouldSuppressVersionChecker returns whether the version checker should be
// suppressed or not
func ShouldSuppressVersionChecker(ctx context.Context) bool {
	shouldSuppress, _ := ctx.Value(versionCheckerSuppressorKey{}).(bool)
	return shouldSuppress
}

// ContextWithVersionCheckerSuppressor adds the given version checker
// suppression to the context.
func ContextWithVersionCheckerSuppressor(ctx context.Context, shouldSuppress bool) context.Context {
	return context.WithValue(ctx, versionCheckerSuppressorKey{}, shouldSuppress)
}
