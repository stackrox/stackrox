package manager

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/unimplemented"
	"golang.org/x/exp/maps"
)

// NewCombinedManager creates a new instance of network flow manager
func NewCombinedManager(
	managerLegacy Manager,
	managerCurrent Manager,
) Manager {
	return &combinedNetworkFlowManager{
		manL:             managerLegacy,
		hostConnectionsL: make(map[string]*hostConnections),
		manC:             managerCurrent,
		hostConnectionsC: make(map[string]*hostConnections),
		enrichmentQueue:  make(map[string]*hostConnections),
		enricherTickerC:  make(chan time.Time),
		stopper:          concurrency.NewStopper(),
	}
}

var _ Manager = (*combinedNetworkFlowManager)(nil)

type combinedNetworkFlowManager struct {
	unimplemented.Receiver

	enricherTickerC <-chan time.Time

	manL             Manager
	hostConnectionsL map[string]*hostConnections
	manC             Manager
	hostConnectionsC map[string]*hostConnections

	stopper concurrency.Stopper

	// Common enrichment queue
	enrichmentQueue      map[string]*hostConnections
	enrichmentQueueMutex sync.RWMutex
}

func (c *combinedNetworkFlowManager) UnregisterCollector(hostname string, sequenceID int64) {
	c.manL.UnregisterCollector(hostname, sequenceID)
	c.manC.UnregisterCollector(hostname, sequenceID)
}

func (c *combinedNetworkFlowManager) RegisterCollector(hostname string) (*hostConnections, int64) {
	c.enrichmentQueueMutex.Lock()
	defer c.enrichmentQueueMutex.Unlock()
	log.Infof("Registering collector for %s", hostname)

	conns := c.enrichmentQueue[hostname] // Collector will write to this
	if conns == nil {
		conns = &hostConnections{
			hostname:    hostname,
			connections: make(map[connection]*connStatus),
			endpoints:   make(map[containerEndpoint]*connStatus),
		}
		c.enrichmentQueue[hostname] = conns
	}

	concurrency.WithLock(&conns.mutex, func() {
		if conns.pendingDeletion != nil {
			// Note that we don't need to check the return value, since `deleteHostConnections` needs to acquire
			// m.connectionsByHostMutex. It can therefore only proceed once this function returns, in which case it will be
			// a no-op due to `pendingDeletion` being `nil`.
			conns.pendingDeletion.Stop()
			conns.pendingDeletion = nil
		}

		conns.currentSequenceID++
	})

	hcL, _ := c.manL.RegisterCollector(hostname)
	hcC, _ := c.manC.RegisterCollector(hostname)
	c.hostConnectionsL[hostname] = hcL
	c.hostConnectionsC[hostname] = hcC

	return conns, conns.currentSequenceID
}

func (c *combinedNetworkFlowManager) PublicIPsValueStream() concurrency.ReadOnlyValueStream[*sensor.IPAddressList] {
	return c.manC.PublicIPsValueStream()
}

func (c *combinedNetworkFlowManager) ExternalSrcsValueStream() concurrency.ReadOnlyValueStream[*sensor.IPNetworkList] {
	return c.manC.ExternalSrcsValueStream()
}

func (c *combinedNetworkFlowManager) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, c.Name()))
	c.manC.Notify(e)
	c.manL.Notify(e)
}

func (c *combinedNetworkFlowManager) ResponsesC() <-chan *message.ExpiringMessage {
	return c.manC.ResponsesC()
}

func (c *combinedNetworkFlowManager) Start() error {
	_ = c.manL.Start()
	_ = c.manC.Start()
	ticker := time.NewTicker(7 * time.Second)
	c.enricherTickerC = ticker.C

	go c.runCopy()

	go func() {
		for {
			select {
			case <-c.manL.ResponsesC():
				// discard message
			case <-c.stopper.Flow().StopRequested():
				return
			}
		}
	}()
	log.Infof("%s has started", c.Name())
	return nil
}

func (c *combinedNetworkFlowManager) runCopy() {
	// This takes the collector data from the enrichment queue and pushes into two enrichment queues for managers.
	// It may delay the data by one tick
	for {
		select {
		case <-c.enricherTickerC:
			c.doCopy()
		case <-c.stopper.Flow().StopRequested():
			log.Infof("%s stops the copy loop", c.Name())
			return
		}
	}
}

func (c *combinedNetworkFlowManager) doCopy() {
	c.enrichmentQueueMutex.Lock()
	defer c.enrichmentQueueMutex.Unlock()
	for hostName, conns := range c.enrichmentQueue {
		if conns != nil {
			// Copy into L
			cL := c.hostConnectionsL[hostName]
			cL.mutex.Lock()
			for conn, status := range conns.connections {
				statusCopy := *status
				// Do not overwrite existing
				if _, found := cL.connections[conn]; !found {
					cL.connections[conn] = &statusCopy
				}
			}
			for ep, status := range conns.endpoints {
				statusCopy := *status
				if _, found := cL.endpoints[ep]; !found {
					cL.endpoints[ep] = &statusCopy
				}
			}
			cL.mutex.Unlock()
			// Copy into C
			cC := c.hostConnectionsC[hostName]
			cC.mutex.Lock()
			for conn, status := range conns.connections {
				statusCopy := *status
				// Do not overwrite existing
				if _, found := cC.connections[conn]; !found {
					cC.connections[conn] = &statusCopy
				}
			}
			for ep, status := range conns.endpoints {
				statusCopy := *status
				if _, found := cC.endpoints[ep]; !found {
					cC.endpoints[ep] = &statusCopy
				}
			}
			cC.mutex.Unlock()
			maps.Clear(conns.connections)
			maps.Clear(conns.endpoints)
		}
	}
}

func (c *combinedNetworkFlowManager) Stop() {
	c.manL.Stop()
	c.manC.Stop()
	if !c.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = c.stopper.Client().Stopped().Wait()
		}()
	}
	c.stopper.Client().Stop()
}

func (c *combinedNetworkFlowManager) Capabilities() []centralsensor.SensorCapability {
	return c.manC.Capabilities()
}

func (c *combinedNetworkFlowManager) Name() string {
	return "combinedNetworkFlowManager"
}
