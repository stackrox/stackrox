package unified

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/detection/runtime"
	"github.com/stackrox/rox/pkg/kubernetes"
)

type detectorImpl struct {
	deploytimeDetector deploytime.Detector
	runtimeDetector    runtime.Detector
}

func (d *detectorImpl) ReconcilePolicies(newList []*storage.Policy) {
	reconcilePolicySets(newList, d.deploytimeDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_DEPLOY)
	})
	reconcilePolicySets(newList, d.runtimeDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_RUNTIME)
	})
}

func (d *detectorImpl) DetectDeployment(ctx deploytime.DetectionContext, deployment *storage.Deployment, images []*storage.Image) []*storage.Alert {
	alerts, err := d.deploytimeDetector.Detect(ctx, deployment, images)
	if err != nil {
		log.Errorf("error running detection on deployment %q: %v", deployment.GetName(), err)
	}
	return alerts

}

func (d *detectorImpl) DetectProcess(deployment *storage.Deployment, images []*storage.Image, process *storage.ProcessIndicator, processNotInBaseline bool) []*storage.Alert {
	alerts, err := d.runtimeDetector.DetectForDeployment(deployment, images, process, processNotInBaseline, nil)
	if err != nil {
		log.Errorf("error running runtime policies for deployment %q and process %q: %v", deployment.GetName(), process.GetSignal().GetExecFilePath(), err)
	}
	return alerts
}

func (d *detectorImpl) DetectKubeEventForDeployment(deployment *storage.Deployment, images []*storage.Image, kubeEvent *storage.KubernetesEvent) []*storage.Alert {
	alerts, err := d.runtimeDetector.DetectForDeployment(deployment, images, nil, false, kubeEvent)
	if err != nil {
		log.Errorf("error running runtime policies for kubernetes event %s: %v", kubernetes.EventAsString(kubeEvent), err)
	}
	return alerts
}
