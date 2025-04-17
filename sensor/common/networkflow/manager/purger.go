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
	maxAge          time.Duration
	clusterEntities EntityStore
	manager         *networkFlowManager

	purgerTicker  *time.Ticker
	purgerTickerC <-chan time.Time

	stopper concurrency.Stopper
	// purgingDone is signaled on each finished purging action
	purgingDone concurrency.Signal
}

func (p *NetworkFlowPurger) Start() error {
	var err error
	if p.manager == nil {
		err = fmt.Errorf("programmer error: network flow purger is not bound to a network flow manager")
	}
	// Allow starting the purger without a manager. This is done to prevent blocking of the entire component when
	// `purgerTickerC` receives a message
	go p.start()
	return err
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

func NewNetworkFlowPurger(clusterEntities EntityStore, maxAge time.Duration, opts ...PurgerOption) *NetworkFlowPurger {
	purgerTicker := time.NewTicker(nonZeroPurgerCycle())
	defer purgerTicker.Stop()

	p := &NetworkFlowPurger{
		clusterEntities: clusterEntities,
		manager:         nil,
		purgerTicker:    purgerTicker,
		purgerTickerC:   purgerTicker.C,
		maxAge:          maxAge,
		stopper:         concurrency.NewStopper(),
		purgingDone:     concurrency.NewSignal(),
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

func (p *NetworkFlowPurger) start() {
	if env.EnrichmentPurgerTickerCycle.DurationSetting() == 0 {
		return
	}
	for {
		select {
		case <-p.stopper.Flow().StopRequested():
			return
		case _, ok := <-p.purgerTickerC:
			p.purgingDone.Reset()
			if !ok {
				// Do not execute potentially-expensive purger rules when ticker channel is closed.
				continue
			}
			if p.manager == nil {
				log.Warn("Programmer error: network flow purger is not bound to a network flow manager. Not purging.")
				p.purgingDone.Signal()
				continue
			}
			numPurgedActiveEp := purgeActiveEndpoints(&p.manager.activeEndpointsMutex, p.maxAge, p.manager.activeEndpoints, p.clusterEntities)
			numPurgedActiveConn := purgeActiveConnections(&p.manager.activeConnectionsMutex, p.maxAge, p.manager.activeConnections, p.clusterEntities)
			numPurgedHostEp, numPurgedHostConn := purgeHostConns(&p.manager.connectionsByHostMutex, p.maxAge, p.manager.connectionsByHost, p.clusterEntities)
			log.Debugf("Purger deleted: "+
				"%d active endpoints, %d active connections, "+
				"%d host endpoints, %d host connections",
				numPurgedActiveEp, numPurgedActiveConn, numPurgedHostEp, numPurgedHostConn)
		}
		p.purgingDone.Signal()
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
