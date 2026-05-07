package detector

import (
	"errors"

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

// StoreProvider provides access to the stores needed by the detector.
type StoreProvider interface {
	Deployments() store.DeploymentStore
	ServiceAccounts() store.ServiceAccountStore
	NetworkPolicies() store.NetworkPolicyStore
	Nodes() store.NodeStore
	Registries() registry.Provider
	ClusterLabels() scopecomp.ClusterLabelProvider
	NamespaceLabels() scopecomp.NamespaceLabelProvider
}

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
	registryStore          registry.Provider
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

// WithClusterID sets the cluster ID provider. Required.
func (b *Builder) WithClusterID(id ClusterIDProvider) *Builder {
	b.clusterID = id
	return b
}

// WithEnforcer sets the policy enforcer. Required.
func (b *Builder) WithEnforcer(e enforcer.Enforcer) *Builder {
	b.enforcer = e
	return b
}

// WithAdmCtrlSettingsMgr sets the admission controller settings manager. Optional.
func (b *Builder) WithAdmCtrlSettingsMgr(mgr admissioncontroller.SettingsManager) *Builder {
	b.admCtrlSettingsMgr = mgr
	return b
}

// WithDeploymentStore sets the deployment store. Required.
func (b *Builder) WithDeploymentStore(ds store.DeploymentStore) *Builder {
	b.deploymentStore = ds
	return b
}

// WithServiceAccountStore sets the service account store. Required.
func (b *Builder) WithServiceAccountStore(sas store.ServiceAccountStore) *Builder {
	b.serviceAccountStore = sas
	return b
}

// WithImageCache sets the image cache. Required.
func (b *Builder) WithImageCache(c cache.Image) *Builder {
	b.imageCache = c
	return b
}

// WithAuditLogEvents sets the channel for receiving audit log events. Required.
func (b *Builder) WithAuditLogEvents(events chan *sensor.AuditEvents) *Builder {
	b.auditLogEvents = events
	return b
}

// WithAuditLogUpdater sets the audit log updater component. Required.
func (b *Builder) WithAuditLogUpdater(u updater.Component) *Builder {
	b.auditLogUpdater = u
	return b
}

// WithNetworkPolicyStore sets the network policy store. Required.
func (b *Builder) WithNetworkPolicyStore(nps store.NetworkPolicyStore) *Builder {
	b.networkPolicyStore = nps
	return b
}

// WithStoreProvider sets all store dependencies from a single provider.
// This is the preferred way to configure stores; it sets DeploymentStore,
// ServiceAccountStore, NetworkPolicyStore, NodeStore, RegistryStore,
// ClusterLabelProvider, and NamespaceLabelProvider.
func (b *Builder) WithStoreProvider(sp StoreProvider) *Builder {
	b.deploymentStore = sp.Deployments()
	b.serviceAccountStore = sp.ServiceAccounts()
	b.networkPolicyStore = sp.NetworkPolicies()
	b.nodeStore = sp.Nodes()
	b.registryStore = sp.Registries()
	b.clusterLabelProvider = sp.ClusterLabels()
	b.namespaceLabelProvider = sp.NamespaceLabels()
	return b
}

// WithRegistryStore sets the registry store directly. Required if not using WithStoreProvider.
func (b *Builder) WithRegistryStore(rs registry.Provider) *Builder {
	b.registryStore = rs
	return b
}

// WithLocalScan sets the local scan component. Required.
func (b *Builder) WithLocalScan(ls *scan.LocalScan) *Builder {
	b.localScan = ls
	return b
}

// WithNodeStore sets the node store. Required if not using WithStoreProvider.
func (b *Builder) WithNodeStore(ns store.NodeStore) *Builder {
	b.nodeStore = ns
	return b
}

// WithClusterLabelProvider sets the cluster label provider for policy scoping.
// Required if not using WithStoreProvider.
func (b *Builder) WithClusterLabelProvider(clp scopecomp.ClusterLabelProvider) *Builder {
	b.clusterLabelProvider = clp
	return b
}

// WithNamespaceLabelProvider sets the namespace label provider for policy scoping.
// Required if not using WithStoreProvider.
func (b *Builder) WithNamespaceLabelProvider(nlp scopecomp.NamespaceLabelProvider) *Builder {
	b.namespaceLabelProvider = nlp
	return b
}

// WithFactSettingsMgr sets the Fact settings manager. Optional.
func (b *Builder) WithFactSettingsMgr(fsm *filesystem.FactSettingsManager) *Builder {
	b.factSettingsMgr = fsm
	return b
}

func (b *Builder) validate() error {
	switch {
	case b.clusterID == nil:
		return errors.New("detector.Builder: ClusterID is required")
	case b.enforcer == nil:
		return errors.New("detector.Builder: Enforcer is required")
	case b.deploymentStore == nil:
		return errors.New("detector.Builder: DeploymentStore is required")
	case b.serviceAccountStore == nil:
		return errors.New("detector.Builder: ServiceAccountStore is required")
	case b.imageCache == nil:
		return errors.New("detector.Builder: ImageCache is required")
	case b.auditLogEvents == nil:
		return errors.New("detector.Builder: AuditLogEvents is required")
	case b.auditLogUpdater == nil:
		return errors.New("detector.Builder: AuditLogUpdater is required")
	case b.networkPolicyStore == nil:
		return errors.New("detector.Builder: NetworkPolicyStore is required")
	case b.registryStore == nil:
		return errors.New("detector.Builder: RegistryStore is required")
	case b.localScan == nil:
		return errors.New("detector.Builder: LocalScan is required")
	case b.nodeStore == nil:
		return errors.New("detector.Builder: NodeStore is required")
	case b.clusterLabelProvider == nil:
		return errors.New("detector.Builder: ClusterLabelProvider is required")
	case b.namespaceLabelProvider == nil:
		return errors.New("detector.Builder: NamespaceLabelProvider is required")
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
