package sensor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/compliance"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/externalsrcs"
	"github.com/stackrox/rox/sensor/common/image"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/networkflow/service"
	"github.com/stackrox/rox/sensor/common/processfilter"
	"github.com/stackrox/rox/sensor/common/processsignal"
	"github.com/stackrox/rox/sensor/common/sensor"
	"github.com/stackrox/rox/sensor/common/sensor/helmconfig"
	signalService "github.com/stackrox/rox/sensor/common/signal"
	k8sadmctrl "github.com/stackrox/rox/sensor/kubernetes/admissioncontroller"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/clusterhealth"
	"github.com/stackrox/rox/sensor/kubernetes/clusterstatus"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
	"github.com/stackrox/rox/sensor/kubernetes/networkpolicies"
	"github.com/stackrox/rox/sensor/kubernetes/orchestrator"
	"github.com/stackrox/rox/sensor/kubernetes/telemetry"
	"github.com/stackrox/rox/sensor/kubernetes/upgrade"
)

var (
	log = logging.LoggerForModule()
)

// CreateSensor takes in a client interface and returns a sensor instantiation
func CreateSensor(client client.Interface, workloadHandler *fake.WorkloadManager) (*sensor.Sensor, error) {
	var admCtrlSettingsMgr admissioncontroller.SettingsManager
	if features.AdmissionControlService.Enabled() {
		admCtrlSettingsMgr = admissioncontroller.NewSettingsManager()
	}

	var helmManagedConfig *central.HelmManagedConfigInit
	if features.SensorInstallationExperience.Enabled() {
		if configFP := helmconfig.HelmConfigFingerprint.Setting(); configFP != "" {
			var err error
			helmManagedConfig, err = helmconfig.Load()
			if err != nil {
				return nil, errors.Wrap(err, "loading Helm-managed cluster config")
			}
			if helmManagedConfig.GetClusterConfig().GetConfigFingerprint() != configFP {
				return nil, errors.Errorf("fingerprint %q of loaded config does not match expected fingerprint %q, config changes can only be applied via 'helm upgrade' or a similar chart-based mechanism", helmManagedConfig.GetClusterConfig().GetConfigFingerprint(), configFP)
			}
			log.Infof("Loaded Helm-managed cluster configuration with fingerprint %q", configFP)
		}
	}
	configHandler := config.NewCommandHandler(admCtrlSettingsMgr, helmManagedConfig)
	enforcer, err := enforcer.New(client)
	if err != nil {
		return nil, errors.Wrap(err, "creating enforcer")
	}

	imageCache := expiringcache.NewExpiringCache(env.ReprocessInterval.DurationSetting())
	policyDetector := detector.New(enforcer, admCtrlSettingsMgr, imageCache)
	listener := listener.New(client, configHandler, policyDetector)

	upgradeCmdHandler, err := upgrade.NewCommandHandler(configHandler)
	if err != nil {
		return nil, errors.Wrap(err, "creating upgrade command handler")
	}

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
		clusterhealth.NewUpdater(client.Kubernetes()),
		complianceCommandHandler,
		processSignals,
		telemetry.NewCommandHandler(client.Kubernetes()),
		upgradeCmdHandler,
	}

	if features.NetworkGraphExternalSrcs.Enabled() {
		components = append(components, externalsrcs.Singleton())
	}

	if admCtrlSettingsMgr != nil {
		components = append(components, k8sadmctrl.NewConfigMapSettingsPersister(client.Kubernetes(), admCtrlSettingsMgr))
	}

	centralClient, err := centralclient.NewClient(env.CentralEndpoint.Setting())
	if err != nil {
		return nil, errors.Wrap(err, "creating central client")
	}

	s := sensor.NewSensor(
		configHandler,
		policyDetector,
		imageService,
		centralClient,
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
	return s, nil
}
