package expiry

import (
	"context"
	"time"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	creds "github.com/stackrox/rox/central/credentialexpiry/service"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

// securedClusterComponent is a synthetic component, not part of the
// v1.GetCertExpiry_Component enum: unlike Central/Scanner/Central DB/Scanner
// V4, there can be any number of secured clusters, each identified by the
// "Name" label rather than by a fixed enum value.
const securedClusterComponent = "SECURED_CLUSTER"

func New(s creds.Service, clusters clusterDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		metrics.Expiry,
		"hours before certificate expires",
		LazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) tracker.FindingErrorSequence[*finding] {
			return track(ctx, s, clusters)
		},
	)
}

func track(ctx context.Context, s creds.Service, clusters clusterDS.DataStore) tracker.FindingErrorSequence[*finding] {
	return func(yield func(*finding, error) bool) {
		if s != nil {
			if !fromCredsService(ctx, s, yield) {
				return
			}
		}
		if clusters == nil {
			return
		}
		fromClustersDS(ctx, clusters, yield)
	}
}

func fromCredsService(ctx context.Context, s creds.Service, yield func(*finding, error) bool) bool {
	for i, component := range v1.GetCertExpiry_Component_name {
		if v1.GetCertExpiry_Component(i) == v1.GetCertExpiry_UNKNOWN {
			continue
		}
		result, err := s.GetCertExpiry(ctx, &v1.GetCertExpiry_Request{
			Component: v1.GetCertExpiry_Component(i),
		})
		if err != nil {
			// Ignore particular component errors, as they do not affect
			// other components metrics.
			logging.LoggerForModule().Errorw("Failed to get certificate expiry",
				logging.String("component", component), logging.Err(err))
			continue
		}
		f := &finding{component: component}
		if result != nil {
			f.hoursUntilExpiration = int(time.Until(result.GetExpiry().AsTime()).Hours())
		}
		if !yield(f, nil) {
			return false
		}
	}
	return true
}

func fromClustersDS(ctx context.Context, clusters clusterDS.DataStore, yield func(*finding, error) bool) {
	collector := tracker.NewFindingCollector(yield)
	collector.Finally(clusters.WalkClusters(ctx, func(cluster *storage.Cluster) error {
		expiry := cluster.GetStatus().GetCertExpiryStatus().GetSensorCertExpiry()
		if expiry == nil {
			// The cluster hasn't connected yet, so there is no cert to report on.
			return nil
		}
		return collector.Yield(&finding{
			component:            securedClusterComponent,
			name:                 cluster.GetName(),
			hoursUntilExpiration: int(time.Until(expiry.AsTime()).Hours()),
		})
	}))
}
