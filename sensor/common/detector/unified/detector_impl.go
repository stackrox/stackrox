package unified

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/detection/runtime"
	"github.com/stackrox/rox/pkg/kubernetes"
)

type detectorImpl struct {
	deploytimeDetector deploytime.Detector
	runtimeDetector    runtime.Detector

	// pendingPolicies holds the raw policy list from Central until first detection.
	// This defers ~6 MB of policy compilation (regexp compilation, matcher trees)
	// until a deployment/process/network event actually triggers detection.
	pendingPolicies []*storage.Policy
	policiesApplied bool
}

func (d *detectorImpl) ReconcilePolicies(newList []*storage.Policy) {
	// Store the raw policies — defer compilation until first Detect* call.
	d.pendingPolicies = newList
	d.policiesApplied = false
}

// ensurePoliciesCompiled compiles pending policies on first detection call.
func (d *detectorImpl) ensurePoliciesCompiled() {
	if d.policiesApplied {
		return
	}
	d.policiesApplied = true
	if len(d.pendingPolicies) == 0 {
		return
	}
	log.Infof("Lazy-compiling %d policies on first detection event", len(d.pendingPolicies))
	reconcilePolicySets(d.pendingPolicies, d.deploytimeDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_DEPLOY)
	})
	reconcilePolicySets(d.pendingPolicies, d.runtimeDetector.PolicySet(), func(p *storage.Policy) bool {
		return isLifecycleStage(p, storage.LifecycleStage_RUNTIME)
	})
	d.pendingPolicies = nil // release raw list
}

func (d *detectorImpl) DetectDeployment(enhancedDeployment booleanpolicy.EnhancedDeployment) []*storage.Alert {
	d.ensurePoliciesCompiled()
	alerts, err := d.deploytimeDetector.Detect(context.Background(), enhancedDeployment)
	if err != nil {
		log.Errorf("Error running detection on deployment %q: %v", enhancedDeployment.Deployment.GetName(), err)
	}
	return alerts
}

func (d *detectorImpl) DetectProcess(enhancedDeployment booleanpolicy.EnhancedDeployment, process *storage.ProcessIndicator, processNotInBaseline bool) []*storage.Alert {
	d.ensurePoliciesCompiled()
	alerts, err := d.runtimeDetector.DetectForDeploymentAndProcess(context.Background(), enhancedDeployment, process, processNotInBaseline)
	if err != nil {
		log.Errorf("Error running runtime policies for deployment %q and process %q: %v", enhancedDeployment.Deployment.GetName(), process.GetSignal().GetExecFilePath(), err)
	}
	return alerts
}

func (d *detectorImpl) DetectKubeEventForDeployment(enhancedDeployment booleanpolicy.EnhancedDeployment, kubeEvent *storage.KubernetesEvent) []*storage.Alert {
	d.ensurePoliciesCompiled()
	alerts, err := d.runtimeDetector.DetectForDeploymentAndKubeEvent(context.Background(), enhancedDeployment, kubeEvent)
	if err != nil {
		log.Errorf("Error running runtime policies for kubernetes event %s: %v", kubernetes.EventAsString(kubeEvent), err)
	}
	return alerts
}

func (d *detectorImpl) DetectNetworkFlowForDeployment(
	enhancedDeployment booleanpolicy.EnhancedDeployment,
	flow *augmentedobjs.NetworkFlowDetails,
) []*storage.Alert {
	d.ensurePoliciesCompiled()
	alerts, err := d.runtimeDetector.DetectForDeploymentAndNetworkFlow(context.Background(), enhancedDeployment, flow)
	if err != nil {
		log.Errorf("Error running runtime policies for network flow %v: %v", flow, err)
	}
	return alerts
}

func (d *detectorImpl) DetectAuditLogEvents(auditEvents *sensor.AuditEvents) []*storage.Alert {
	d.ensurePoliciesCompiled()
	alerts, err := d.runtimeDetector.DetectForAuditEvents(context.Background(), auditEvents.GetEvents())
	if err != nil {
		log.Errorf("Error evaluating runtime policies for audit events: %q", err)
		return nil
	}
	return alerts
}

func (d *detectorImpl) DetectNodeFileAccess(node *storage.Node, access *storage.FileAccess) []*storage.Alert {
	d.ensurePoliciesCompiled()
	alerts, err := d.runtimeDetector.DetectForNodeAndFileAccess(context.Background(), node, access)
	if err != nil {
		log.Errorf("Error evaluating runtime policies for node file accesses: %q", err)
		return nil
	}
	return alerts
}

func (d *detectorImpl) DetectFileAccessForDeployment(enhancedDeployment booleanpolicy.EnhancedDeployment, fileAccess *storage.FileAccess) []*storage.Alert {
	d.ensurePoliciesCompiled()
	alerts, err := d.runtimeDetector.DetectForDeploymentAndFileAccess(context.Background(), enhancedDeployment, fileAccess)
	if err != nil {
		log.Errorf("Error evaluating runtime policies for deployment file accesses: %q", err)
		return nil
	}
	return alerts
}
