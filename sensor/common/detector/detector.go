package detector

import (
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/detection/runtime"
	"github.com/stackrox/rox/pkg/logging"
	options "github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
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
func New(enforcer enforcer.Enforcer, admCtrlConfigPersister admissioncontroller.ConfigPersister) Detector {
	builder := matcher.NewBuilder(
		matcher.NewRegistry(
			nil,
		),
		options.OptionsMap,
	)
	return &detectorImpl{
		deploytimeDetector:       deploytime.NewDetector(detection.NewPolicySet(detection.NewPolicyCompiler(builder))),
		runtimeDetector:          runtime.NewDetector(detection.NewPolicySet(detection.NewPolicyCompiler(builder))),
		runtimeWhitelistDetector: runtime.NewDetector(detection.NewPolicySet(detection.NewPolicyCompiler(builder))),

		output:                    make(chan *central.MsgFromSensor),
		deploymentAlertOutputChan: make(chan outputResult),
		deploymentProcessingMap:   make(map[string]int64),

		enricher:        newEnricher(),
		deploymentStore: newDeploymentStore(),
		whitelistEval:   whitelist.NewWhitelistEvaluator(),
		deduper:         newDeduper(),
		enforcer:        enforcer,

		admCtrlConfigPersister: admCtrlConfigPersister,

		detectorStopper:   concurrency.NewStopper(),
		serializerStopper: concurrency.NewStopper(),
		alertStopSig:      concurrency.NewSignal(),
	}
}

type detectorImpl struct {
	deploytimeDetector       deploytime.Detector
	runtimeDetector          runtime.Detector
	runtimeWhitelistDetector runtime.Detector

	output                    chan *central.MsgFromSensor
	deploymentAlertOutputChan chan outputResult

	deploymentProcessingMap  map[string]int64
	deploymentProcessingLock sync.RWMutex

	enricher        *enricher
	deploymentStore *deploymentStore
	whitelistEval   whitelist.Evaluator
	enforcer        enforcer.Enforcer
	deduper         *deduper

	admCtrlConfigPersister admissioncontroller.ConfigPersister

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
					isMostRecentUpdate = !exists || result.timestamp > value
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

func isLifecycleStage(policy *storage.Policy, stage storage.LifecycleStage) bool {
	for _, s := range policy.GetLifecycleStages() {
		if s == stage {
			return true
		}
	}
	return false
}

func reconcilePolicySets(sync *central.PolicySync, policySet detection.PolicySet, matcher func(p *storage.Policy) bool) {
	policyIDSet := set.NewStringSet()
	for _, v := range policySet.GetCompiledPolicies() {
		policyIDSet.Add(v.Policy().GetId())
	}

	for _, p := range sync.GetPolicies() {
		if !matcher(p) {
			continue
		}
		if err := policySet.UpsertPolicy(p); err != nil {
			log.Errorf("error upserting policy %q: %v", p.GetName(), err)
			continue
		}
		policyIDSet.Remove(p.GetId())
	}
	for removedPolicyID := range policyIDSet {
		if err := policySet.RemovePolicy(removedPolicyID); err != nil {
			log.Errorf("error removing policy %q", removedPolicyID)
		}
	}
}

func (d *detectorImpl) processPolicySync(sync *central.PolicySync) error {
	reconcilePolicySets(sync, d.deploytimeDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_DEPLOY)
	})
	reconcilePolicySets(sync, d.runtimeDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_RUNTIME) && !p.GetFields().GetWhitelistEnabled()
	})
	reconcilePolicySets(sync, d.runtimeWhitelistDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_RUNTIME) && p.GetFields().GetWhitelistEnabled()
	})
	d.deduper.reset()

	if d.admCtrlConfigPersister != nil {
		d.admCtrlConfigPersister.UpdatePolicies(sync.GetPolicies())
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
			alerts, err := d.deploytimeDetector.Detect(deploytime.DetectionContext{}, scanOutput.deployment, scanOutput.images)
			if err != nil {
				log.Errorf("error running detection on deployment %q: %v", scanOutput.deployment.GetName(), err)
			}
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
	alerts, err := d.runtimeDetector.Detect(deployment, images, pi)
	if err != nil {
		log.Errorf("error running runtime policies for deployment %q and process %q: %v", deployment.GetName(), pi.GetSignal().GetExecFilePath(), err)
	}

	// We need to handle the whitelist policies separately because there is no distinct logic in the runtime
	// detection logic and it always returns true
	if !d.whitelistEval.IsInWhitelist(pi) {
		whitelistAlerts, err := d.runtimeWhitelistDetector.Detect(deployment, images, pi)
		if err != nil {
			log.Errorf("error evaluating whitelist policies against deployment %q and process %q: %v", deployment.GetName(), pi.GetSignal().GetExecFilePath(), err)
		}
		alerts = append(alerts, whitelistAlerts...)
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
