package clusterid

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clusterid"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	once           sync.Once
	clusterID      string
	clusterIDMutex sync.Mutex

	clusterIDAvailable = concurrency.NewSignal()
)

func clusterIDFromCert() string {
	id, err := clusterid.ParseClusterIDFromServiceCert(storage.ServiceType_SENSOR_SERVICE)
	if err != nil {
		log.Fatalf("Error parsing cluster id from certificate: %v", err)
	}
	return id
}

// Get returns the cluster id parsed from the service certficate
func Get() string {
	once.Do(func() {
		id := clusterIDFromCert()
		if features.SensorInstallationExperience.Enabled() && id == uuid.Nil.String() {
			log.Infof("Certificate has wildcard subject %s. Waiting to receive cluster ID from central...", id)
			clusterIDAvailable.Wait()
		} else {
			clusterIDMutex.Lock()
			defer clusterIDMutex.Unlock()
			clusterID = id
			clusterIDAvailable.Signal()
		}
	})
	return clusterID
}

// Set sets the global cluster ID value.
func Set(value string) {
	clusterIDMutex.Lock()
	defer clusterIDMutex.Unlock()

	if clusterID != "" && value == clusterID {
		return // cluster ID is already set and is not updated
	}

	if value == "" {
		value = clusterIDFromCert()
		if features.SensorInstallationExperience.Enabled() && value == uuid.Nil.String() {
			// Non-empty value _must_ be set if we are using a wildcard cert. Note that while an old central version
			// might not populate the ID field, in this case we should not even reach this point, as a Helm-managed
			// sensor should bail out if Central does not support Helm-managed clusters.
			log.Panic("Received an empty dynamic cluster ID and certificate does not contain a cluster ID; please upgrade Central to a more recent version")
		}
	} else {
		log.Infof("Received dynamic cluster ID %s", value)
	}

	clusterID = value
	clusterIDAvailable.Signal()
}
