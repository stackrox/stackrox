package detector

import (
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/detector/unified"
	"github.com/stackrox/rox/sensor/common/detector/whitelist"
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
func New(enforcer enforcer.Enforcer, admCtrlSettingsMgr admissioncontroller.SettingsManager, cache expiringcache.Cache) Detector {
	return &detectorImpl{
		unifiedDetector: unified.NewDetector(),

		output:                    make(chan *central.MsgFromSensor),
		deploymentAlertOutputChan: make(chan outputResult),
		deploymentProcessingMap:   make(map[string]int64),

		enricher:        newEnricher(cache),
		deploymentStore: newDeploymentStore(),
		whitelistEval:   whitelist.NewWhitelistEvaluator(),
		deduper:         newDeduper(),
		enforcer:        enforcer,

		admCtrlSettingsMgr: admCtrlSettingsMgr,

		detectorStopper:   concurrency.NewStopper(),
		serializerStopper: concurrency.NewStopper(),
		alertStopSig:      concurrency.NewSignal(),
	}
}

type detectorImpl struct {
	unifiedDetector unified.Detector

	output                    chan *central.MsgFromSensor
	deploymentAlertOutputChan chan outputResult

	deploymentProcessingMap  map[string]int64
	deploymentProcessingLock sync.RWMutex

	// This lock ensures that processing is done one at a time
	// When a policy is updated, we will reflush the deployments cache back through detection
	deploymentDetectionLock sync.Mutex

	enricher        *enricher
	deploymentStore *deploymentStore
	whitelistEval   whitelist.Evaluator
	enforcer        enforcer.Enforcer
	deduper         *deduper

	admCtrlSettingsMgr admissioncontroller.SettingsManager

	detectorStopper   concurrency.Stopper
	serializerStopper concurrency.Stopper
	alertStopSig      concurrency.Signal
}

func (d *detectorImpl) Start() error {
	go d.runDetector()
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
					isMostRecentUpdate = !exists || result.timestamp >= value
					if isMostRecentUpdate {
						d.deploymentProcessingMap[alertResults.GetDeploymentId()] = result.timestamp
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
	d.serializerStopper.Stop()

	// We don't need to wait for these to be stopped as they are simple select statements
	// and not used within event loops
	d.alertStopSig.Signal()
	d.enricher.stop()

	d.detectorStopper.WaitForStopped()
	d.serializerStopper.WaitForStopped()
}

func (d *detectorImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.SensorDetectionCap}
}

func (d *detectorImpl) processPolicySync(sync *central.PolicySync) error {
	d.unifiedDetector.ReconcilePolicies(sync.GetPolicies())
	d.deduper.reset()

	// Take deployment lock and flush
	concurrency.WithLock(&d.deploymentDetectionLock, func() {
		for _, deployment := range d.deploymentStore.getAll() {
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
	d.deduper.reset()
	return nil
}

func (d *detectorImpl) processWhitelistSync(sync *central.WhitelistSync) error {
	for _, w := range sync.GetWhitelists() {
		d.whitelistEval.AddWhitelist(w)
	}
	return nil
}

func (d *detectorImpl) ProcessMessage(msg *central.MsgToSensor) error {
	switch {
	case msg.GetPolicySync() != nil:
		return d.processPolicySync(msg.GetPolicySync())
	case msg.GetReassessPolicies() != nil:
		return d.processReassessPolicies(msg.GetReassessPolicies())
	case msg.GetWhitelistSync() != nil:
		return d.processWhitelistSync(msg.GetWhitelistSync())
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
			alerts := d.unifiedDetector.DetectDeployment(deploytime.DetectionContext{}, scanOutput.deployment, scanOutput.images)

			sort.Slice(alerts, func(i, j int) bool {
				return alerts[i].GetPolicy().GetId() < alerts[j].GetPolicy().GetId()
			})

			select {
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

func (d *detectorImpl) processDeploymentNoLock(deployment *storage.Deployment, action central.ResourceAction) {
	switch action {
	case central.ResourceAction_REMOVE_RESOURCE:
		d.deploymentStore.removeDeployment(deployment.GetId())
		d.whitelistEval.RemoveDeployment(deployment.GetId())
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
		d.deploymentStore.upsertDeployment(deployment)
		d.deduper.addDeployment(deployment)
		d.markDeploymentForProcessing(deployment.GetId())
		go d.enricher.blockingScan(deployment, action)
	case central.ResourceAction_UPDATE_RESOURCE:
		d.deploymentStore.upsertDeployment(deployment)

		// Check if the deployment has changes that require detection, which is more expensive than hashing
		// If not, then just return
		if !d.deduper.needsProcessing(deployment) {
			return
		}
		d.markDeploymentForProcessing(deployment.GetId())
		go d.enricher.blockingScan(deployment, action)
	}
}

func (d *detectorImpl) SetClient(conn *grpc.ClientConn) {
	d.enricher.imageSvc = v1.NewImageServiceClient(conn)
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
	deployment := d.deploymentStore.getDeployment(pi.GetDeploymentId())
	if deployment == nil {
		log.Debugf("Deployment has already been removed: %+v", pi)
		// Because the indicator was already enriched with a deployment, this means the deployment is gone
		return
	}
	images := d.enricher.getImages(deployment)

	// Run detection now
	alerts := d.unifiedDetector.DetectProcess(deployment, images, pi, d.whitelistEval.IsOutsideLockedWhitelist(pi))

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
