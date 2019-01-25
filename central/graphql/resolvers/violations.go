package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func init() {
	schema := getBuilder()
	schema.AddQuery("violations(query: String): [Alert!]!")
	schema.AddQuery("violation(id: ID!): Alert")
}

// Violations returns a list of all violations, or those that match the requested query
func (resolver *Resolver) Violations(ctx context.Context, args rawQuery) ([]*alertResolver, error) {
	if err := alertAuth(ctx); err != nil {
		return nil, err
	}
	q, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}
	if q == nil {
		q = search.EmptyQuery()
	}
	return resolver.wrapListAlerts(
		resolver.ViolationsDataStore.SearchListAlerts(q))
}

// Violation returns the violation with the requested ID
func (resolver *Resolver) Violation(ctx context.Context, args struct{ graphql.ID }) (*alertResolver, error) {
	if err := alertAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapAlert(
		resolver.ViolationsDataStore.GetAlert(string(args.ID)))
}

func (resolver *Resolver) getAlert(id string) *storage.Alert {
	alert, ok, err := resolver.ViolationsDataStore.GetAlert(id)
	if err != nil || !ok {
		return nil
	}
	return alert
}
