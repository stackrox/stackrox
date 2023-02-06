package declarativeconfig

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

type originCheckerKey struct{}

// AllowOnlyDeclarativeOperations signals that the context holder is allowed to modify declarative resources.
const AllowOnlyDeclarativeOperations = true

// WithAllowOnlyDeclarativeOperations returns a context that is a child of the given context and allows to modify
// declarative resources.
func WithAllowOnlyDeclarativeOperations(ctx context.Context) context.Context {
	return context.WithValue(ctx, originCheckerKey{}, AllowOnlyDeclarativeOperations)
}

// IsOriginModifiable returns whether context allows to modify declarative resources.
func IsOriginModifiable(ctx context.Context, origin storage.Traits_Origin) bool {
	if ctx.Value(originCheckerKey{}) == AllowOnlyDeclarativeOperations {
		return origin == storage.Traits_DECLARATIVE
	}
	return origin == storage.Traits_IMPERATIVE
}
