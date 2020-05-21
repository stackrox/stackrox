package unified

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/detection/runtime"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type legacyDetectorImpl struct {
	deploytimeDetector       deploytime.Detector
	runtimeDetector          runtime.Detector
	runtimeWhitelistDetector runtime.Detector
}

func (l *legacyDetectorImpl) ReconcilePolicies(newList []*storage.Policy) {
	reconcilePolicySets(newList, l.deploytimeDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_DEPLOY)
	})
	reconcilePolicySets(newList, l.runtimeDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_RUNTIME) && !p.GetFields().GetWhitelistEnabled()
	})
	reconcilePolicySets(newList, l.runtimeWhitelistDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_RUNTIME) && p.GetFields().GetWhitelistEnabled()
	})
}

func (l *legacyDetectorImpl) DetectDeployment(ctx deploytime.DetectionContext, deployment *storage.Deployment, images []*storage.Image) []*storage.Alert {
	alerts, err := l.deploytimeDetector.Detect(ctx, deployment, images)
	if err != nil {
		log.Errorf("error running detection on deployment %q: %v", deployment.GetName(), err)
	}
	return alerts

}

func (l *legacyDetectorImpl) DetectProcess(deployment *storage.Deployment, images []*storage.Image, process *storage.ProcessIndicator, processOutsideWhitelist bool) []*storage.Alert {
	alerts, err := l.runtimeDetector.Detect(deployment, images, process, processOutsideWhitelist)
	if err != nil {
		log.Errorf("error running runtime policies for deployment %q and process %q: %v", deployment.GetName(), process.GetSignal().GetExecFilePath(), err)
	}

	// We need to handle the whitelist policies separately because there is no distinct logic in the runtime
	// detection logic and it always returns true
	if processOutsideWhitelist {
		whitelistAlerts, err := l.runtimeWhitelistDetector.Detect(deployment, images, process, processOutsideWhitelist)
		if err != nil {
			log.Errorf("error evaluating whitelist policies against deployment %q and process %q: %v", deployment.GetName(), process.GetSignal().GetExecFilePath(), err)
		}
		alerts = append(alerts, whitelistAlerts...)
	}
	return alerts
}
