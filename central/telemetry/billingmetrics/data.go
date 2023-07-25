package billingmetrics

import (
	"context"
	"time"

	"github.com/pkg/errors"
	bmetrics "github.com/stackrox/rox/central/billingmetrics/store"
	cluStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/sensor/service/pipeline/clustermetrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
)

var (
	clusterReader = sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Cluster))

	metricsWriter = sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Administration))

	previousMetrics = &clustermetrics.BillingMetrics{}
)

func average(metrics ...*clustermetrics.BillingMetrics) clustermetrics.BillingMetrics {
	n := int64(len(metrics))
	a := clustermetrics.BillingMetrics{}
	if n == 0 {
		return a
	}
	for _, m := range metrics {
		a.TotalNodes += m.TotalNodes
		a.TotalCores += m.TotalCores
	}
	a.TotalNodes /= n
	a.TotalCores /= n
	return a
}

func getClusterIDs(ctx context.Context) (set.StringSet, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx, clusterReader)

	clusters, err := cluStore.Singleton().GetClusters(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cluster datastore failure")
	}
	ids := set.NewStringSet()
	for _, cluster := range clusters {
		ids.Add(cluster.GetId())
	}
	return ids, nil
}

func checkIn(metrics clustermetrics.BillingMetrics) error {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), metricsWriter)

	err := bmetrics.Singleton().Insert(ctx, &storage.BillingMetrics{
		Ts: protoconv.ConvertTimeToTimestamp(time.Now()),
		Sr: &storage.BillingMetrics_SecuredResources{
			Nodes: int32(metrics.TotalNodes),
			Cores: int32(metrics.TotalCores)},
	})
	return errors.Wrap(err, "billing metrics datastore failure")
}
