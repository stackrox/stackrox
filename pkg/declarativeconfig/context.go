package declarativeconfig

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

type originCheckerKey struct{}

const allowOnlyDeclarativeOperations = 0
const allowModifyDeclarativeOrImperative = 1

// WithModifyDeclarativeResource returns a context that is a child of the given context and allows to modify
// proto messages with the traits origin == DECLARATIVE.
func WithModifyDeclarativeResource(ctx context.Context) context.Context {
	return context.WithValue(ctx, originCheckerKey{}, allowOnlyDeclarativeOperations)
}

// WithModifyDeclarativeOrImperative returns a context that is a child of the given context and allows to modify
// proto messages with the traits origin == DECLARATIVE or DECLARATIVE_ORPHANED or IMPERATIVE
func WithModifyDeclarativeOrImperative(ctx context.Context) context.Context {
	return context.WithValue(ctx, originCheckerKey{}, allowModifyDeclarativeOrImperative)
}

// ResourceWithTraits is a common interface for proto messages containing storage.Traits.
type ResourceWithTraits interface {
	GetTraits() *storage.Traits
}

// HasModifyDeclarativeResourceKey returns a bool indicating whether the given context allows to modify
// proto messages that are created declaratively.
func HasModifyDeclarativeResourceKey(ctx context.Context) bool {
	val := ctx.Value(originCheckerKey{})
	return val == allowOnlyDeclarativeOperations || val == allowModifyDeclarativeOrImperative
}

// CanModifyResource returns whether context holder is allowed to modify resource.
func CanModifyResource(ctx context.Context, resource ResourceWithTraits) bool {
	if ctx.Value(originCheckerKey{}) == allowOnlyDeclarativeOperations {
		return IsDeclarativeOrigin(resource)
	}
	if ctx.Value(originCheckerKey{}) == allowModifyDeclarativeOrImperative {
		return IsDeclarativeOrigin(resource) || resource.GetTraits().GetOrigin() == storage.Traits_IMPERATIVE
	}
	return resource.GetTraits().GetOrigin() == storage.Traits_IMPERATIVE
}
