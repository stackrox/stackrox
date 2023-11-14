package manager

import (
	"context"
	"sync/atomic"
	"unsafe"

	"github.com/gogo/protobuf/types"
	pkgErr "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/detection/runtime"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/admission-control/errors"
	"github.com/stackrox/rox/sensor/admission-control/resources"
	"google.golang.org/grpc"
	admission "k8s.io/api/admission/v1"
)

var (
	log = logging.LoggerForModule()

	allowAlwaysUsers = set.NewFrozenStringSet(
		"system:kube-scheduler",
		"system:kube-controller-manager",
		"system:kube-proxy",
	)

	allowAlwaysGroups = set.NewFrozenStringSet(
		"system:nodes",
	)
)

type state struct {
	*sensor.AdmissionControlSettings
	deploytimeDetector deploytime.Detector

	allRuntimePoliciesDetector                    runtime.Detector
	runtimeDetectorForPoliciesWithoutDeployFields runtime.Detector
	runtimeDetectorForPoliciesWithDeployFields    runtime.Detector

	bypassForUsers, bypassForGroups set.FrozenStringSet
	enforcedOps                     map[admission.Operation]struct{}

	centralConn *grpc.ClientConn
}

func (s *state) clusterID() string {
	clusterID := s.GetClusterId()
	if clusterID == "" {
		clusterID = getClusterID()
	}
	return clusterID
}

func (s *state) activeForOperation(op admission.Operation) bool {
	_, active := s.enforcedOps[op]
	return active
}

type manager struct {
	stopper concurrency.Stopper

	client     sensor.ImageServiceClient
	imageCache sizeboundedcache.Cache[string, imageCacheEntry]

	depClient        sensor.DeploymentServiceClient
	resourceUpdatesC chan *sensor.AdmCtrlUpdateResourceRequest
	namespaces       *resources.NamespaceStore
	deployments      *resources.DeploymentStore
	pods             *resources.PodStore
	initialSyncSig   concurrency.Signal

	settingsStream     *concurrency.ValueStream[*sensor.AdmissionControlSettings]
	settingsC          chan *sensor.AdmissionControlSettings
	lastSettingsUpdate *types.Timestamp

	syncC chan *concurrency.Signal

	statePtr unsafe.Pointer

	cacheVersion string

	sensorConnStatus concurrency.Flag

	alertsC chan []*storage.Alert

	ownNamespace string
}

// NewManager creates a new manager
func NewManager(namespace string, maxImageCacheSize int64, imageServiceClient sensor.ImageServiceClient, deploymentServiceClient sensor.DeploymentServiceClient) *manager {
	cache, err := sizeboundedcache.New(maxImageCacheSize, 2*size.MB, func(key string, value imageCacheEntry) int64 {
		return int64(len(key) + value.Size())
	})
	utils.CrashOnError(err)

	podStore := resources.NewPodStore()
	depStore := resources.NewDeploymentStore(podStore)
	nsStore := resources.NewNamespaceStore(depStore, podStore)
	return &manager{
		settingsStream: concurrency.NewValueStream[*sensor.AdmissionControlSettings](nil),
		settingsC:      make(chan *sensor.AdmissionControlSettings),
		stopper:        concurrency.NewStopper(),
		syncC:          make(chan *concurrency.Signal),

		client:     imageServiceClient,
		imageCache: cache,

		alertsC: make(chan []*storage.Alert),

		namespaces:       nsStore,
		deployments:      depStore,
		pods:             podStore,
		resourceUpdatesC: make(chan *sensor.AdmCtrlUpdateResourceRequest),
		initialSyncSig:   concurrency.NewSignal(),
		depClient:        deploymentServiceClient,

		ownNamespace: namespace,
	}
}

func (m *manager) currentState() *state {
	return (*state)(atomic.LoadPointer(&m.statePtr))
}

func (m *manager) SettingsStream() concurrency.ReadOnlyValueStream[*sensor.AdmissionControlSettings] {
	return m.settingsStream
}

func (m *manager) IsReady() bool {
	return m.currentState() != nil
}

func (m *manager) Sync(ctx context.Context) error {
	syncSig := concurrency.NewSignal()
	select {
	case m.syncC <- &syncSig:
	case <-ctx.Done():
		return ctx.Err()
	case <-m.stopper.Client().Stopped().Done():
		return m.stopper.Client().Stopped().ErrorWithDefault(pkgErr.New("manager was stopped"))
	}

	select {
	case <-syncSig.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-m.stopper.Client().Stopped().Done():
		return m.stopper.Client().Stopped().ErrorWithDefault(pkgErr.New("manager was stopped"))
	}
}

func (m *manager) Start() {
	go m.run()
}

func (m *manager) Stop() {
	m.stopper.Client().Stop()
}

func (m *manager) Stopped() concurrency.ErrorWaitable {
	return m.stopper.Client().Stopped()
}

func (m *manager) SettingsUpdateC() chan<- *sensor.AdmissionControlSettings {
	return m.settingsC
}

func (m *manager) ResourceUpdatesC() chan<- *sensor.AdmCtrlUpdateResourceRequest {
	return m.resourceUpdatesC
}

func (m *manager) InitialResourceSyncSig() *concurrency.Signal {
	return &m.initialSyncSig
}

func (m *manager) run() {
	defer m.stopper.Flow().ReportStopped()
	defer log.Info("Stopping watcher")

	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			m.ProcessNewSettings(nil)
			return
		case newSettings := <-m.settingsC:
			m.ProcessNewSettings(newSettings)
		case req := <-m.resourceUpdatesC:
			m.processUpdateResourceRequest(req)
		default:
			// Select on syncC only if there is nothing to be read from the main
			// channels. The duplication of select branches is a bit ugly, but inevitable
			// without reflection.
			select {
			case <-m.stopper.Flow().StopRequested():
				m.ProcessNewSettings(nil)
				return
			case newSettings := <-m.settingsC:
				m.ProcessNewSettings(newSettings)
			case req := <-m.resourceUpdatesC:
				m.processUpdateResourceRequest(req)
			case syncSig := <-m.syncC:
				syncSig.Signal()
			}
		}
	}
}

// ProcessNewSettings processes new settings
func (m *manager) ProcessNewSettings(newSettings *sensor.AdmissionControlSettings) {
	if newSettings == nil {
		log.Info("DISABLING admission control service (config map was deleted)")
		atomic.StorePointer(&m.statePtr, nil)
		m.lastSettingsUpdate = nil
		m.settingsStream.Push(newSettings) // typed nil ptr, not nil!
		return
	}

	if m.lastSettingsUpdate != nil && newSettings.GetTimestamp().Compare(m.lastSettingsUpdate) <= 0 {
		return // no update
	}

	allRuntimePolicySet := detection.NewPolicySet()
	runtimePoliciesWithDeployFields, runtimePoliciesWithoutDeployFields := detection.NewPolicySet(), detection.NewPolicySet()
	for _, policy := range newSettings.GetRuntimePolicies().GetPolicies() {
		if policyfields.ContainsScanRequiredFields(policy) && !newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline() {
			log.Warn(errors.ImageScanUnavailableMsg(policy))
			continue
		}

		if err := allRuntimePolicySet.UpsertPolicy(policy); err != nil {
			log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
		}

		if booleanpolicy.ContainsDeployTimeFields(policy) {
			if err := runtimePoliciesWithDeployFields.UpsertPolicy(policy); err != nil {
				log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
			}
		} else {
			if err := runtimePoliciesWithoutDeployFields.UpsertPolicy(policy); err != nil {
				log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
			}
		}
		log.Debugf("Upserted policy %q (%s)", policy.GetName(), policy.GetId())
	}

	enforceOnCreates := newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnabled()
	enforceOnUpdates := newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates()

	deployTimePolicySet := detection.NewPolicySet()
	if enforceOnCreates || enforceOnUpdates {
		for _, policy := range newSettings.GetEnforcedDeployTimePolicies().GetPolicies() {
			if policyfields.ContainsScanRequiredFields(policy) &&
				!newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline() {
				log.Warn(errors.ImageScanUnavailableMsg(policy))
				continue
			}
			if err := deployTimePolicySet.UpsertPolicy(policy); err != nil {
				log.Errorf("Unable to upsert policy %q (%s), will not be able to enforce", policy.GetName(), policy.GetId())
			}
		}
	}

	enforcedOperations := make(map[admission.Operation]struct{})
	if enforceOnCreates {
		enforcedOperations[admission.Create] = struct{}{}
	}

	if enforceOnUpdates {
		enforcedOperations[admission.Update] = struct{}{}
	}

	oldState := m.currentState()
	newState := &state{
		AdmissionControlSettings:                      newSettings,
		deploytimeDetector:                            deploytime.NewDetector(deployTimePolicySet),
		allRuntimePoliciesDetector:                    runtime.NewDetector(allRuntimePolicySet),
		runtimeDetectorForPoliciesWithDeployFields:    runtime.NewDetector(runtimePoliciesWithDeployFields),
		runtimeDetectorForPoliciesWithoutDeployFields: runtime.NewDetector(runtimePoliciesWithoutDeployFields),
		bypassForUsers:                                allowAlwaysUsers,
		bypassForGroups:                               allowAlwaysGroups,
		enforcedOps:                                   enforcedOperations,
	}

	if oldState != nil && newSettings.GetCentralEndpoint() == oldState.GetCentralEndpoint() {
		newState.centralConn = oldState.centralConn
	} else {
		if oldState != nil && oldState.centralConn != nil {
			// This *should* be non-blocking, but that's not documented, so move to a goroutine to be on the safe
			// side.
			go func() {
				if err := oldState.centralConn.Close(); err != nil {
					log.Warnf("Error closing previous connection to Central after change of central endpoint: %v", err)
				}
			}()
		}

		if newSettings.GetCentralEndpoint() != "" {
			conn, err := clientconn.AuthenticatedGRPCConnection(newSettings.GetCentralEndpoint(), mtls.CentralSubject, clientconn.UseServiceCertToken(true))
			if err != nil {
				log.Errorf("Could not create connection to Central: %v", err)
			} else {
				newState.centralConn = conn
			}
		}
	}

	if newSettings.GetCacheVersion() != m.cacheVersion {
		m.imageCache.Purge()
		m.cacheVersion = newSettings.GetCacheVersion()
	}

	//#nosec G103
	atomic.StorePointer(&m.statePtr, unsafe.Pointer(newState))
	if m.lastSettingsUpdate == nil {
		log.Info("RE-ENABLING admission control service")
	}
	m.lastSettingsUpdate = newSettings.GetTimestamp()

	enforceablePolicies := 0
	for _, policy := range allRuntimePolicySet.GetCompiledPolicies() {
		if len(policy.Policy().GetEnforcementActions()) > 0 {
			enforceablePolicies++
		}
	}
	log.Infof("Applied new admission control settings "+
		"(enforcing on %d deploy-time policies; "+
		"detecting on %d run-time policies; "+
		"enforcing on %d run-time policies)",
		len(deployTimePolicySet.GetCompiledPolicies()),
		len(allRuntimePolicySet.GetCompiledPolicies()),
		enforceablePolicies)

	m.settingsStream.Push(newSettings)
}

func (m *manager) HandleValidate(req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	state := m.currentState()

	if state == nil {
		return nil, pkgErr.New("admission controller is disabled, not handling request")
	}

	return m.evaluateAdmissionRequest(state, req)
}

func (m *manager) HandleK8sEvent(req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	state := m.currentState()
	if state == nil {
		return nil, pkgErr.New("admission controller is disabled, not handling request")
	}

	return m.evaluateRuntimeAdmissionRequest(state, req)
}

func (m *manager) SensorConnStatusFlag() *concurrency.Flag {
	return &m.sensorConnStatus
}

func (m *manager) Alerts() <-chan []*storage.Alert {
	return m.alertsC
}

func (m *manager) filterAndPutAttemptedAlertsOnChan(op admission.Operation, alerts ...*storage.Alert) {
	var filtered []*storage.Alert
	for _, alert := range alerts {
		if alert.GetDeployment() == nil {
			continue
		}

		if alert.GetEnforcement() == nil {
			continue
		}

		// Update enforcement for deploy time policy enforcements.
		if op == admission.Create {
			alert.GetDeployment().Inactive = true
			alert.Enforcement = &storage.Alert_Enforcement{
				Action:  storage.EnforcementAction_FAIL_DEPLOYMENT_CREATE_ENFORCEMENT,
				Message: "Failed deployment create in response to this policy violation.",
			}
		} else if op == admission.Update {
			alert.Enforcement = &storage.Alert_Enforcement{
				Action:  storage.EnforcementAction_FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT,
				Message: "Failed deployment update in response to this policy violation.",
			}
		}

		alert.State = storage.ViolationState_ATTEMPTED

		filtered = append(filtered, alert)
	}

	if len(filtered) > 0 {
		go m.putAlertsOnChan(filtered)
	}
}

func (m *manager) putAlertsOnChan(alerts []*storage.Alert) {
	select {
	case <-m.stopper.Flow().StopRequested():
		return
	case m.alertsC <- alerts:
	}
}

func (m *manager) processUpdateResourceRequest(req *sensor.AdmCtrlUpdateResourceRequest) {
	switch req.GetResource().(type) {
	case *sensor.AdmCtrlUpdateResourceRequest_Synced:
		m.initialSyncSig.Signal()
		log.Info("Initial resource sync with Sensor complete")
	case *sensor.AdmCtrlUpdateResourceRequest_Deployment:
		m.deployments.ProcessEvent(req.GetAction(), req.GetDeployment())
	case *sensor.AdmCtrlUpdateResourceRequest_Pod:
		m.pods.ProcessEvent(req.GetAction(), req.GetPod())
	case *sensor.AdmCtrlUpdateResourceRequest_Namespace:
		m.namespaces.ProcessEvent(req.GetAction(), req.GetNamespace())
	default:
		log.Warnf("Received message of unknown type %T from sensor, not sure what to do with it ...", m)
	}
}

func (m *manager) getDeploymentForPod(namespace, podName string) *storage.Deployment {
	return m.deployments.Get(namespace, m.pods.GetDeploymentID(namespace, podName))
}
