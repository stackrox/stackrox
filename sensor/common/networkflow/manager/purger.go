package manager

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

var (
	// How often purger should run. The Purger removes old endpoints from activeEndpoints slice.
	// This is important for cases when Collector or the orchestrator never reports a given endpoint
	// as deleted, because there is no other mechanism that would remove an endpoint from memory.
	purgerCycleSetting = env.EnrichmentPurgerTickerCycle.DurationSetting()
)

type PurgerOption func(purger *NetworkFlowPurger)

// WithPurgerTicker overrides the default enrichment ticker
func WithPurgerTicker(ticker <-chan time.Time) PurgerOption {
	return func(purger *NetworkFlowPurger) {
		if ticker != nil {
			purger.purgerTickerC = ticker
		}
	}
}

type NetworkFlowPurger struct {
	// enrichmentQueue represents connectionsByHost in the network manager.
	enrichmentQueue      map[string]*hostConnections
	enrichmentQueueMutex *sync.Mutex

	clusterEntities EntityStore

	activeConnectionsMutex *sync.Mutex
	// activeConnections tracks all connections reported by Collector that are believed to be active.
	// A connection is active until Collector sends a NetworkConnectionInfo message with `lastSeen` set to a non-nil value,
	// or until Sensor decides that such message may never arrive and decides that a given connection is no longer active.
	activeConnections    map[connection]*networkConnIndicatorWithAge
	activeEndpointsMutex *sync.Mutex
	// An endpoint is active until Collector sends a NetworkConnectionInfo message with `lastSeen` set to a non-nil value,
	// or until Sensor decides that such message may never arrive and decides that a given endpoint is no longer active.
	activeEndpoints map[containerEndpoint]*containerEndpointIndicatorWithAge

	purgerTicker  *time.Ticker
	purgerTickerC <-chan time.Time

	stopper concurrency.Stopper
}

func (p *NetworkFlowPurger) Start() error {
	if p.activeConnectionsMutex == nil {
		return fmt.Errorf("cannot start network flow purger without active connections mutex")
	}
	if p.activeConnections == nil {
		return fmt.Errorf("cannot start network flow purger without active connections")
	}
	if p.activeEndpointsMutex == nil {
		return fmt.Errorf("cannot start network flow purger without active endpoints mutex")
	}
	if p.activeEndpoints == nil {
		return fmt.Errorf("cannot start network flow purger without active endpoints")
	}
	if p.enrichmentQueueMutex == nil {
		return fmt.Errorf("cannot start network flow purger without enrichment queue mutex")
	}
	if p.enrichmentQueue == nil {
		return fmt.Errorf("cannot start network flow purger without enrichment queue")
	}
	go p.start(env.EnrichmentPurgerTickerMaxAge.DurationSetting())
	return nil
}

func (p *NetworkFlowPurger) Stop(_ error) {
	if !p.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = p.stopper.Client().Stopped().Wait()
		}()
	}
	p.stopper.Client().Stop()
}

func (p *NetworkFlowPurger) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{}
}

func (p *NetworkFlowPurger) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (p *NetworkFlowPurger) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}

func nonZeroPurgerCycle() time.Duration {
	if purgerCycleSetting > 0 {
		return purgerCycleSetting
	}
	return time.Hour
}

func NewNetworkFlowPurger(clusterEntities EntityStore, opts ...PurgerOption) *NetworkFlowPurger {
	purgerTicker := time.NewTicker(nonZeroPurgerCycle())
	defer purgerTicker.Stop()

	p := &NetworkFlowPurger{
		clusterEntities:        clusterEntities,
		enrichmentQueue:        nil,
		enrichmentQueueMutex:   &sync.Mutex{},
		activeConnections:      nil,
		activeConnectionsMutex: &sync.Mutex{},
		activeEndpoints:        nil,
		activeEndpointsMutex:   &sync.Mutex{},

		purgerTicker:  purgerTicker,
		purgerTickerC: purgerTicker.C,
		stopper:       nil,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *NetworkFlowPurger) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, "NetworkFlowPurger"))

	switch e {
	case common.SensorComponentEventResourceSyncFinished:
		p.purgerTicker.Reset(nonZeroPurgerCycle())
	case common.SensorComponentEventOfflineMode:
		if !features.SensorCapturesIntermediateEvents.Enabled() {
			p.purgerTicker.Stop()
		}
	}
}

func (p *NetworkFlowPurger) start(maxAge time.Duration) {
	if env.EnrichmentPurgerTickerCycle.DurationSetting() == 0 {
		return
	}
	for {
		select {
		case <-p.stopper.Flow().StopRequested():
			return
		case <-p.purgerTickerC:
			numPurgedActiveEp := purgeActiveEndpoints(p.activeEndpointsMutex, maxAge, p.activeEndpoints, p.clusterEntities)
			numPurgedActiveConn := purgeActiveConnections(p.activeConnectionsMutex, maxAge, p.activeConnections, p.clusterEntities)
			numPurgedHostEp, numPurgedHostConn := purgeHostConns(p.enrichmentQueueMutex, maxAge, p.enrichmentQueue, p.clusterEntities)
			log.Debugf("Purger deleted: "+
				"%d active endpoints, %d active connections, "+
				"%d host endpoints, %d host connections",
				numPurgedActiveEp, numPurgedActiveConn, numPurgedHostEp, numPurgedHostConn)
		}
	}
}

func purgeHostConns(mutex *sync.Mutex, maxAge time.Duration, enrichmentQueue map[string]*hostConnections, store EntityStore) (numPurgedEps, numPurgedConns int) {
	timer := prometheus.NewTimer(flowMetrics.ActiveEndpointsPurgerDuration.WithLabelValues("hostConns"))
	defer timer.ObserveDuration()
	numPurgedEps = 0
	numPurgedConns = 0
	concurrency.WithLock(mutex, func() {
		for _, c := range enrichmentQueue {
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
