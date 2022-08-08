package embeddedobjs

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// componentContextKey is the key for the *storage.EmbeddedImageScanComponent value in the context.
// All data is scoped to the embedding image.
type componentContextKey struct{}

// componentContextValue holds the value of the distro in the context.
type componentContextValue struct {
	component   *storage.EmbeddedImageScanComponent
	os          string
	lastScanned *types.Timestamp
}

// ComponentContext returns a new context with the component attached.
func ComponentContext(ctx context.Context, os string, lastScanned *types.Timestamp, component *storage.EmbeddedImageScanComponent) context.Context {
	return context.WithValue(ctx, componentContextKey{}, &componentContextValue{
		component:   component,
		os:          os,
		lastScanned: lastScanned,
	})
}

// ComponentFromContext returns the component from the input context.
func ComponentFromContext(context context.Context) *storage.EmbeddedImageScanComponent {
	if context == nil {
		return nil
	}
	value := context.Value(componentContextKey{})
	if value == nil {
		return nil
	}
	return value.(*componentContextValue).component
}

// LastScannedFromContext returns the last scanned time of the component, scoped to embedding image, from the input context.
func LastScannedFromContext(context context.Context) *types.Timestamp {
	if context == nil {
		return nil
	}
	value := context.Value(componentContextKey{})
	if value == nil {
		return nil
	}
	return value.(*componentContextValue).lastScanned
}

// OSFromContext returns the operating system of the component, scoped to embedding image, from the input context.
func OSFromContext(context context.Context) string {
	if context == nil {
		return ""
	}
	value := context.Value(componentContextKey{})
	if value == nil {
		return ""
	}
	return value.(*componentContextValue).os
}
