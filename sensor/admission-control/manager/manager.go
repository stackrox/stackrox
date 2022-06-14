package manager

import (
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/size"
	"google.golang.org/grpc"
	admission "k8s.io/api/admission/v1"
)

// Manager manages the main business logic of the admission control service.
type Manager interface {
	Start() error
	Stop()
	Stopped() concurrency.ErrorWaitable

	SettingsUpdateC() chan<- *sensor.AdmissionControlSettings
	ResourceUpdatesC() chan<- *sensor.AdmCtrlUpdateResourceRequest

	SettingsStream() concurrency.ReadOnlyValueStream
	SensorConnStatusFlag() *concurrency.Flag
	InitialResourceSyncSig() *concurrency.Signal

	IsReady() bool

	HandleValidate(request *admission.AdmissionRequest) (*admission.AdmissionResponse, error)
	HandleK8sEvent(request *admission.AdmissionRequest) (*admission.AdmissionResponse, error)

	Alerts() <-chan []*storage.Alert
}

// New creates a new admission control manager
func New(conn *grpc.ClientConn, namespace string) Manager {
	return NewManager(namespace, 200*size.MB, sensor.NewImageServiceClient(conn), sensor.NewDeploymentServiceClient(conn))
}
