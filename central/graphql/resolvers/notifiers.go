package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("notifiers: [Notifier!]!"),
		schema.AddQuery("notifier(id: ID!): Notifier"),
	)
}

// Notifiers gets all available notifiers. In theory v1.GetNotifiersRequest has fields that we should represent here,
// but in practice nobody uses them and they're not implemented in the store.
func (resolver *Resolver) Notifiers(ctx context.Context) ([]*notifierResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Notifiers")
	if err := readIntegrations(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNotifiers(
		resolver.NotifierStore.GetScrubbedNotifiers(ctx))
}

// Notifier gets a single notifier by ID
func (resolver *Resolver) Notifier(ctx context.Context, args struct{ graphql.ID }) (*notifierResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Notifier")
	if err := readIntegrations(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNotifier(
		resolver.NotifierStore.GetScrubbedNotifier(ctx, string(args.ID)))
}
