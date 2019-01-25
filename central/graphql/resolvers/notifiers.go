package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/generated/api/v1"
)

func init() {
	schema := getBuilder()
	schema.AddQuery("notifiers: [Notifier!]!")
	schema.AddQuery("notifier(id: ID!): Notifier")
}

// Notifiers gets all available notifiers. In theory v1.GetNotifiersRequest has fields that we should represent here,
// but in practice nobody uses them and they're not implemented in the store.
func (resolver *Resolver) Notifiers(ctx context.Context) ([]*notifierResolver, error) {
	if err := notifierAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNotifiers(
		resolver.NotifierStore.GetNotifiers(&v1.GetNotifiersRequest{}))
}

// Notifier gets a single notifier by ID
func (resolver *Resolver) Notifier(ctx context.Context, args struct{ graphql.ID }) (*notifierResolver, error) {
	if err := notifierAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapNotifier(
		resolver.NotifierStore.GetNotifier(string(args.ID)))
}
