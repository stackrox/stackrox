package sensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/compliance"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/image"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/networkflow/service"
	"github.com/stackrox/rox/sensor/common/processfilter"
	"github.com/stackrox/rox/sensor/common/processsignal"
	"github.com/stackrox/rox/sensor/common/sensor"
	signalService "github.com/stackrox/rox/sensor/common/signal"
	k8sadmctrl "github.com/stackrox/rox/sensor/kubernetes/admissioncontroller"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/clusterstatus"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
	"github.com/stackrox/rox/sensor/kubernetes/networkpolicies"
	"github.com/stackrox/rox/sensor/kubernetes/orchestrator"
	"github.com/stackrox/rox/sensor/kubernetes/telemetry"
)

// CreateSensor takes in a client interface and returns a sensor instantiation
func CreateSensor(client client.Interface, workloadHandler *fake.WorkloadManager, extraComponents ...common.SensorComponent) *sensor.Sensor {
	var admCtrlSettingsMgr admissioncontroller.SettingsManager
	if features.AdmissionControlService.Enabled() {
		admCtrlSettingsMgr = admissioncontroller.NewSettingsManager()
	}

	configHandler := config.NewCommandHandler(admCtrlSettingsMgr)

	enforcer := enforcer.MustCreate(client.Kubernetes())

	imageCache := expiringcache.NewExpiringCache(env.ReprocessInterval.DurationSetting())
	policyDetector := detector.New(enforcer, admCtrlSettingsMgr, imageCache)
	listener := listener.New(client, configHandler, policyDetector)

	o := orchestrator.New(client.Kubernetes())
	complianceService := compliance.NewService(o)
	imageService := image.NewService(imageCache)
	complianceCommandHandler := compliance.NewCommandHandler(complianceService)

	// Create Process Pipeline
	indicators := make(chan *central.MsgFromSensor)
	processPipeline := processsignal.NewProcessPipeline(indicators, clusterentities.StoreInstance(), processfilter.Singleton(), policyDetector)
	processSignals := signalService.New(processPipeline, indicators)
	components := []common.SensorComponent{
		listener,
		enforcer,
		manager.Singleton(),
		networkpolicies.NewCommandHandler(client.Kubernetes()),
		clusterstatus.NewUpdater(client.Kubernetes()),
		complianceCommandHandler,
		processSignals,
		telemetry.NewCommandHandler(client.Kubernetes()),
	}
	components = append(components, extraComponents...)

	if admCtrlSettingsMgr != nil {
		components = append(components, k8sadmctrl.NewConfigMapSettingsPersister(client.Kubernetes(), admCtrlSettingsMgr))
	}

	s := sensor.NewSensor(
		configHandler,
		policyDetector,
		imageService,
		components...,
	)

	if workloadHandler != nil {
		workloadHandler.SetSignalHandlers(processPipeline, manager.Singleton())
	}

	apiServices := []grpc.APIService{
		service.Singleton(),
		processSignals,
		complianceService,
		imageService,
	}

	if admCtrlSettingsMgr != nil {
		apiServices = append(apiServices, admissioncontroller.NewManagementService(admCtrlSettingsMgr))
	}

	s.AddAPIServices(apiServices...)
	return s
}
