package manager

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/timestamp"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

func (m *networkFlowManager) purgeStaleEndpoints(tickerC <-chan time.Time) {
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		case <-tickerC:
			maxAge := env.EnrichmentPurgerTickerMaxAge.DurationSetting()
			start := time.Now()
			concurrency.WithLock(&m.connectionsByHostMutex, func() {
				for _, conns := range m.connectionsByHost {
					concurrency.WithLock(&conns.mutex, func() {
						purgeHostConnsNoLock(maxAge, conns, m.clusterEntities)
					})
				}
			})
			flowMetrics.ActiveEndpointsPurgerDuration.WithLabelValues("hostConns").Observe(float64(time.Since(start).Milliseconds()))
		}
	}
}

func purgeHostConnsNoLock(maxAge time.Duration, conns *hostConnections, store EntityStore) {
	for endpoint, status := range conns.endpoints {
		// remove if the endpoint is not in the store (also not in history)
		if len(store.LookupByEndpoint(endpoint.endpoint)) == 0 {
			delete(conns.endpoints, endpoint)
			flowMetrics.PurgerEvents.WithLabelValues("hostEndpoint", "endpoint-gone").Inc()
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
			cutOff := timestamp.Now().Add(-maxAge)
			if cutOff.After(status.tsAdded) {
				flowMetrics.PurgerEvents.WithLabelValues("hostEndpoint", "max-age-reached").Inc()
				delete(conns.endpoints, endpoint)
			}
		}
	}
	for conn, status := range conns.connections {
		// Remove if the related container is not found (but keep historical)
		_, found, _ := store.LookupByContainerID(conn.containerID)
		if !found {
			delete(conns.connections, conn)
			flowMetrics.PurgerEvents.WithLabelValues("hostConnection", "containerID-gone").Inc()
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
			cutOff := timestamp.Now().Add(-maxAge)
			if cutOff.After(status.tsAdded) {
				flowMetrics.PurgerEvents.WithLabelValues("hostConnection", "max-age-reached").Inc()
				delete(conns.connections, conn)
			}
		}
	}
}
