package resolvers

import "context"

type policyStatusResolver struct {
	status          string
	failingPolicies []*policyResolver
}

func (resolver *policyStatusResolver) Status(ctx context.Context) (string, error) {
	return resolver.status, nil
}

func (resolver *policyStatusResolver) FailingPolicies(ctx context.Context) ([]*policyResolver, error) {
	return resolver.failingPolicies, nil
}
