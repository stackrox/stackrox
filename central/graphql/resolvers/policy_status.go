package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/search"
)

type policyStatusResolver struct {
	ctx              context.Context
	root             *Resolver
	status           string
	failingPolicyIds []string
}

func (resolver *policyStatusResolver) Status(ctx context.Context) (string, error) {
	return resolver.status, nil
}

func (resolver *policyStatusResolver) FailingPolicies(ctx context.Context) ([]*policyResolver, error) {
	if len(resolver.failingPolicyIds) == 0 {
		return nil, nil
	}
	return resolver.root.wrapPolicies(
		resolver.root.PolicyDataStore.SearchRawPolicies(
			resolver.ctx,
			search.NewQueryBuilder().AddDocIDs(resolver.failingPolicyIds...).ProtoQuery(),
		),
	)
}
