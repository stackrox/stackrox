package embeddedobjs

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
)

// nodeComponentContextKey is the key for the *storage.EmbeddedNodeScanComponent value in the context.
// All data is scoped to the embedding node.
type nodeComponentContextKey struct{}

// nodeComponentContextValue holds the value of the distro in the context.
type nodeComponentContextValue struct {
	component   *storage.EmbeddedNodeScanComponent
	lastScanned *time.Time
}

// NodeComponentContext returns a new context with the component attached.
func NodeComponentContext(ctx context.Context, lastScanned *time.Time, component *storage.EmbeddedNodeScanComponent) context.Context {
	return context.WithValue(ctx, nodeComponentContextKey{}, &nodeComponentContextValue{
		component:   component,
		lastScanned: lastScanned,
	})
}

// NodeComponentFromContext returns the component from the input context.
func NodeComponentFromContext(context context.Context) *storage.EmbeddedNodeScanComponent {
	if context == nil {
		return nil
	}
	value := context.Value(nodeComponentContextKey{})
	if value == nil {
		return nil
	}
	return value.(*nodeComponentContextValue).component
}

// NodeComponentLastScannedFromContext returns the last scanned time of the component, scoped to embedding node, from the input context.
func NodeComponentLastScannedFromContext(context context.Context) *time.Time {
	if context == nil {
		return nil
	}
	value := context.Value(nodeComponentContextKey{})
	if value == nil {
		return nil
	}
	return value.(*nodeComponentContextValue).lastScanned
}
