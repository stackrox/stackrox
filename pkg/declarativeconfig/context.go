package declarativeconfig

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

type originCheckerKey struct{}

const allowOnlyDeclarativeOperations = true

// WithModifyDeclarativeResource returns a context that is a child of the given context and allows to modify
// proto messages with the traits origin == DECLARATIVE.
func WithModifyDeclarativeResource(ctx context.Context) context.Context {
	return context.WithValue(ctx, originCheckerKey{}, allowOnlyDeclarativeOperations)
}

// ResourceWithTraits is a common interface for proto messages containing storage.Traits.
type ResourceWithTraits interface {
	GetTraits() *storage.Traits
}

// HasModifyDeclarativeResourceKey returns a bool indicating whether the given context allows to modify
// proto messages that are created declaratively.
func HasModifyDeclarativeResourceKey(ctx context.Context) bool {
	return ctx.Value(originCheckerKey{}) == allowOnlyDeclarativeOperations
}

// CanModifyResource returns whether context holder is allowed to modify resource.
func CanModifyResource(ctx context.Context, resource ResourceWithTraits) bool {
	if ctx.Value(originCheckerKey{}) == allowOnlyDeclarativeOperations {
		return IsDeclarativeOrigin(resource)
	}
	return resource.GetTraits().GetOrigin() == storage.Traits_IMPERATIVE
}
