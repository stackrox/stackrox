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
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
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

// indicatorDeleteHandler is an entity that need to execute an operation when an entity is deleted
type indicatorDeleteHandler interface {
	HandlePurgedConnectionIndicator(conn *indicator.NetworkConn)
	HandlePurgedEndpointIndicator(ep *indicator.ContainerEndpoint)
}

// WithIndicatorDeleteHandler sets the delete handler for the purger
func WithIndicatorDeleteHandler(handler indicatorDeleteHandler) PurgerOption {
	return func(purger *NetworkFlowPurger) {
		purger.indicatorDeleteHandler = handler
	}
}

// deleteHandler is an entity that need to execute an operation when an entity is deleted
type deleteHandler interface {
	HandlePurgedConnection(conn *connection)
	HandlePurgedEndpoint(ep *containerEndpoint)
}

// WithDeleteHandler sets the delete handler for the purger
func WithDeleteHandler(handler deleteHandler) PurgerOption {
	return func(purger *NetworkFlowPurger) {
		purger.deleteHandler = handler
	}
}

type purgerDataSource interface {
	GetActiveConnections() (mutex *sync.RWMutex, activeConnections map[connection]*networkConnIndicatorWithAge)
	GetActiveEndpoints() (mutex *sync.RWMutex, activeEndpoints map[containerEndpoint]*containerEndpointIndicatorWithAge)
	GetHostConns() (mutex *sync.RWMutex, hostConns map[string]*hostConnections)
}

// WithDataSource sets the data source that the purger analyses to possibly generate deletions
func WithDataSource(src purgerDataSource) PurgerOption {
	return func(purger *NetworkFlowPurger) {
		purger.purgerDataSource = src
	}
}

type NetworkFlowPurger struct {
	maxAge           time.Duration
	clusterEntities  EntityStore
	purgerDataSource purgerDataSource

	indicatorDeleteHandler indicatorDeleteHandler
	deleteHandler          deleteHandler

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
		clusterEntities:        clusterEntities,
		purgerTicker:           purgerTicker,
		purgerTickerC:          purgerTicker.C,
		maxAge:                 maxAge,
		stopper:                concurrency.NewStopper(),
		purgingDone:            concurrency.NewSignal(),
		indicatorDeleteHandler: nil,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *NetworkFlowPurger) Start() error {
	if p.purgerDataSource == nil {
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

func (p *NetworkFlowPurger) Stop() {
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
	numPurgedActiveConn, numPurgedActiveEp := 0, 0
	wg := concurrency.NewWaitGroup(2)

	// Endpoints
	mutex, activeEndpoints := p.purgerDataSource.GetActiveEndpoints()
	toDeleteEp := purgeActiveEndpoints(mutex, p.maxAge, activeEndpoints, p.clusterEntities)
	go p.handlePurgedEndpoints(&wg, toDeleteEp, &numPurgedActiveEp)

	// Connections
	mutex, activeConnections := p.purgerDataSource.GetActiveConnections()
	toDeleteConn := purgeActiveConnections(mutex, p.maxAge, activeConnections, p.clusterEntities)
	go p.handlePurgedConnections(&wg, toDeleteConn, &numPurgedActiveConn)

	// Legacy update computer may accumulate entities in the hostConnections map (enrichment queue), thus purging is needed.
	mutex, hostConns := p.purgerDataSource.GetHostConns()
	numPurgedHostEp, numPurgedHostConn := purgeHostConns(mutex, p.maxAge, hostConns, p.clusterEntities)
	<-wg.Done()
	log.Debugf("Purger deleted: "+
		"%d active endpoints, %d active connections, "+
		"%d host endpoints, %d host connections",
		numPurgedActiveEp, numPurgedActiveConn, numPurgedHostEp, numPurgedHostConn)
}

func (p *NetworkFlowPurger) handlePurgedEndpoints(wg *concurrency.WaitGroup, toDeleteEp <-chan epPair, numPurged *int) {
	defer wg.Add(-1)
	for ep := range toDeleteEp {
		*numPurged++
		if p.indicatorDeleteHandler != nil {
			p.indicatorDeleteHandler.HandlePurgedEndpointIndicator(ep.epInd)
		}
		if p.deleteHandler != nil {
			p.deleteHandler.HandlePurgedEndpoint(ep.ep)
		}
	}
}

func (p *NetworkFlowPurger) handlePurgedConnections(wg *concurrency.WaitGroup, toDeleteConn <-chan connPair, numPurged *int) {
	defer wg.Add(-1)
	for conn := range toDeleteConn {
		*numPurged++
		if p.indicatorDeleteHandler != nil {
			p.indicatorDeleteHandler.HandlePurgedConnectionIndicator(conn.connInd)
		}
		if p.deleteHandler != nil {
			p.deleteHandler.HandlePurgedConnection(conn.conn)
		}
	}
}

func purgeHostConns(mutex *sync.RWMutex, maxAge time.Duration, enrichmentQueue map[string]*hostConnections, store EntityStore) (numPurgedEps, numPurgedConns int) {
	timer := prometheus.NewTimer(flowMetrics.PurgerRunDuration.WithLabelValues("hostConns"))
	defer timer.ObserveDuration()
	numPurgedEps = 0
	numPurgedConns = 0
	concurrency.WithRLock(mutex, func() {
		for _, c := range enrichmentQueue {
			concurrency.WithLock(&c.mutex, func() {
				numPurgedEps += purgeHostConnsEndpointsNoLock(maxAge, c, store)
				numPurgedConns += purgeHostConnsConnectionsNoLock(maxAge, c, store)
			})
		}
	})
	return numPurgedEps, numPurgedConns
}

func purgeHostConnsEndpointsNoLock(maxAge time.Duration, conns *hostConnections, store EntityStore) (numPurgedEps int) {
	numPurgedEps = 0
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
	return numPurgedEps
}
func purgeHostConnsConnectionsNoLock(maxAge time.Duration, conns *hostConnections, store EntityStore) (numPurgedConns int) {
	numPurgedConns = 0
	cutOff := timestamp.Now().Add(-maxAge)
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
	return numPurgedConns
}

// epPair groups two variants of storing data about container endpoint. Used only internally to trigger deletion.
type epPair struct {
	ep    *containerEndpoint
	epInd *indicator.ContainerEndpoint
}

func purgeActiveEndpoints(mutex *sync.RWMutex, maxAge time.Duration, activeEndpoints map[containerEndpoint]*containerEndpointIndicatorWithAge, store EntityStore) <-chan epPair {
	timer := prometheus.NewTimer(flowMetrics.PurgerRunDuration.WithLabelValues("activeEndpoints"))
	defer timer.ObserveDuration()
	return concurrency.WithRLock1(mutex, func() <-chan epPair {
		log.Debug("Purging active endpoints")
		return purgeActiveEndpointsNoLock(maxAge, activeEndpoints, store)
	})
}

func purgeActiveEndpointsNoLock(maxAge time.Duration,
	endpoints map[containerEndpoint]*containerEndpointIndicatorWithAge,
	store EntityStore) <-chan epPair {
	toDelete := make(chan epPair)
	cutOff := timestamp.Now().Add(-maxAge)
	go func() {
		defer close(toDelete)
		for endpoint, ind := range endpoints {
			// Remove if the related container is not found (but keep historical) and endpoint is unknown
			_, contIDfound, _ := store.LookupByContainerID(endpoint.containerID)
			endpointFound := len(store.LookupByEndpoint(endpoint.endpoint)) > 0
			if !contIDfound && !endpointFound {
				// Make sure that Sensor knows absolutely nothing about that endpoint.
				// There is still a chance that endpoint maybe unknown, but we know the container ID
				// and this is sufficient to make the plop feature work.
				flowMetrics.PurgerEvents.WithLabelValues("activeEndpoint", "endpoint-&-containerID-gone").Inc()
				toDelete <- epPair{
					ep:    &endpoint,
					epInd: &ind.ContainerEndpoint,
				}
				continue
			}
			if maxAge > 0 {
				// finally, remove all that didn't get any update from collector for a given time
				if cutOff.After(ind.lastUpdate) {
					flowMetrics.PurgerEvents.WithLabelValues("activeEndpoint", "max-age-reached").Inc()
					toDelete <- epPair{
						ep:    &endpoint,
						epInd: &ind.ContainerEndpoint,
					}
				}
			}
		}
	}()
	return toDelete
}

// connPair groups two variants of storing data about connection. Used only internally to trigger deletion.
type connPair struct {
	conn    *connection
	connInd *indicator.NetworkConn
}

func purgeActiveConnections(mutex *sync.RWMutex, maxAge time.Duration, activeConnections map[connection]*networkConnIndicatorWithAge, store EntityStore) <-chan connPair {
	timer := prometheus.NewTimer(flowMetrics.PurgerRunDuration.WithLabelValues("activeConnections"))
	defer timer.ObserveDuration()
	return concurrency.WithRLock1(mutex, func() <-chan connPair {
		log.Debug("Purging active connections")
		return purgeActiveConnectionsNoLock(maxAge, activeConnections, store)
	})
}

func purgeActiveConnectionsNoLock(maxAge time.Duration,
	conns map[connection]*networkConnIndicatorWithAge,
	store EntityStore) <-chan connPair {
	toDelete := make(chan connPair)
	cutOff := timestamp.Now().Add(-maxAge)
	go func() {
		defer close(toDelete)
		for conn, ind := range conns {
			// Remove if the related container is not found (but keep historical)
			_, found, _ := store.LookupByContainerID(conn.containerID)
			if !found {
				flowMetrics.PurgerEvents.WithLabelValues("activeConnection", "containerID-gone").Inc()
				toDelete <- connPair{
					conn:    &conn,
					connInd: &ind.NetworkConn,
				}
				continue
			}
			if maxAge > 0 {
				// finally, remove all that didn't get any update from collector for a given time
				if cutOff.After(ind.lastUpdate) {
					flowMetrics.PurgerEvents.WithLabelValues("activeConnection", "max-age-reached").Inc()
					toDelete <- connPair{
						conn:    &conn,
						connInd: &ind.NetworkConn,
					}
				}
			}
		}
	}()
	return toDelete
}
