package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/utils"
)

// TODO(ROX-6194): This file provides a resolver for the deprecated
//   `Policy.whitelists` field to maintain backward compatibility for GraphQL.
//   Removed after the deprecation cycle started with the 55.0 release.

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolver("Policy", "whitelists: [Exclusion]!"),
	)
}

func (resolver *policyResolver) Whitelists(ctx context.Context) ([]*exclusionResolver, error) {
	value := resolver.data.GetExclusions()
	return resolver.root.wrapExclusions(value, nil)
}
