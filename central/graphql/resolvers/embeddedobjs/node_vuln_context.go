package embeddedobjs

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// nodeVulnContextKey is the key for the *storage.NodeVulnerability value in the context.
type nodeVulnContextKey struct{}

// nodeVulnContextValue holds the value of the distro in the context.
type nodeVulnContextValue struct {
	vuln        *storage.NodeVulnerability
	lastScanned *types.Timestamp
}

// NodeVulnContext returns a new context with the vuln attached.
func NodeVulnContext(ctx context.Context, vuln *storage.NodeVulnerability) context.Context {
	return context.WithValue(ctx, nodeVulnContextKey{}, &nodeVulnContextValue{
		vuln: vuln,
	})
}

// NodeVulnFromContext returns the vuln from the input context.
func NodeVulnFromContext(context context.Context) *storage.NodeVulnerability {
	if context == nil {
		return nil
	}
	value := context.Value(nodeVulnContextKey{})
	if value == nil {
		return nil
	}
	return value.(*nodeVulnContextValue).vuln
}
