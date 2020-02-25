package detector

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/enforcer"
	"google.golang.org/grpc"
)

var log = logging.LoggerForModule()

// Detector is the sensor component that syncs policies from Central and runs detection
type Detector interface {
	common.SensorComponent

	ProcessDeployment(deployment *storage.Deployment, action central.ResourceAction)
	ProcessIndicator(indicator *storage.ProcessIndicator)
	SetClient(conn *grpc.ClientConn)
}

// New returns a new detector
func New(enforcer enforcer.Enforcer) Detector {
	return &detectorImpl{
		enforcer: enforcer,
	}
}

type detectorImpl struct {
	enforcer enforcer.Enforcer
}

func (d *detectorImpl) Start() error {
	return nil
}

func (d *detectorImpl) Stop(err error) {}

func (d *detectorImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{
		centralsensor.SensorDetectionCap,
	}
}

func (d *detectorImpl) ProcessMessage(msg *central.MsgToSensor) error {
	switch {
	case msg.GetPolicySync() != nil:
		log.Infof("Policy Sync: %+v", msg.GetPolicySync())
	case msg.GetReassessPolicies() != nil:
		log.Infof("Reassess Policies: %+v", msg.GetReassessPolicies())
	case msg.GetWhitelistSync() != nil:
		log.Infof("Get whitelist sync: %+v", msg.GetWhitelistSync())
	}
	return nil
}

func (d *detectorImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

func (d *detectorImpl) ProcessDeployment(deployment *storage.Deployment, action central.ResourceAction) {
}

func (d *detectorImpl) SetClient(conn *grpc.ClientConn) {
}

func (d *detectorImpl) ProcessIndicator(pi *storage.ProcessIndicator) {
}
