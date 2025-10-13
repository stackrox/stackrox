package resolvers

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("clusterHealthCounter(query: String): ClusterHealthCounter!"),
		schema.AddType("ClusterHealthCounter", []string{
			"total: Int!",
			"uninitialized: Int!",
			"healthy: Int!",
			"degraded: Int!",
			"unhealthy: Int!",
		}),
	)
}

// ClusterHealthCounterResolver counts the clusters by their health status.
type ClusterHealthCounterResolver struct {
	total         int32
	uninitialized int32
	healthy       int32
	degraded      int32
	unhealthy     int32
}

// ClusterHealthCounter returns counts of clusters in various health buckets.
func (resolver *Resolver) ClusterHealthCounter(ctx context.Context, args RawQuery) (*ClusterHealthCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ClusterHealthCounter")

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	return newClusterHealthCounterResolver(ctx, resolver, q)
}

func newClusterHealthCounterResolver(ctx context.Context, root *Resolver, q *v1.Query) (*ClusterHealthCounterResolver, error) {
	total, err := root.ClusterDataStore.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	unhealthy, err := root.ClusterDataStore.Search(ctx, search.ConjunctionQuery(q, search.NewQueryBuilder().AddExactMatches(search.ClusterStatus, storage.ClusterHealthStatus_UNHEALTHY.String()).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	degraded, err := root.ClusterDataStore.Search(ctx, search.ConjunctionQuery(q, search.NewQueryBuilder().AddExactMatches(search.ClusterStatus, storage.ClusterHealthStatus_DEGRADED.String()).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	healthy, err := root.ClusterDataStore.Search(ctx, search.ConjunctionQuery(q, search.NewQueryBuilder().AddExactMatches(search.ClusterStatus, storage.ClusterHealthStatus_HEALTHY.String()).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	resolver := &ClusterHealthCounterResolver{
		total:     int32(len(total)),
		healthy:   int32(len(healthy)),
		degraded:  int32(len(degraded)),
		unhealthy: int32(len(unhealthy)),
	}
	resolver.uninitialized = resolver.total - resolver.healthy - resolver.degraded - resolver.unhealthy
	return resolver, nil
}

// Total returns total the number of clusters.
func (cr *ClusterHealthCounterResolver) Total(_ context.Context) int32 {
	return cr.total
}

// Uninitialized returns the number of clusters that are in uninitialized state.
func (cr *ClusterHealthCounterResolver) Uninitialized(_ context.Context) int32 {
	return cr.uninitialized
}

// Healthy returns the number of clusters that are in healthy state.
func (cr *ClusterHealthCounterResolver) Healthy(_ context.Context) int32 {
	return cr.healthy
}

// Degraded returns the number of clusters that are in degraded state.
func (cr *ClusterHealthCounterResolver) Degraded(_ context.Context) int32 {
	return cr.degraded
}

// Unhealthy returns the number of clusters that are in unhealthy state.
func (cr *ClusterHealthCounterResolver) Unhealthy(_ context.Context) int32 {
	return cr.unhealthy
}
