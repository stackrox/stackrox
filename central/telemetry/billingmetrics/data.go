package billingmetrics

import (
	"context"
	"time"

	"github.com/pkg/errors"
	bmetrics "github.com/stackrox/rox/central/billingmetrics"
	cluStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/pipeline/clustermetrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
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
		a.TotalMilliCores += m.TotalMilliCores
	}
	a.TotalNodes /= n
	a.TotalMilliCores /= n
	return a
}

func getClusterIDs() (set.StringSet, error) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), clusterReader)

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

	_, err := bmetrics.Singleton().PutMetrics(ctx, &v1.BillingMetricsInsertRequest{
		Ts: protoconv.ConvertTimeToTimestamp(time.Now()),
		Metrics: &v1.SecuredResourcesMetrics{
			Nodes:      int32(metrics.TotalNodes),
			Millicores: int32(metrics.TotalMilliCores)},
	})
	return errors.Wrap(err, "billing metrics datastore failure")
}
