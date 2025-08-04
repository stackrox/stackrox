package clusterid

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clusterid"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log  = logging.LoggerForModule()
	once sync.Once
)

type Handler interface {
	Get() string
	GetNoWait() string
	Set(string)
}

type clusterIDImpl struct {
	once                          sync.Once
	clusterID                     string
	clusterIDMutex                sync.RWMutex
	clusterIDAvailable            concurrency.Signal
	isInitCertClusterID           func(string) bool
	getClusterID                  func(string, string) (string, error)
	parseClusterIDFromServiceCert func(storage.ServiceType) (string, error)
}

// NewClusterID creates a new Handler handler
func NewClusterID() Handler {
	var ret Handler
	once.Do(func() {
		ret = newClusterID()
	})
	if ret == nil {
		log.Panicf("programmer error: NewClusterID should not be called more than once outside of testing environments")
	}
	return ret
}

// NewClusterIDForTesting creates a new Handler for testing
func NewClusterIDForTesting(_ *testing.T) *clusterIDImpl {
	return newClusterID()
}

func newClusterID() *clusterIDImpl {
	return &clusterIDImpl{
		clusterIDAvailable:            concurrency.NewSignal(),
		isInitCertClusterID:           centralsensor.IsInitCertClusterID,
		getClusterID:                  centralsensor.GetClusterID,
		parseClusterIDFromServiceCert: clusterid.ParseClusterIDFromServiceCert,
	}
}

// Get returns the cluster id parsed from the service certificate
func (c *clusterIDImpl) Get() string {
	c.once.Do(func() {
		id := c.clusterIDFromCert()
		if c.isInitCertClusterID(id) {
			log.Infof("Certificate has wildcard subject %s. Waiting to receive cluster ID from central...", id)
			c.clusterIDAvailable.Wait()
		} else {
			c.clusterIDMutex.Lock()
			defer c.clusterIDMutex.Unlock()
			c.clusterID = id
			c.clusterIDAvailable.Signal()
		}
	})
	return c.GetNoWait()
}

// GetNoWait returns the cluster id without waiting until it is available.
func (c *clusterIDImpl) GetNoWait() string {
	c.clusterIDMutex.RLock()
	defer c.clusterIDMutex.RUnlock()
	return c.clusterID
}

// Set sets the global cluster ID value.
func (c *clusterIDImpl) Set(value string) {
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
	} else if c.clusterID != effectiveClusterID {
		log.Panicf("Newly set cluster ID value %q conflicts with previous value %q", effectiveClusterID, c.clusterID)
	}
}

func (c *clusterIDImpl) clusterIDFromCert() string {
	id, err := c.parseClusterIDFromServiceCert(storage.ServiceType_SENSOR_SERVICE)
	if err != nil {
		log.Panicf("Error parsing cluster id from certificate: %v", err)
	}
	return id
}
