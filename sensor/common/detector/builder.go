package detector

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/scopecomp"
	queueScaler "github.com/stackrox/rox/pkg/sensor/queue"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/detector/baseline"
	detectorMetrics "github.com/stackrox/rox/sensor/common/detector/metrics"
	networkBaselineEval "github.com/stackrox/rox/sensor/common/detector/networkbaseline"
	"github.com/stackrox/rox/sensor/common/detector/queue"
	"github.com/stackrox/rox/sensor/common/detector/unified"
	"github.com/stackrox/rox/sensor/common/enforcer"
	"github.com/stackrox/rox/sensor/common/externalsrcs"
	"github.com/stackrox/rox/sensor/common/filesystem"
	"github.com/stackrox/rox/sensor/common/image/cache"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/scan"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/common/updater"
)

// Builder constructs a Detector with a fluent API.
type Builder struct {
	clusterID              ClusterIDProvider
	enforcer               enforcer.Enforcer
	admCtrlSettingsMgr     admissioncontroller.SettingsManager
	deploymentStore        store.DeploymentStore
	serviceAccountStore    store.ServiceAccountStore
	imageCache             cache.Image
	auditLogEvents         chan *sensor.AuditEvents
	auditLogUpdater        updater.Component
	networkPolicyStore     store.NetworkPolicyStore
	registryStore          *registry.Store
	localScan              *scan.LocalScan
	nodeStore              store.NodeStore
	clusterLabelProvider   scopecomp.ClusterLabelProvider
	namespaceLabelProvider scopecomp.NamespaceLabelProvider
	factSettingsMgr        *filesystem.FactSettingsManager
}

// NewBuilder returns a new Builder for constructing a Detector.
func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) WithClusterID(id ClusterIDProvider) *Builder {
	b.clusterID = id
	return b
}

func (b *Builder) WithEnforcer(e enforcer.Enforcer) *Builder {
	b.enforcer = e
	return b
}

func (b *Builder) WithAdmCtrlSettingsMgr(mgr admissioncontroller.SettingsManager) *Builder {
	b.admCtrlSettingsMgr = mgr
	return b
}

func (b *Builder) WithDeploymentStore(ds store.DeploymentStore) *Builder {
	b.deploymentStore = ds
	return b
}

func (b *Builder) WithServiceAccountStore(sas store.ServiceAccountStore) *Builder {
	b.serviceAccountStore = sas
	return b
}

func (b *Builder) WithImageCache(c cache.Image) *Builder {
	b.imageCache = c
	return b
}

func (b *Builder) WithAuditLogEvents(events chan *sensor.AuditEvents) *Builder {
	b.auditLogEvents = events
	return b
}

func (b *Builder) WithAuditLogUpdater(u updater.Component) *Builder {
	b.auditLogUpdater = u
	return b
}

func (b *Builder) WithNetworkPolicyStore(nps store.NetworkPolicyStore) *Builder {
	b.networkPolicyStore = nps
	return b
}

func (b *Builder) WithRegistryStore(rs *registry.Store) *Builder {
	b.registryStore = rs
	return b
}

func (b *Builder) WithLocalScan(ls *scan.LocalScan) *Builder {
	b.localScan = ls
	return b
}

func (b *Builder) WithNodeStore(ns store.NodeStore) *Builder {
	b.nodeStore = ns
	return b
}

func (b *Builder) WithClusterLabelProvider(clp scopecomp.ClusterLabelProvider) *Builder {
	b.clusterLabelProvider = clp
	return b
}

func (b *Builder) WithNamespaceLabelProvider(nlp scopecomp.NamespaceLabelProvider) *Builder {
	b.namespaceLabelProvider = nlp
	return b
}

func (b *Builder) WithFactSettingsMgr(fsm *filesystem.FactSettingsManager) *Builder {
	b.factSettingsMgr = fsm
	return b
}

func (b *Builder) validate() error {
	type field struct {
		name  string
		value any
	}
	for _, f := range []field{
		{"ClusterID", b.clusterID},
		{"Enforcer", b.enforcer},
		{"DeploymentStore", b.deploymentStore},
		{"ServiceAccountStore", b.serviceAccountStore},
		{"ImageCache", b.imageCache},
		{"AuditLogEvents", b.auditLogEvents},
		{"AuditLogUpdater", b.auditLogUpdater},
		{"NetworkPolicyStore", b.networkPolicyStore},
		{"RegistryStore", b.registryStore},
		{"LocalScan", b.localScan},
		{"NodeStore", b.nodeStore},
		{"ClusterLabelProvider", b.clusterLabelProvider},
		{"NamespaceLabelProvider", b.namespaceLabelProvider},
	} {
		if f.value == nil {
			return errors.Errorf("detector.Builder: %s is required", f.name)
		}
	}
	return nil
}

// Build creates a new Detector from the builder configuration.
func (b *Builder) Build() (Detector, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	detectorStopper := concurrency.NewStopper()
	netFlowQueueSize := queueScaler.ScaleSizeOnNonDefault(env.DetectorNetworkFlowBufferSize)
	piQueueSize := queueScaler.ScaleSizeOnNonDefault(env.DetectorProcessIndicatorBufferSize)
	fileAccessQueueSize := queueScaler.ScaleSizeOnNonDefault(env.DetectorFileAccessBufferSize)
	deploymentQueueSize := 0
	if env.DetectorDeploymentBufferSize.IntegerSetting() > 0 {
		deploymentQueueSize = queueScaler.ScaleSizeOnNonDefault(env.DetectorDeploymentBufferSize)
	}
	netFlowQueue := queue.NewQueue[*queue.FlowQueueItem](
		detectorStopper,
		"FlowsQueue",
		netFlowQueueSize,
		detectorMetrics.DetectorNetworkFlowQueueOperations,
		detectorMetrics.DetectorNetworkFlowDroppedCount,
	)
	piQueue := queue.NewQueue[*queue.IndicatorQueueItem](
		detectorStopper,
		"PIsQueue",
		piQueueSize,
		detectorMetrics.DetectorProcessIndicatorQueueOperations,
		detectorMetrics.DetectorProcessIndicatorDroppedCount,
	)
	// We only need the SimpleQueue since the deploymentQueue will not be paused/resumed
	deploymentQueue := queue.NewSimpleQueue[*queue.DeploymentQueueItem](
		"DeploymentQueue",
		deploymentQueueSize,
		detectorMetrics.DetectorDeploymentQueueOperations,
		detectorMetrics.DetectorDeploymentDroppedCount,
	)

	fileAccessQueue := queue.NewQueue[*queue.FileAccessQueueItem](
		detectorStopper,
		"FileAccessQueue",
		fileAccessQueueSize,
		detectorMetrics.DetectorFileAccessQueueOperations,
		detectorMetrics.DetectorFileAccessDroppedCount,
	)

	return &detectorImpl{
		unifiedDetector: unified.NewDetector(b.clusterLabelProvider, b.namespaceLabelProvider),

		output:                    make(chan *message.ExpiringMessage),
		auditEventsChan:           b.auditLogEvents,
		deploymentAlertOutputChan: make(chan outputResult),
		deploymentProcessingMap:   make(map[string]int64),

		enricher:            newEnricher(b.clusterID, b.imageCache, b.serviceAccountStore, b.registryStore, b.localScan),
		serviceAccountStore: b.serviceAccountStore,
		deploymentStore:     b.deploymentStore,
		nodeStore:           b.nodeStore,
		extSrcsStore:        externalsrcs.StoreInstance(),
		baselineEval:        baseline.NewBaselineEvaluator(),
		networkbaselineEval: networkBaselineEval.NewNetworkBaselineEvaluator(),
		deduper:             newDeduper(),
		enforcer:            b.enforcer,

		admCtrlSettingsMgr: b.admCtrlSettingsMgr,
		auditLogUpdater:    b.auditLogUpdater,
		factSettingsMgr:    b.factSettingsMgr,

		detectorStopper:   detectorStopper,
		auditStopper:      concurrency.NewStopper(),
		serializerStopper: concurrency.NewStopper(),
		alertStopSig:      concurrency.NewSignal(),

		networkPolicyStore: b.networkPolicyStore,

		networkFlowsQueue: netFlowQueue,
		indicatorsQueue:   piQueue,
		deploymentsQueue:  deploymentQueue,
		fileAccessQueue:   fileAccessQueue,
	}, nil
}
