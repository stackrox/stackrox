package sensor

import (
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/compliance"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/networkflow/service"
	"github.com/stackrox/rox/sensor/common/sensor"
	signalService "github.com/stackrox/rox/sensor/common/signal"
	k8sadmctrl "github.com/stackrox/rox/sensor/kubernetes/admissioncontroller"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/clusterstatus"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
	"github.com/stackrox/rox/sensor/kubernetes/networkpolicies"
	"github.com/stackrox/rox/sensor/kubernetes/orchestrator"
	"github.com/stackrox/rox/sensor/kubernetes/telemetry"
)

// CreateSensor takes in a client interface and returns a sensor instantiation
func CreateSensor(client client.Interface, extraComponents ...common.SensorComponent) *sensor.Sensor {
	var admCtrlConfigPersister admissioncontroller.ConfigPersister
	if features.AdmissionControlService.Enabled() {
		admCtrlConfigPersister = k8sadmctrl.NewConfigPersister(client.Kubernetes())
	}

	configHandler := config.NewCommandHandler(admCtrlConfigPersister)

	enforcer := enforcer.MustCreate(client.Kubernetes())
	policyDetector := detector.New(enforcer, admCtrlConfigPersister)
	listener := listener.New(client, configHandler, policyDetector)

	o := orchestrator.New(client.Kubernetes())
	complianceService := compliance.NewService(o)
	complianceCommandHandler := compliance.NewCommandHandler(complianceService)

	processSignals := signalService.New(policyDetector)

	components := []common.SensorComponent{
		listener,
		enforcer,
		manager.Singleton(),
		networkpolicies.NewCommandHandler(client.Kubernetes()),
		clusterstatus.NewUpdater(client.Kubernetes()),
		complianceCommandHandler,
		processSignals,
	}
	components = append(components, extraComponents...)

	if admCtrlConfigPersister != nil {
		components = append(components, admCtrlConfigPersister)
	}

	if features.DiagnosticBundle.Enabled() || features.Telemetry.Enabled() {
		components = append(components, telemetry.NewCommandHandler(client.Kubernetes()))
	}

	s := sensor.NewSensor(
		configHandler,
		policyDetector,
		components...,
	)

	s.AddAPIServices(
		service.Singleton(),
		processSignals,
		complianceService,
	)
	return s
}
