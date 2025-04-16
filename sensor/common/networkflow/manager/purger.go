package manager

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

func nonZeroPurgerCycle() time.Duration {
	if purgerCycleSetting > 0 {
		return purgerCycleSetting
	}
	return time.Hour
}

func (m *networkFlowManager) notifyPurger(e common.SensorComponentEvent) {
	switch e {
	case common.SensorComponentEventResourceSyncFinished:
		m.purgerTicker.Reset(nonZeroPurgerCycle())
	case common.SensorComponentEventOfflineMode:
		if !features.SensorCapturesIntermediateEvents.Enabled() {
			m.purgerTicker.Stop()
		}
	}
}

func (m *networkFlowManager) startPurger(tickerC <-chan time.Time, maxAge time.Duration) {
	if env.EnrichmentPurgerTickerCycle.DurationSetting() == 0 {
		return
	}
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		case <-tickerC:
			numPurgedActiveEp := purgeActiveEndpoints(&m.activeEndpointsMutex, maxAge, m.activeEndpoints, m.clusterEntities)
			numPurgedActiveConn := purgeActiveConnections(&m.activeConnectionsMutex, maxAge, m.activeConnections, m.clusterEntities)
			numPurgedHostEp, numPurgedHostConn := purgeHostConns(&m.connectionsByHostMutex, maxAge, m.connectionsByHost, m.clusterEntities)
			log.Debugf("Purger deleted: "+
				"%d active endpoints, %d active connections, "+
				"%d host endpoints, %d host connections",
				numPurgedActiveEp, numPurgedActiveConn, numPurgedHostEp, numPurgedHostConn)
		}
	}
}

func purgeHostConns(mutex *sync.Mutex, maxAge time.Duration, hostConns map[string]*hostConnections, store EntityStore) (numPurgedEps, numPurgedConns int) {
	timer := prometheus.NewTimer(flowMetrics.ActiveEndpointsPurgerDuration.WithLabelValues("hostConns"))
	defer timer.ObserveDuration()
	numPurgedEps = 0
	numPurgedConns = 0
	concurrency.WithLock(mutex, func() {
		for _, c := range hostConns {
			concurrency.WithLock(&c.mutex, func() {
				npe, npc := purgeHostConnsNoLock(maxAge, c, store)
				numPurgedEps += npe
				numPurgedConns += npc
			})
		}
	})
	return numPurgedEps, numPurgedConns
}

func purgeHostConnsNoLock(maxAge time.Duration, conns *hostConnections, store EntityStore) (numPurgedEps, numPurgedConns int) {
	numPurgedEps = 0
	numPurgedConns = 0
	for endpoint, status := range conns.endpoints {
		// remove if the endpoint is not in the store (also not in history)
		if len(store.LookupByEndpoint(endpoint.endpoint)) == 0 {
			delete(conns.endpoints, endpoint)
			numPurgedEps++
			flowMetrics.PurgerEvents.WithLabelValues("hostEndpoint", "endpoint-gone").Inc()
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
			cutOff := timestamp.Now().Add(-maxAge)
			if cutOff.After(status.tsAdded) {
				flowMetrics.PurgerEvents.WithLabelValues("hostEndpoint", "max-age-reached").Inc()
				delete(conns.endpoints, endpoint)
				numPurgedEps++
			}
		}
	}
	for conn, status := range conns.connections {
		// Remove if the related container is not found (but keep historical)
		_, found, _ := store.LookupByContainerID(conn.containerID)
		if !found {
			delete(conns.connections, conn)
			flowMetrics.PurgerEvents.WithLabelValues("hostConnection", "containerID-gone").Inc()
			numPurgedConns++
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
			cutOff := timestamp.Now().Add(-maxAge)
			if cutOff.After(status.tsAdded) {
				flowMetrics.PurgerEvents.WithLabelValues("hostConnection", "max-age-reached").Inc()
				delete(conns.connections, conn)
				numPurgedConns++
			}
		}
	}
	return numPurgedEps, numPurgedConns
}

func purgeActiveEndpoints(mutex *sync.Mutex, maxAge time.Duration, activeEndpoints map[containerEndpoint]*containerEndpointIndicatorWithAge, store EntityStore) int {
	timer := prometheus.NewTimer(flowMetrics.ActiveEndpointsPurgerDuration.WithLabelValues("activeEndpoints"))
	defer timer.ObserveDuration()
	return concurrency.WithLock1(mutex, func() int {
		log.Debug("Purging active endpoints")
		return purgeActiveEndpointsNoLock(maxAge, activeEndpoints, store)
	})
}

func purgeActiveEndpointsNoLock(maxAge time.Duration,
	endpoints map[containerEndpoint]*containerEndpointIndicatorWithAge,
	store EntityStore) int {
	numPurged := 0
	for endpoint, age := range endpoints {
		// Remove if the endpoint is not in the store (also not in history)
		if len(store.LookupByEndpoint(endpoint.endpoint)) == 0 {
			delete(endpoints, endpoint)
			numPurged++
			flowMetrics.PurgerEvents.WithLabelValues("activeEndpoint", "endpoint-gone").Inc()
			continue
		}
		// Remove if the related container is not found (but keep historical)
		_, found, _ := store.LookupByContainerID(endpoint.containerID)
		if !found {
			delete(endpoints, endpoint)
			numPurged++
			flowMetrics.PurgerEvents.WithLabelValues("activeEndpoint", "containerID-gone").Inc()
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
			cutOff := timestamp.Now().Add(-maxAge)
			if cutOff.After(age.lastUpdate) {
				flowMetrics.PurgerEvents.WithLabelValues("activeEndpoint", "max-age-reached").Inc()
				delete(endpoints, endpoint)
				numPurged++
			}
		}
	}
	return numPurged
}

func purgeActiveConnections(mutex *sync.Mutex, maxAge time.Duration, activeConnections map[connection]*networkConnIndicatorWithAge, store EntityStore) int {
	timer := prometheus.NewTimer(flowMetrics.ActiveEndpointsPurgerDuration.WithLabelValues("activeConnections"))
	defer timer.ObserveDuration()
	return concurrency.WithLock1(mutex, func() int {
		log.Debug("Purging active connections")
		return purgeActiveConnectionsNoLock(maxAge, activeConnections, store)
	})
}

func purgeActiveConnectionsNoLock(maxAge time.Duration,
	conns map[connection]*networkConnIndicatorWithAge,
	store EntityStore) int {
	numPurged := 0
	for conn, age := range conns {
		// Remove if the related container is not found (but keep historical)
		_, found, _ := store.LookupByContainerID(conn.containerID)
		if !found {
			delete(conns, conn)
			numPurged++
			flowMetrics.PurgerEvents.WithLabelValues("activeConnection", "containerID-gone").Inc()
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
			cutOff := timestamp.Now().Add(-maxAge)
			if cutOff.After(age.lastUpdate) {
				flowMetrics.PurgerEvents.WithLabelValues("activeConnection", "max-age-reached").Inc()
				delete(conns, conn)
				numPurged++
			}
		}
	}
	return numPurged
}
