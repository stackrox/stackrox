package clusterid

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clusterid"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

type handlerImpl struct {
	once                          sync.Once
	clusterID                     string
	clusterIDMutex                sync.RWMutex
	clusterIDAvailable            concurrency.Signal
	isInitCertClusterID           func(string) bool
	getClusterID                  func(string, string) (string, error)
	parseClusterIDFromServiceCert func(storage.ServiceType) (string, error)

	// Fields for init certificate upgrade.
	// Protected by clusterIDMutex.
	isInitCertificate        bool
	onInitCertUpgrade        func()
	initCertUpgradeTriggered bool
}

// NewHandler creates a new clusterID handler
// This should be treated as a singleton unless it's called in a test
func NewHandler() *handlerImpl {
	return &handlerImpl{
		clusterIDAvailable:            concurrency.NewSignal(),
		isInitCertClusterID:           centralsensor.IsInitCertClusterID,
		getClusterID:                  centralsensor.GetClusterID,
		parseClusterIDFromServiceCert: clusterid.ParseClusterIDFromServiceCert,
	}
}

// Get returns the cluster id parsed from the service certificate
func (c *handlerImpl) Get() string {
	c.once.Do(func() {
		id := c.clusterIDFromCert()
		if c.isInitCertClusterID(id) {
			c.clusterIDMutex.Lock()
			c.isInitCertificate = true
			c.clusterIDMutex.Unlock()
			log.Infof("Certificate has wildcard subject %s. Waiting to receive cluster ID from central...", id)
			c.clusterIDAvailable.Wait()
		} else {
			concurrency.WithLock(&c.clusterIDMutex, func() {
				c.clusterID = id
				c.clusterIDAvailable.Signal()
			})
		}
	})
	return c.GetNoWait()
}

// GetNoWait returns the cluster id without waiting until it is available.
func (c *handlerImpl) GetNoWait() string {
	c.clusterIDMutex.RLock()
	defer c.clusterIDMutex.RUnlock()
	return c.clusterID
}

// RegisterInitCertUpgradeCallback registers a callback to be invoked when the cluster ID
// transitions from an init certificate to a real cluster ID.
// If the transition has already occurred by the time this is called, the callback is
// invoked immediately.
func (c *handlerImpl) RegisterInitCertUpgradeCallback(callback func()) {
	c.clusterIDMutex.Lock()
	defer c.clusterIDMutex.Unlock()

	c.onInitCertUpgrade = callback

	// If the init-cert -> real-cluster-ID transition has already completed by the
	// time the callback is registered, invoke it immediately so callers do not
	// depend on the ordering of Set() vs RegisterInitCertUpgradeCallback().
	// The transition is complete if initCertUpgradeTriggered is true.
	if callback != nil && c.initCertUpgradeTriggered {
		log.Info("Init certificate already upgraded - triggering certificate upgrade callback immediately")
		go callback()
	}
}

// Set sets the global cluster ID value.
func (c *handlerImpl) Set(value string) {
	effectiveClusterID, err := c.getClusterID(value, c.clusterIDFromCert())
	if err != nil {
		log.Panicf("Invalid dynamic cluster ID value %q: %v", value, err)
	}
	if value != "" {
		log.Infof("Received dynamic cluster ID %q", value)
	}

	c.clusterIDMutex.Lock()
	defer c.clusterIDMutex.Unlock()

	if c.clusterID == "" {
		c.clusterID = effectiveClusterID
		c.clusterIDAvailable.Signal()

		// If transitioning from init cert to real cluster ID, mark transition and trigger callback.
		if c.isInitCertificate && !c.isInitCertClusterID(effectiveClusterID) {
			c.isInitCertificate = false       // Only upgrade once.
			c.initCertUpgradeTriggered = true // Track that transition occurred.
			if c.onInitCertUpgrade != nil {
				log.Info("Init certificate detected - triggering certificate upgrade callback")
				// Capture callback to avoid holding lock during invocation.
				cb := c.onInitCertUpgrade
				go cb()
			}
		}
	} else if c.clusterID != effectiveClusterID {
		log.Panicf("Newly set cluster ID value %q conflicts with previous value %q", effectiveClusterID, c.clusterID)
	}
}

func (c *handlerImpl) clusterIDFromCert() string {
	id, err := c.parseClusterIDFromServiceCert(storage.ServiceType_SENSOR_SERVICE)
	if err != nil {
		log.Panicf("Error parsing cluster id from certificate: %v", err)
	}
	return id
}
