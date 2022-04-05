package detector

import (
	"sort"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/networkbaseline"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/detector/baseline"
	networkBaselineEval "github.com/stackrox/rox/sensor/common/detector/networkbaseline"
	"github.com/stackrox/rox/sensor/common/detector/networkpolicy"
	"github.com/stackrox/rox/sensor/common/detector/unified"
	"github.com/stackrox/rox/sensor/common/enforcer"
	"github.com/stackrox/rox/sensor/common/externalsrcs"
	"github.com/stackrox/rox/sensor/common/imagecacheutils"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/common/updater"
	"google.golang.org/grpc"
)

var (
	log                             = logging.LoggerForModule()
	_   common.CentralGRPCConnAware = (*detectorImpl)(nil)
)

// Detector is the sensor component that syncs policies from Central and runs detection
//go:generate mockgen-wrapper
type Detector interface {
	common.SensorComponent
	common.CentralGRPCConnAware

	ProcessDeployment(deployment *storage.Deployment, action central.ResourceAction)
	ReprocessDeployments(deploymentIDs ...string)
	ProcessIndicator(indicator *storage.ProcessIndicator)
	ProcessNetworkFlow(flow *storage.NetworkFlow)
}

// New returns a new detector
func New(enforcer enforcer.Enforcer, admCtrlSettingsMgr admissioncontroller.SettingsManager,
	deploymentStore store.DeploymentStore, cache expiringcache.Cache, auditLogEvents chan *sensor.AuditEvents,
	auditLogUpdater updater.Component, networkPolicyStore store.NetworkPolicyStore) Detector {
	return &detectorImpl{
		unifiedDetector: unified.NewDetector(),

		output:                    make(chan *central.MsgFromSensor),
		auditEventsChan:           auditLogEvents,
		deploymentAlertOutputChan: make(chan outputResult),
		deploymentProcessingMap:   make(map[string]int64),

		enricher:            newEnricher(cache),
		deploymentStore:     deploymentStore,
		extSrcsStore:        externalsrcs.StoreInstance(),
		baselineEval:        baseline.NewBaselineEvaluator(),
		networkbaselineEval: networkBaselineEval.NewNetworkBaselineEvaluator(),
		deduper:             newDeduper(),
		enforcer:            enforcer,

		admCtrlSettingsMgr: admCtrlSettingsMgr,
		auditLogUpdater:    auditLogUpdater,

		detectorStopper:   concurrency.NewStopper(),
		auditStopper:      concurrency.NewStopper(),
		serializerStopper: concurrency.NewStopper(),
		alertStopSig:      concurrency.NewSignal(),

		networkpolicyFinder: networkpolicy.NewFinder(networkPolicyStore),
	}
}

type detectorImpl struct {
	unifiedDetector unified.Detector

	output                    chan *central.MsgFromSensor
	auditEventsChan           chan *sensor.AuditEvents
	deploymentAlertOutputChan chan outputResult

	deploymentProcessingMap  map[string]int64
	deploymentProcessingLock sync.RWMutex

	// This lock ensures that processing is done one at a time
	// When a policy is updated, we will reflush the deployments cache back through detection
	deploymentDetectionLock sync.Mutex

	enricher            *enricher
	deploymentStore     store.DeploymentStore
	extSrcsStore        externalsrcs.Store
	baselineEval        baseline.Evaluator
	networkbaselineEval networkBaselineEval.Evaluator
	enforcer            enforcer.Enforcer
	deduper             *deduper

	admCtrlSettingsMgr admissioncontroller.SettingsManager
	auditLogUpdater    updater.Component

	detectorStopper   concurrency.Stopper
	auditStopper      concurrency.Stopper
	serializerStopper concurrency.Stopper
	alertStopSig      concurrency.Signal

	admissionCacheNeedsFlush bool

	networkpolicyFinder *networkpolicy.Finder
}

func (d *detectorImpl) Start() error {
	go d.runDetector()
	go d.runAuditLogEventDetector()
	go d.serializeDeployTimeOutput()
	return nil
}

type outputResult struct {
	results   *central.AlertResults
	timestamp int64
	action    central.ResourceAction
}

// serializeDeployTimeOutput serializes all messages that are going to be output. This allows us to guarantee the ordering
// of the messages. e.g. an alert update is not sent once the alert removal msg has been sent and alerts generated
// from an older version of a deployment
func (d *detectorImpl) serializeDeployTimeOutput() {
	defer d.serializerStopper.Stopped()
	for {
		select {
		case <-d.serializerStopper.StopDone():
			return
		case result := <-d.deploymentAlertOutputChan:
			alertResults := result.results

			switch result.action {
			case central.ResourceAction_REMOVE_RESOURCE:
				// Remove the deployment from being processed
				concurrency.WithLock(&d.deploymentProcessingLock, func() {
					delete(d.deploymentProcessingMap, alertResults.GetDeploymentId())
				})
			case central.ResourceAction_CREATE_RESOURCE:
				// Regardless if an UPDATE was processed before the create, we should try to enforce on the CREATE
				d.enforcer.ProcessAlertResults(result.action, storage.LifecycleStage_DEPLOY, alertResults)
				fallthrough
			case central.ResourceAction_UPDATE_RESOURCE:
				var isMostRecentUpdate bool
				concurrency.WithRLock(&d.deploymentProcessingLock, func() {
					value, exists := d.deploymentProcessingMap[alertResults.GetDeploymentId()]
					if !exists {
						// CREATE and UPDATE actions write a 0 timestamp into the map to signify that it is being processed
						// whereas a REMOVE deletes the deployment ID entry. Once we have processed a REMOVE, we cannot send
						// more deploytime alerts that are active as those alerts will not be cleaned up
						// instead, mark the states of all alerts as RESOLVED
						for _, alert := range alertResults.GetAlerts() {
							alert.State = storage.ViolationState_RESOLVED
						}
						isMostRecentUpdate = true
					} else {
						isMostRecentUpdate = result.timestamp >= value
						if isMostRecentUpdate {
							d.deploymentProcessingMap[alertResults.GetDeploymentId()] = result.timestamp
						}
					}

				})
				// If the deployment is not being marked as being processed, then it was already removed and don't push to the channel
				// If the timestamp of the deployment is older than one that has already been processed then also ignore
				if !isMostRecentUpdate {
					continue
				}
			}
			select {
			case <-d.serializerStopper.StopDone():
				return
			case d.output <- createAlertResultsMsg(result.action, alertResults):
			}
		}
	}
}

func (d *detectorImpl) Stop(err error) {
	d.detectorStopper.Stop()
	d.auditStopper.Stop()
	d.serializerStopper.Stop()

	// We don't need to wait for these to be stopped as they are simple select statements
	// and not used within event loops
	d.alertStopSig.Signal()
	d.enricher.stop()

	d.detectorStopper.WaitForStopped()
	d.auditStopper.WaitForStopped()
	d.serializerStopper.WaitForStopped()
}

func (d *detectorImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.SensorDetectionCap}
}

func (d *detectorImpl) processPolicySync(sync *central.PolicySync) error {
	// Note: Assume the version of the policies received from central is never
	// older than sensor's version. Convert to latest if this proves wrong.
	d.unifiedDetector.ReconcilePolicies(sync.GetPolicies())
	d.deduper.reset()

	// Take deployment lock and flush
	concurrency.WithLock(&d.deploymentDetectionLock, func() {
		for _, deployment := range d.deploymentStore.GetAll() {
			d.processDeploymentNoLock(deployment, central.ResourceAction_UPDATE_RESOURCE)
		}
	})

	if d.admCtrlSettingsMgr != nil {
		d.admCtrlSettingsMgr.UpdatePolicies(sync.GetPolicies())
	}
	return nil
}

func (d *detectorImpl) processReassessPolicies(_ *central.ReassessPolicies) error {
	// Clear the image caches and make all the deployments flow back through by clearing out the hash
	d.enricher.imageCache.RemoveAll()
	if d.admCtrlSettingsMgr != nil {
		d.admCtrlSettingsMgr.FlushCache()
	}
	d.deduper.reset()
	return nil
}

func (d *detectorImpl) processBaselineSync(sync *central.BaselineSync) error {
	for _, b := range sync.GetBaselines() {
		d.baselineEval.AddBaseline(b)
	}
	return nil
}

func (d *detectorImpl) processNetworkBaselineSync(sync *central.NetworkBaselineSync) error {
	errs := errorhelpers.NewErrorList("processing network baseline sync")
	for _, baseline := range sync.GetNetworkBaselines() {
		err := d.networkbaselineEval.AddBaseline(baseline)
		// Remember the error and continue looping
		if err != nil {
			errs.AddError(err)
		}
	}
	return errs.ToError()
}

func (d *detectorImpl) processUpdatedImage(image *storage.Image) error {
	key := imagecacheutils.GetImageCacheKey(image)

	newValue := &cacheValue{
		image: image,
	}
	d.enricher.imageCache.Add(key, newValue)
	d.admissionCacheNeedsFlush = true
	return nil
}

func (d *detectorImpl) processReprocessDeployments() error {
	if d.admissionCacheNeedsFlush && d.admCtrlSettingsMgr != nil {
		// Would prefer to do a targeted flush
		d.admCtrlSettingsMgr.FlushCache()
	}
	d.admissionCacheNeedsFlush = false
	d.deduper.reset()
	return nil
}

func (d *detectorImpl) ProcessMessage(msg *central.MsgToSensor) error {
	switch {
	case msg.GetPolicySync() != nil:
		return d.processPolicySync(msg.GetPolicySync())
	case msg.GetReassessPolicies() != nil:
		return d.processReassessPolicies(msg.GetReassessPolicies())
	case msg.GetUpdatedImage() != nil:
		return d.processUpdatedImage(msg.GetUpdatedImage())
	case msg.GetReprocessDeployments() != nil:
		return d.processReprocessDeployments()
	case msg.GetBaselineSync() != nil:
		return d.processBaselineSync(msg.GetBaselineSync())
	case msg.GetNetworkBaselineSync() != nil:
		return d.processNetworkBaselineSync(msg.GetNetworkBaselineSync())
	}
	return nil
}

func (d *detectorImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return d.output
}

func (d *detectorImpl) runDetector() {
	defer d.detectorStopper.Stopped()

	for {
		select {
		case <-d.detectorStopper.StopDone():
			return
		case scanOutput := <-d.enricher.outputChan():
			alerts := d.unifiedDetector.DetectDeployment(deploytime.DetectionContext{}, booleanpolicy.EnhancedDeployment{
				Deployment:             scanOutput.deployment,
				Images:                 scanOutput.images,
				NetworkPoliciesApplied: scanOutput.networkPoliciesApplied,
			})

			sort.Slice(alerts, func(i, j int) bool {
				return alerts[i].GetPolicy().GetId() < alerts[j].GetPolicy().GetId()
			})

			select {
			case <-d.detectorStopper.StopDone():
				return
			case <-d.serializerStopper.StopDone():
				return
			case d.deploymentAlertOutputChan <- outputResult{
				results: &central.AlertResults{
					DeploymentId: scanOutput.deployment.GetId(),
					Alerts:       alerts,
				},
				timestamp: scanOutput.deployment.GetStateTimestamp(),
				action:    scanOutput.action,
			}:
			}
		}
	}
}

func (d *detectorImpl) runAuditLogEventDetector() {
	defer d.auditStopper.Stopped()
	for {
		select {
		case <-d.auditStopper.StopDone():
			return
		case auditEvents := <-d.auditEventsChan:
			alerts := d.unifiedDetector.DetectAuditLogEvents(auditEvents)
			if len(alerts) == 0 {
				// No need to process runtime alerts that have no violations
				continue
			}

			// Force update the audit log status since alerts were detected
			// This is required because if sensor were to restart right after this alert, it's possible that
			// the saved state is prior to this the event that generated this alert (because the updater updates on a timer)
			// To avoid duplicate alerts force the state to be updated
			// This is non-blocking as the updates happen on another goroutine
			d.auditLogUpdater.ForceUpdate()

			sort.Slice(alerts, func(i, j int) bool {
				return alerts[i].GetPolicy().GetId() < alerts[j].GetPolicy().GetId()
			})
			select {
			case <-d.auditStopper.StopDone():
				return
			case <-d.serializerStopper.StopDone():
				return
			case d.output <- &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Action: central.ResourceAction_CREATE_RESOURCE,
						Resource: &central.SensorEvent_AlertResults{
							AlertResults: &central.AlertResults{
								Source: central.AlertResults_AUDIT_EVENT,
								Alerts: alerts,
								Stage:  storage.LifecycleStage_RUNTIME,
							},
						},
					},
				},
			}:
			}
		}
	}
}

func (d *detectorImpl) markDeploymentForProcessing(id string) {
	d.deploymentProcessingLock.Lock()
	defer d.deploymentProcessingLock.Unlock()
	if _, ok := d.deploymentProcessingMap[id]; !ok {
		// This marks an entry that signifies that we haven't processed the remove message yet
		d.deploymentProcessingMap[id] = 0
	}
}

func (d *detectorImpl) ProcessDeployment(deployment *storage.Deployment, action central.ResourceAction) {
	d.deploymentDetectionLock.Lock()
	defer d.deploymentDetectionLock.Unlock()

	d.processDeploymentNoLock(deployment, action)
}

func (d *detectorImpl) ReprocessDeployments(deploymentIDs ...string) {
	d.deploymentDetectionLock.Lock()
	defer d.deploymentDetectionLock.Unlock()

	for _, deploymentID := range deploymentIDs {
		d.deduper.removeDeployment(deploymentID)
	}
}

func (d *detectorImpl) processDeploymentNoLock(deployment *storage.Deployment, action central.ResourceAction) {
	switch action {
	case central.ResourceAction_REMOVE_RESOURCE:
		d.baselineEval.RemoveDeployment(deployment.GetId())
		d.deduper.removeDeployment(deployment.GetId())

		go func() {
			// Push an empty AlertResults object to the channel which will mark deploytime alerts as stale
			// This allows us to not worry about synchronizing alert msgs with deployment msgs
			select {
			case <-d.alertStopSig.Done():
				return
			case d.deploymentAlertOutputChan <- outputResult{
				results: &central.AlertResults{DeploymentId: deployment.GetId()},
				action:  action,
			}:
			}
		}()
	case central.ResourceAction_CREATE_RESOURCE:
		d.deduper.addDeployment(deployment)
		d.markDeploymentForProcessing(deployment.GetId())
		go d.enricher.blockingScan(deployment, d.networkpolicyFinder.GetNetworkPoliciesApplied(deployment), action)
	case central.ResourceAction_UPDATE_RESOURCE:
		// Check if the deployment has changes that require detection, which is more expensive than hashing
		// If not, then just return
		if !d.deduper.needsProcessing(deployment) {
			return
		}
		d.markDeploymentForProcessing(deployment.GetId())
		go d.enricher.blockingScan(deployment, d.networkpolicyFinder.GetNetworkPoliciesApplied(deployment), action)
	}
}

func (d *detectorImpl) SetCentralGRPCClient(cc grpc.ClientConnInterface) {
	d.enricher.imageSvc = v1.NewImageServiceClient(cc)
}

func (d *detectorImpl) ProcessIndicator(pi *storage.ProcessIndicator) {
	go d.processIndicator(pi)
}

func createAlertResultsMsg(action central.ResourceAction, alertResults *central.AlertResults) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     alertResults.GetDeploymentId(),
				Action: action,
				Resource: &central.SensorEvent_AlertResults{
					AlertResults: &central.AlertResults{
						DeploymentId: alertResults.GetDeploymentId(),
						Alerts:       alertResults.GetAlerts(),
						Stage:        alertResults.GetStage(),
					},
				},
			},
		},
	}
}

func (d *detectorImpl) processIndicator(pi *storage.ProcessIndicator) {
	deployment := d.deploymentStore.Get(pi.GetDeploymentId())
	if deployment == nil {
		log.Debugf("Deployment has already been removed: %+v", pi)
		// Because the indicator was already enriched with a deployment, this means the deployment is gone
		return
	}
	images := d.enricher.getImages(deployment)

	// Run detection now
	alerts := d.unifiedDetector.DetectProcess(booleanpolicy.EnhancedDeployment{
		Deployment: deployment,
		Images:     images,
	}, pi, d.baselineEval.IsOutsideLockedBaseline(pi))
	if len(alerts) == 0 {
		// No need to process runtime alerts that have no violations
		return
	}
	alertResults := &central.AlertResults{
		DeploymentId: pi.GetDeploymentId(),
		Alerts:       alerts,
		Stage:        storage.LifecycleStage_RUNTIME,
	}

	d.enforcer.ProcessAlertResults(central.ResourceAction_CREATE_RESOURCE, storage.LifecycleStage_RUNTIME, alertResults)

	select {
	case <-d.alertStopSig.Done():
		return
	case d.output <- createAlertResultsMsg(central.ResourceAction_CREATE_RESOURCE, alertResults):
	}
}

func (d *detectorImpl) ProcessNetworkFlow(flow *storage.NetworkFlow) {
	go d.processNetworkFlow(flow)
}

type networkEntityDetails struct {
	name                string
	deploymentNamespace string
	deploymentType      string
}

func (d *detectorImpl) getNetworkFlowEntityDetails(info *storage.NetworkEntityInfo) (networkEntityDetails, error) {
	switch info.GetType() {
	case storage.NetworkEntityInfo_DEPLOYMENT:
		deployment := d.deploymentStore.Get(info.GetId())
		if deployment == nil {
			// Maybe the deployment is already removed. Don't run the flow through policy anymore
			return networkEntityDetails{}, errors.Errorf("Deployment with ID: %q not found while trying to run network flow policy", info.GetId())
		}
		return networkEntityDetails{
			name:                deployment.GetName(),
			deploymentNamespace: deployment.GetNamespace(),
			deploymentType:      deployment.GetType(),
		}, nil
	case storage.NetworkEntityInfo_EXTERNAL_SOURCE:
		extsrc := d.extSrcsStore.LookupByID(info.GetId())
		if extsrc == nil {
			return networkEntityDetails{}, errors.Errorf("External source with ID: %q not found while trying to run network flow policy", info.GetId())
		}
		return networkEntityDetails{
			name: extsrc.GetExternalSource().GetName(),
		}, nil
	case storage.NetworkEntityInfo_INTERNET:
		return networkEntityDetails{
			name: networkgraph.InternetExternalSourceName,
		}, nil
	default:
		return networkEntityDetails{}, errors.Errorf("Unsupported network entity type: %q", info.GetType())
	}
}

func (d *detectorImpl) processAlertsForFlowOnEntity(
	entity *storage.NetworkEntityInfo,
	flowDetails *augmentedobjs.NetworkFlowDetails,
) {
	if entity.GetType() != storage.NetworkEntityInfo_DEPLOYMENT {
		return
	}
	deployment := d.deploymentStore.Get(entity.GetId())
	if deployment == nil {
		// Probably the deployment was deleted just before we had fetched entity names.
		log.Warnf("Stop processing alerts for network flow on deployment %q. No deployment was found", entity.GetId())
		return
	}

	images := d.enricher.getImages(deployment)
	alerts := d.unifiedDetector.DetectNetworkFlowForDeployment(booleanpolicy.EnhancedDeployment{
		Deployment: deployment,
		Images:     images,
	}, flowDetails)
	if len(alerts) == 0 {
		// No need to process runtime alerts that have no violations
		return
	}
	alertResults := &central.AlertResults{
		DeploymentId: deployment.GetId(),
		Alerts:       alerts,
		Stage:        storage.LifecycleStage_RUNTIME,
	}

	d.enforcer.ProcessAlertResults(central.ResourceAction_CREATE_RESOURCE, storage.LifecycleStage_RUNTIME, alertResults)

	select {
	case <-d.alertStopSig.Done():
		return
	case d.output <- createAlertResultsMsg(central.ResourceAction_CREATE_RESOURCE, alertResults):
	}
}

func (d *detectorImpl) processNetworkFlow(flow *storage.NetworkFlow) {
	// Only run the flows through policies if the entity types are supported
	_, srcTypeSupported := networkbaseline.ValidBaselinePeerEntityTypes[flow.GetProps().GetSrcEntity().GetType()]
	_, dstTypeSupported := networkbaseline.ValidBaselinePeerEntityTypes[flow.GetProps().GetDstEntity().GetType()]
	if !srcTypeSupported || !dstTypeSupported {
		return
	}

	// First extract more information of the flow. Mainly entity names
	srcDetails, err := d.getNetworkFlowEntityDetails(flow.GetProps().GetSrcEntity())
	if err != nil {
		log.Errorf("Error looking up source entity details while running network flow policy: %v", err)
		return
	}
	dstDetails, err := d.getNetworkFlowEntityDetails(flow.GetProps().GetDstEntity())
	if err != nil {
		log.Errorf("Error looking up destination entity details while running network flow policy: %v", err)
		return
	}
	// Check if flow is anomalous
	flowIsNotInBaseline := d.networkbaselineEval.IsOutsideLockedBaseline(flow, srcDetails.name, dstDetails.name)
	flowDetails := &augmentedobjs.NetworkFlowDetails{
		SrcEntityName:          srcDetails.name,
		SrcEntityType:          flow.GetProps().GetSrcEntity().GetType(),
		DstEntityName:          dstDetails.name,
		DstEntityType:          flow.GetProps().GetDstEntity().GetType(),
		DstPort:                flow.GetProps().GetDstPort(),
		L4Protocol:             flow.GetProps().GetL4Protocol(),
		NotInNetworkBaseline:   flowIsNotInBaseline,
		LastSeenTimestamp:      extractTimestamp(flow),
		SrcDeploymentNamespace: srcDetails.deploymentNamespace,
		SrcDeploymentType:      srcDetails.deploymentType,
		DstDeploymentNamespace: dstDetails.deploymentNamespace,
		DstDeploymentType:      dstDetails.deploymentType,
	}

	d.processAlertsForFlowOnEntity(flow.GetProps().GetSrcEntity(), flowDetails)
	d.processAlertsForFlowOnEntity(flow.GetProps().GetDstEntity(), flowDetails)
}

func extractTimestamp(flow *storage.NetworkFlow) *types.Timestamp {
	// If the flow has terminated already, then use the last seen timestamp.
	if timestamp := flow.GetLastSeenTimestamp(); timestamp != nil {
		return timestamp
	}
	// If the flow is still active, use the current timestamp.
	return types.TimestampNow()
}
