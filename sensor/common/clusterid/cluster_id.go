package clusterid

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clusterid"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	once           sync.Once
	clusterID      string
	clusterIDMutex sync.RWMutex

	clusterIDAvailable = concurrency.NewSignal()
)

func clusterIDFromCert() string {
	id, err := clusterid.ParseClusterIDFromServiceCert(storage.ServiceType_SENSOR_SERVICE)
	if err != nil {
		log.Fatalf("Error parsing cluster id from certificate: %v", err)
	}
	return id
}

// Get returns the cluster id parsed from the service certificate
func Get() string {
	once.Do(func() {
		id := clusterIDFromCert()
		if centralsensor.IsInitCertClusterID(id) {
			log.Infof("Certificate has wildcard subject %s. Waiting to receive cluster ID from central...", id)
			clusterIDAvailable.Wait()
		} else {
			clusterIDMutex.Lock()
			defer clusterIDMutex.Unlock()
			clusterID = id
			clusterIDAvailable.Signal()
		}
	})
	return GetNoWait()
}

// GetWithWait waits until we receive the cluster ID from Central or the given waitable is triggered.
func GetWithWait(waitable concurrency.Waitable) (string, error) {
	// Trigger the sync.Once function by calling Get asynchronously.
	// We cannot trust the results because a second call to Get will skip the
	// sync.Once function and call GetNoWait which will return "" if the ID is not available yet.
	go func() {
		_ = Get()
	}()
	select {
	case <-clusterIDAvailable.Done():
	case <-waitable.Done():
		return "", errors.New("context cancelled")
	}
	// At this point we know for sure we got the cluster id from central.
	return GetNoWait(), nil
}

// GetNoWait returns the cluster id without waiting until it is available.
func GetNoWait() string {
	clusterIDMutex.RLock()
	defer clusterIDMutex.RUnlock()
	return clusterID
}

// Set sets the global cluster ID value.
func Set(value string) {
	effectiveClusterID, err := centralsensor.GetClusterID(value, clusterIDFromCert())
	if err != nil {
		log.Panicf("Invalid dynamic cluster ID value %q: %v", value, err)
	}
	if value != "" {
		log.Infof("Received dynamic cluster ID %q", value)
	}

	clusterIDMutex.Lock()
	defer clusterIDMutex.Unlock()

	if clusterID == "" {
		clusterID = effectiveClusterID
		clusterIDAvailable.Signal()
	} else if clusterID != effectiveClusterID {
		log.Panicf("Newly set cluster ID value %q conflicts with previous value %q", effectiveClusterID, clusterID)
	}
}
