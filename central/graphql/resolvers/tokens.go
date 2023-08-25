package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("tokens(revoked:Boolean): [TokenMetadata!]!"),
		schema.AddQuery("token(id:ID!): TokenMetadata"),
	)
}

// Tokens gets a list of all tokens (or just the ones that are or are not resolved)
func (resolver *Resolver) Tokens(ctx context.Context, args struct{ Revoked *bool }) ([]*tokenMetadataResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Tokens")
	if err := readIntegrations(ctx); err != nil {
		return nil, err
	}
	req := &v1.GetAPITokensRequest{}
	if args.Revoked != nil {
		req.RevokedOneof = &v1.GetAPITokensRequest_Revoked{Revoked: *args.Revoked}
	}
	return resolver.wrapTokenMetadatas(
		resolver.APITokenBackend.GetTokens(ctx, req))
}

// Token gets a single API token by ID
func (resolver *Resolver) Token(ctx context.Context, args struct{ graphql.ID }) (*tokenMetadataResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Token")
	if err := readIntegrations(ctx); err != nil {
		return nil, err
	}
	token, err := resolver.APITokenBackend.GetTokenOrNil(ctx, string(args.ID))
	return resolver.wrapTokenMetadata(token, token != nil, err)
}
