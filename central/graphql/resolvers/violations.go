package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("violations(query: String): [Alert!]!"),
		schema.AddQuery("violation(id: ID!): Alert"),
	)
}

// Violations returns a list of all violations, or those that match the requested query
func (resolver *Resolver) Violations(ctx context.Context, args rawQuery) ([]*alertResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Violations")
	if err := readAlerts(ctx); err != nil {
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
		resolver.ViolationsDataStore.SearchListAlerts(ctx, q))
}

// Violation returns the violation with the requested ID
func (resolver *Resolver) Violation(ctx context.Context, args struct{ graphql.ID }) (*alertResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Violation")
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapAlert(
		resolver.ViolationsDataStore.GetAlert(ctx, string(args.ID)))
}

func (resolver *Resolver) getAlert(ctx context.Context, id string) *storage.Alert {
	alert, ok, err := resolver.ViolationsDataStore.GetAlert(ctx, id)
	if err != nil || !ok {
		return nil
	}
	return alert
}
