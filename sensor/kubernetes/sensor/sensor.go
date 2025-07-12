package sensor

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	sensorInternal "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sensor/queue"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/certdistribution"
	"github.com/stackrox/rox/sensor/common/compliance"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/delegatedregistry"
	"github.com/stackrox/rox/sensor/common/deployment"
	"github.com/stackrox/rox/sensor/common/deploymentenhancer"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/externalsrcs"
	"github.com/stackrox/rox/sensor/common/heritage"
	"github.com/stackrox/rox/sensor/common/image"
	"github.com/stackrox/rox/sensor/common/image/cache"
	"github.com/stackrox/rox/sensor/common/installmethod"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/networkflow/service"
	"github.com/stackrox/rox/sensor/common/processfilter"
	"github.com/stackrox/rox/sensor/common/processsignal"
	"github.com/stackrox/rox/sensor/common/reprocessor"
	"github.com/stackrox/rox/sensor/common/scan"
	"github.com/stackrox/rox/sensor/common/sensor"
	signalService "github.com/stackrox/rox/sensor/common/signal"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	k8sadmctrl "github.com/stackrox/rox/sensor/kubernetes/admissioncontroller"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh"
	"github.com/stackrox/rox/sensor/kubernetes/clusterhealth"
	"github.com/stackrox/rox/sensor/kubernetes/clustermetrics"
	"github.com/stackrox/rox/sensor/kubernetes/clusterstatus"
	"github.com/stackrox/rox/sensor/kubernetes/complianceoperator"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline"
	"github.com/stackrox/rox/sensor/kubernetes/helm"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"github.com/stackrox/rox/sensor/kubernetes/networkpolicies"
	"github.com/stackrox/rox/sensor/kubernetes/orchestrator"
	"github.com/stackrox/rox/sensor/kubernetes/telemetry"
	"github.com/stackrox/rox/sensor/kubernetes/upgrade"
)

var (
	log = logging.LoggerForModule()
)

// CreateSensor takes in a client interface and returns a sensor instantiation
func CreateSensor(cfg *CreateOptions) (*sensor.Sensor, error) {
	log.Info("Running sensor with Kubernetes re-sync disabled")

	hm := heritage.NewHeritageManager(pods.GetPodNamespace(), cfg.k8sClient, time.Now())
	storeProvider := resources.InitializeStore(hm)
	admCtrlSettingsMgr := admissioncontroller.NewSettingsManager(storeProvider.Deployments(), storeProvider.Pods())

	helmManagedConfig, err := helm.GetHelmManagedConfig(storage.ServiceType_SENSOR_SERVICE)
	if err != nil {
		return nil, errors.Wrap(err, "assembling Helm configuration")
	}

	log.Infof("Install method: %q", helmManagedConfig.GetManagedBy())
	installmethod.Set(helmManagedConfig.GetManagedBy())

	if cfg.introspectionK8sClient == nil {
		// This is used so we can still identify sensor's deployment with the fake-workloads.
		cfg.introspectionK8sClient = cfg.k8sClient
	}

	deploymentIdentification := FetchDeploymentIdentification(context.Background(), cfg.introspectionK8sClient.Kubernetes())
	log.Infof("Determined deployment identification: %s", protoutils.NewWrapper(deploymentIdentification))

	auditLogEventsInput := make(chan *sensorInternal.AuditEvents)
	auditLogCollectionManager := compliance.NewAuditLogCollectionManager()

	o := orchestrator.New(cfg.k8sClient.Kubernetes())
	complianceMultiplexer := compliance.NewMultiplexer()
	// TODO(ROX-16931): Turn auditLogEventsInput and auditLogCollectionManager into ComplianceComponents if possible
	complianceService := compliance.NewService(o, auditLogEventsInput, auditLogCollectionManager, complianceMultiplexer.ComplianceC())

	configHandler := config.NewCommandHandler(admCtrlSettingsMgr, deploymentIdentification, helmManagedConfig, auditLogCollectionManager)
	enforcer, err := enforcer.New(cfg.k8sClient)
	if err != nil {
		return nil, errors.Wrap(err, "creating enforcer")
	}

	imageCache := expiringcache.NewExpiringCache[cache.Key, cache.Value](env.ReprocessInterval.DurationSetting())

	localScan := scan.NewLocalScan(storeProvider.Registries(), storeProvider.RegistryMirrors())
	delegatedRegistryHandler := delegatedregistry.NewHandler(storeProvider.Registries(), localScan)

	pubSub := internalmessage.NewMessageSubscriber()

	policyDetector := detector.New(enforcer, admCtrlSettingsMgr, storeProvider.Deployments(), storeProvider.ServiceAccounts(), imageCache, auditLogEventsInput, auditLogCollectionManager, storeProvider.NetworkPolicies(), storeProvider.Registries(), localScan)
	reprocessorHandler := reprocessor.NewHandler(admCtrlSettingsMgr, policyDetector, imageCache)
	pipeline := eventpipeline.New(cfg.k8sClient, configHandler, policyDetector, reprocessorHandler, k8sNodeName.Setting(), cfg.traceWriter, storeProvider, cfg.eventPipelineQueueSize, pubSub)
	admCtrlMsgForwarder := admissioncontroller.NewAdmCtrlMsgForwarder(admCtrlSettingsMgr, pipeline)

	imageService := image.NewService(imageCache, storeProvider.Registries(), storeProvider.RegistryMirrors())
	complianceCommandHandler := compliance.NewCommandHandler(complianceService)

	// Create Process Pipeline
	indicators := make(chan *message.ExpiringMessage, queue.ScaleSizeOnNonDefault(env.ProcessIndicatorBufferSize))
	processPipeline := processsignal.NewProcessPipeline(indicators, storeProvider.Entities(), processfilter.Singleton(), policyDetector)
	var processSignals signalService.Service
	if cfg.signalServiceAuthFuncOverride != nil && cfg.localSensor {
		processSignals = signalService.New(processPipeline, indicators,
			signalService.WithAuthFuncOverride(cfg.signalServiceAuthFuncOverride),
			signalService.WithTraceWriter(cfg.processIndicatorWriter))
	} else {
		processSignals = signalService.New(processPipeline, indicators, signalService.WithTraceWriter(cfg.processIndicatorWriter))
	}
	networkFlowManager :=
		manager.NewManager(storeProvider.Entities(), externalsrcs.StoreInstance(), policyDetector, pubSub)
	enhancer := deploymentenhancer.CreateEnhancer(storeProvider)

	vmService := virtualmachine.NewService()
	components := []common.SensorComponent{
		admCtrlMsgForwarder,
		enforcer,
		networkFlowManager,
		networkpolicies.NewCommandHandler(cfg.k8sClient.Kubernetes()),
		clusterstatus.NewUpdater(cfg.k8sClient),
		clusterhealth.NewUpdater(cfg.k8sClient.Kubernetes(), 0),
		clustermetrics.New(cfg.k8sClient.Kubernetes()),
		complianceCommandHandler,
		processSignals,
		telemetry.NewCommandHandler(cfg.k8sClient.Kubernetes(), storeProvider),
		externalsrcs.Singleton(),
		admissioncontroller.AlertHandlerSingleton(),
		auditLogCollectionManager,
		reprocessorHandler,
		delegatedRegistryHandler,
		imageService,
		enhancer,
		complianceService,
		vmService,
	}
	matcher := compliance.NewNodeIDMatcher(storeProvider.Nodes())
	nodeInventoryHandler := compliance.NewNodeInventoryHandler(complianceService.NodeInventories(), complianceService.IndexReportWraps(), matcher, matcher)
	complianceMultiplexer.AddComponentWithComplianceC(nodeInventoryHandler)
	// complianceMultiplexer must start after all components that implement common.ComplianceComponent
	// i.e., after nodeInventoryHandler
	components = append(components, nodeInventoryHandler, complianceMultiplexer)

	coReadySignal := concurrency.NewSignal()
	coInfoUpdater := complianceoperator.NewInfoUpdater(cfg.k8sClient.Kubernetes(), 0, &coReadySignal)
	components = append(components, coInfoUpdater, complianceoperator.NewRequestHandler(cfg.k8sClient.Dynamic(), coInfoUpdater, &coReadySignal))

	if !cfg.localSensor {
		upgradeCmdHandler, err := upgrade.NewCommandHandler(configHandler)
		if err != nil {
			return nil, errors.Wrap(err, "creating upgrade command handler")
		}
		components = append(components, upgradeCmdHandler)
	}

	sensorNamespace := pods.GetPodNamespace()

	if admCtrlSettingsMgr != nil {
		components = append(components, k8sadmctrl.NewConfigMapSettingsPersister(cfg.k8sClient.Kubernetes(), admCtrlSettingsMgr, sensorNamespace))
	}

	if centralsensor.SecuredClusterIsNotManagedManually(helmManagedConfig) {
		podName := os.Getenv("POD_NAME")
		components = append(components,
			certrefresh.NewSecuredClusterTLSIssuer(cfg.introspectionK8sClient.Kubernetes(), sensorNamespace, podName))
	}

	s := sensor.NewSensor(
		configHandler,
		policyDetector,
		imageService,
		cfg.centralConnFactory,
		pubSub,
		cfg.certLoader,
		components...,
	)

	if cfg.workloadManager != nil {
		cfg.workloadManager.SetSignalHandlers(processPipeline, networkFlowManager)
	}

	var networkFlowService service.Service
	if cfg.networkFlowServiceAuthFuncOverride != nil && cfg.localSensor {
		networkFlowService = service.NewService(networkFlowManager,
			service.WithAuthFuncOverride(cfg.networkFlowServiceAuthFuncOverride),
			service.WithTraceWriter(cfg.networkFlowWriter))
	} else {
		networkFlowService = service.NewService(networkFlowManager, service.WithTraceWriter(cfg.networkFlowWriter))
	}
	apiServices := []grpc.APIService{
		networkFlowService,
		processSignals,
		complianceService,
		imageService,
		deployment.NewService(storeProvider.Deployments(), storeProvider.Pods()),
		vmService,
	}

	if admCtrlSettingsMgr != nil {
		apiServices = append(apiServices, admissioncontroller.NewManagementService(admCtrlSettingsMgr, admissioncontroller.AlertHandlerSingleton()))
	}

	apiServices = append(apiServices, certdistribution.NewService(cfg.k8sClient.Kubernetes(), sensorNamespace))

	s.AddAPIServices(apiServices...)
	return s, nil
}
