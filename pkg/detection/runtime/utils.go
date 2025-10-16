package runtime

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
)

// constructProcessAlert constructs an alert.
func constructProcessAlert(policy *storage.Policy, deployment *storage.Deployment, violations booleanpolicy.Violations) *storage.Alert {
	if len(violations.AlertViolations) == 0 && violations.ProcessViolation == nil {
		return nil
	}
	alert := constructGenericRuntimeAlert(policy, deployment, violations.AlertViolations)
	alert.SetProcessViolation(violations.ProcessViolation)
	if action, msg := buildEnforcement(policy); action != storage.EnforcementAction_UNSET_ENFORCEMENT {
		ae := &storage.Alert_Enforcement{}
		ae.SetAction(action)
		ae.SetMessage(msg)
		alert.SetEnforcement(ae)
	}
	return alert
}

func constructKubeEventAlert(
	policy *storage.Policy,
	kubeEvent *storage.KubernetesEvent,
	kubeResource interface{},
	violations booleanpolicy.Violations,
) *storage.Alert {
	if len(violations.AlertViolations) == 0 {
		return nil
	}

	// NOTE: Most Kube Event alerts will have a Resource entity instead of a Deployment. However, there are a few exceptions
	// such as pod exec/port forward policies that have deployment. To differentiate we will be using the policy event source
	// Currently all audit log events have Resource
	if policy.GetEventSource() == storage.EventSource_AUDIT_LOG_EVENT {
		return constructResourceRuntimeAlert(policy, kubeEvent, violations.AlertViolations)
		// Audit Log event source policies cannot have enforcement (for now)
	}

	alert := constructGenericRuntimeAlert(policy, kubeResource.(*storage.Deployment), violations.AlertViolations)
	if action, msg := buildKubeEventEnforcement(policy); action != storage.EnforcementAction_UNSET_ENFORCEMENT {
		ae := &storage.Alert_Enforcement{}
		ae.SetAction(action)
		ae.SetMessage(msg)
		alert.SetEnforcement(ae)
	}
	return alert
}

func constructNetworkFlowAlert(
	policy *storage.Policy,
	deployment *storage.Deployment,
	_ *augmentedobjs.NetworkFlowDetails,
	violations booleanpolicy.Violations,
) *storage.Alert {
	if len(violations.AlertViolations) == 0 {
		return nil
	}
	alert := constructGenericRuntimeAlert(policy, deployment, violations.AlertViolations)
	// TODO: there is no network flow policy enforcement for now
	return alert
}

func constructGenericRuntimeAlert(
	policy *storage.Policy,
	deployment *storage.Deployment,
	violations []*storage.Alert_Violation,
) *storage.Alert {
	alert := &storage.Alert{}
	alert.SetId(uuid.NewV4().String())
	alert.SetPolicy(policy.CloneVT())
	alert.SetLifecycleStage(storage.LifecycleStage_RUNTIME)
	alert.SetDeployment(proto.ValueOrDefault(convert.ToAlertDeployment(deployment).Deployment))
	alert.SetViolations(violations)
	alert.SetTime(protocompat.TimestampNow())
	return alert
}

func constructResourceRuntimeAlert(
	policy *storage.Policy,
	kubeEvent *storage.KubernetesEvent,
	violations []*storage.Alert_Violation,
) *storage.Alert {
	alert := &storage.Alert{}
	alert.SetId(uuid.NewV4().String())
	alert.SetPolicy(policy.CloneVT())
	alert.SetLifecycleStage(storage.LifecycleStage_RUNTIME)
	alert.SetResource(proto.ValueOrDefault(convert.ToAlertResource(kubeEvent).Resource))
	alert.SetViolations(violations)
	alert.SetTime(protocompat.TimestampNow())
	return alert
}

func buildEnforcement(policy *storage.Policy) (enforcement storage.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == storage.EnforcementAction_KILL_POD_ENFORCEMENT {
			return storage.EnforcementAction_KILL_POD_ENFORCEMENT,
				"StackRox killed pods in deployment in response to this policy violation."
		}
	}
	return storage.EnforcementAction_UNSET_ENFORCEMENT, ""
}

func buildKubeEventEnforcement(policy *storage.Policy) (enforcement storage.EnforcementAction, message string) {
	for _, enforcementAction := range policy.GetEnforcementActions() {
		if enforcementAction == storage.EnforcementAction_FAIL_KUBE_REQUEST_ENFORCEMENT {
			return storage.EnforcementAction_FAIL_KUBE_REQUEST_ENFORCEMENT,
				"StackRox failed Kubernetes request in response to this policy violation."
		}
	}
	return storage.EnforcementAction_UNSET_ENFORCEMENT, ""
}
