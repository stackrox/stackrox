package manager

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

type PurgerOption func(purger *NetworkFlowPurger)

// WithPurgerTicker overrides the default enrichment ticker - use only for testing!
func WithPurgerTicker(_ *testing.T, ticker <-chan time.Time) PurgerOption {
	return func(purger *NetworkFlowPurger) {
		if ticker != nil {
			purger.purgerTickerC = ticker
		}
	}
}

// WithManager binds the purger to the network flow manager
func WithManager(mgr *networkFlowManager) PurgerOption {
	return func(purger *NetworkFlowPurger) {
		purger.manager = mgr
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

// NewNetworkFlowPurger implements Sensor Component and is tightly bound to the networkFlowManager.
// It can start in any order with relation to the networkFlowManager. The binding of networkFlowManager and the purger
// is done by using the `WithPurger` option when constructing the manager: `manager.NewManager(..., manager.WithPurger(purger))`.
// The purger is designed to always consume the messages from `purgerTickerC` - even if the binding to networkFlowManager
// fails or the purger is explicitly disabled using env var.
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

func (p *NetworkFlowPurger) Start() error {
	if p.manager == nil {
		p.stopper.Flow().ReportStopped() // to ensure that Stop doesn't block
		return errors.New("programmer error: network flow purger is not bound to a network flow manager")
	}
	if env.EnrichmentPurgerTickerCycle.DurationSetting() == 0 {
		p.stopper.Flow().ReportStopped() // to ensure that Stop doesn't block
		return errors.New("network flow purger is disabled")
	}

	// Allow starting the purger without a manager. This is done to prevent blocking of the entire component when
	// `purgerTickerC` receives a message
	go p.run()
	return nil
}

func (p *NetworkFlowPurger) Stop(_ error) {
	p.purgerTicker.Stop()
	if !p.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = p.stopper.Client().Stopped().Wait()
		}()
	}
	p.stopper.Client().Stop()
}

// nonZeroPurgerCycle delivers a non-zero duration to be used in timers (they panic when set with 0 duration)
func nonZeroPurgerCycle() time.Duration {
	purgerCycleSetting := env.EnrichmentPurgerTickerCycle.DurationSetting()
	if purgerCycleSetting > 0 {
		return purgerCycleSetting
	}
	// Disabled purger will wake up every 71 minutes and execute a noop.
	// We use 71; a prime number higher than 60 (but not too close to it - there maybe many things happening every hour),
	// so that it is easier to detect and locate a potential source of a problem if something happens every 71 minutes.
	return 71 * time.Minute
}

func (p *NetworkFlowPurger) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, "NetworkFlowPurger"))

	switch e {
	// Purger could start earlier than this, but we stick to the `SensorComponentEventResourceSyncFinished` as it also
	// enables the networkFlowManager.
	case common.SensorComponentEventResourceSyncFinished:
		d := nonZeroPurgerCycle()
		p.purgerTicker.Reset(d)
		log.Debugf("NetworkFlowPurger will execute in %s", d.String())
	case common.SensorComponentEventOfflineMode:
		if !features.SensorCapturesIntermediateEvents.Enabled() {
			p.purgerTicker.Stop()
		}
	}
}

func (p *NetworkFlowPurger) run() {
	defer p.stopper.Flow().ReportStopped()
	for {
		select {
		case <-p.stopper.Flow().StopRequested():
			return
		case _, chanOpen := <-p.purgerTickerC:
			p.purgingDone.Reset()
			// Do not execute potentially-expensive purger rules when ticker channel is closed.
			if chanOpen {
				p.runPurger()
			}
			p.purgingDone.Signal()
		}
	}
}

func (p *NetworkFlowPurger) runPurger() {
	numPurgedActiveEp := purgeActiveEndpoints(&p.manager.activeEndpointsMutex, p.maxAge, p.manager.activeEndpoints, p.clusterEntities)
	numPurgedActiveConn := purgeActiveConnections(&p.manager.activeConnectionsMutex, p.maxAge, p.manager.activeConnections, p.clusterEntities)
	numPurgedHostEp, numPurgedHostConn := purgeHostConns(&p.manager.connectionsByHostMutex, p.maxAge, p.manager.connectionsByHost, p.clusterEntities)
	log.Debugf("Purger deleted: "+
		"%d active endpoints, %d active connections, "+
		"%d host endpoints, %d host connections",
		numPurgedActiveEp, numPurgedActiveConn, numPurgedHostEp, numPurgedHostConn)
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
	cutOff := timestamp.Now().Add(-maxAge)
	for endpoint, status := range conns.endpoints {
		// Remove if the related container is not found (but keep historical) and endpoint is unknown
		_, contIDfound, _ := store.LookupByContainerID(endpoint.containerID)
		endpointFound := len(store.LookupByEndpoint(endpoint.endpoint)) > 0
		if !contIDfound && !endpointFound {
			// Make sure that Sensor knows absolutely nothing about that endpoint.
			// There is still a chance that endpoint maybe unknown, but we know the container ID
			// and this is sufficient to make the plop feature work.
			flowMetrics.PurgerEvents.WithLabelValues("hostEndpoint", "endpoint-&-containerID-gone").Inc()
			delete(conns.endpoints, endpoint)
			numPurgedEps++
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
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
			flowMetrics.PurgerEvents.WithLabelValues("hostConnection", "containerID-gone").Inc()
			delete(conns.connections, conn)
			numPurgedConns++
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
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
	cutOff := timestamp.Now().Add(-maxAge)
	for endpoint, age := range endpoints {
		// Remove if the related container is not found (but keep historical) and endpoint is unknown
		_, contIDfound, _ := store.LookupByContainerID(endpoint.containerID)
		endpointFound := len(store.LookupByEndpoint(endpoint.endpoint)) > 0
		if !contIDfound && !endpointFound {
			// Make sure that Sensor knows absolutely nothing about that endpoint.
			// There is still a chance that endpoint maybe unknown, but we know the container ID
			// and this is sufficient to make the plop feature work.
			flowMetrics.PurgerEvents.WithLabelValues("activeEndpoint", "endpoint-&-containerID-gone").Inc()
			delete(endpoints, endpoint)
			numPurged++
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
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
	cutOff := timestamp.Now().Add(-maxAge)
	for conn, age := range conns {
		// Remove if the related container is not found (but keep historical)
		_, found, _ := store.LookupByContainerID(conn.containerID)
		if !found {
			flowMetrics.PurgerEvents.WithLabelValues("activeConnection", "containerID-gone").Inc()
			delete(conns, conn)
			numPurged++
			continue
		}
		if maxAge > 0 {
			// finally, remove all that didn't get any update from collector for a given time
			if cutOff.After(age.lastUpdate) {
				flowMetrics.PurgerEvents.WithLabelValues("activeConnection", "max-age-reached").Inc()
				delete(conns, conn)
				numPurged++
			}
		}
	}
	return numPurged
}
