package embeddedobjs

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// vulnContextKey is the key for the *storage.EmbeddedVulnerability value in the context.
type vulnContextKey struct{}

// vulnContextValue holds the value of the distro in the context.
type vulnContextValue struct {
	vuln *storage.EmbeddedVulnerability
}

// VulnContext returns a new context with the vuln attached.
func VulnContext(ctx context.Context, vuln *storage.EmbeddedVulnerability) context.Context {
	return context.WithValue(ctx, vulnContextKey{}, &vulnContextValue{
		vuln: vuln,
	})
}

// VulnFromContext returns the vuln from the input context.
func VulnFromContext(context context.Context) *storage.EmbeddedVulnerability {
	if context == nil {
		return nil
	}
	value := context.Value(vulnContextKey{})
	if value == nil {
		return nil
	}
	return value.(*vulnContextValue).vuln
}
