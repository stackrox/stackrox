package manager

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/size"
	"google.golang.org/grpc"
	admission "k8s.io/api/admission/v1"
)

// Manager manages the main business logic of the admission control service.
type Manager interface {
	Start()
	Stop()
	Stopped() concurrency.ErrorWaitable

	SettingsUpdateC() chan<- *sensor.AdmissionControlSettings
	ResourceUpdatesC() chan<- *sensor.AdmCtrlUpdateResourceRequest

	SettingsStream() concurrency.ReadOnlyValueStream[*sensor.AdmissionControlSettings]
	SensorConnStatusFlag() *concurrency.Flag
	InitialResourceSyncSig() *concurrency.Signal

	IsReady() bool

	HandleValidate(request *admission.AdmissionRequest) (*admission.AdmissionResponse, error)
	HandleK8sEvent(request *admission.AdmissionRequest) (*admission.AdmissionResponse, error)

	Alerts() <-chan []*storage.Alert

	// Sync waits until the manager has processed all events (settings or resource updates) that have been
	// submitted before Sync was called, the given context expires, or the manager is stopped.
	// In the latter two cases, an error is returned.
	Sync(ctx context.Context) error
}

// New creates a new admission control manager
func New(conn *grpc.ClientConn, namespace string) Manager {
	return NewManager(namespace, 200*size.MB, sensor.NewImageServiceClient(conn), sensor.NewDeploymentServiceClient(conn))
}
