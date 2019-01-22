package common

import (
	"strings"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// CheckImageScannerInUse checks if we have atleast one image scanner in place.
func CheckImageScannerInUse(ctx framework.ComplianceContext) {
	var scanners []string
	for _, integration := range ctx.Data().ImageIntegrations() {
		for _, category := range integration.GetCategories() {
			if category == storage.ImageIntegrationCategory_SCANNER {
				scanners = append(scanners, integration.Name)
			}
		}
	}

	if len(scanners) > 0 {
		if len(scanners) == 1 {
			framework.Passf(ctx, "An image vulnerability scanner (%s) is configured", scanners[0])
		} else {
			framework.Passf(ctx, "%d image vulnerability scanners are configured", len(scanners))
		}
	} else {
		framework.Failf(ctx, "No image vulnerability scanners are configured")
	}
}

// CheckBuildTimePolicyEnforced checks if any build time policies are being enforced.
func CheckBuildTimePolicyEnforced(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		for _, stage := range p.GetLifecycleStages() {
			if stage == storage.LifecycleStage_BUILD && !p.Disabled && len(p.EnforcementActions) != 0 {
				framework.Pass(ctx, "At least one build time policy is enabled and enforced")
				return
			}
		}
	}

	framework.Fail(ctx, "Unable to find a build time policy that is enabled and enforced")
}

// CheckPolicyInUse checks if a policy is in use.
func CheckPolicyInUse(ctx framework.ComplianceContext, name string) {
	policies := ctx.Data().Policies()
	p := policies[name]

	if p.GetDisabled() {
		framework.Failf(ctx, "'%s' policy is not in use", name)
		return
	}

	framework.Passf(ctx, "'%s' policy is in use", name)
}

// CheckPolicyEnforced checks if a policy is in use and is enforced.
func CheckPolicyEnforced(ctx framework.ComplianceContext, name string) {
	policies := ctx.Data().Policies()
	p := policies[name]

	if p.GetDisabled() {
		framework.Failf(ctx, "'%s' policy is not in use", name)
		return
	}

	if len(p.GetEnforcementActions()) == 0 {
		framework.Failf(ctx, "'%s' policy is not being enforced", name)
		return
	}

	framework.Passf(ctx, "'%s' policy is in use and is enforced", name)
}

// CheckAnyPolicyInCategory checks if there are any enabled policies in the given category.
func CheckAnyPolicyInCategory(ctx framework.ComplianceContext, category string) {
	categoryPolicies := ctx.Data().PolicyCategories()
	policySet := categoryPolicies[category]
	if policySet.Cardinality() == 0 {
		framework.Failf(ctx, "No policies are in place to monitor '%s' category issues", category)
		return
	}
	framework.Passf(ctx, "Policies are in place to monitor '%s' category issues", category)
}

// CheckAnyPolicyInCategoryEnforced checks if there are any enabled policies in the given category and are enforced.
func CheckAnyPolicyInCategoryEnforced(ctx framework.ComplianceContext, category string) {
	categoryPolicies := ctx.Data().PolicyCategories()
	policySet := categoryPolicies[category]
	if policySet.Cardinality() == 0 {
		framework.Failf(ctx, "No policies are in place to monitor '%s' category issues", category)
		return
	}
	policies := policySet.AsSlice()
	for _, policy := range policies {
		CheckPolicyEnforced(ctx, policy)
	}
}

// DeploymentHasHostMounts returns true if the deployment has host mounts.
func DeploymentHasHostMounts(deployment *storage.Deployment) bool {
	for _, container := range deployment.Containers {
		for _, vol := range container.Volumes {
			if vol.Type == "HostPath" {
				return true
			}
		}
	}
	return false
}

// IsPolicyEnabled returns true if the policy is enabled.
func IsPolicyEnabled(p *storage.Policy) Andable {
	return func() bool {
		return !p.Disabled
	}
}

// IsPolicyEnforced returns true if the policy has one or more enforcement actions.
func IsPolicyEnforced(p *storage.Policy) Andable {
	return func() bool {
		return len(p.GetEnforcementActions()) != 0
	}
}

// ClusterHasNetworkPolicies ensures the cluster has ingress/egress network policies and does
// not use host network namespace.
func ClusterHasNetworkPolicies(ctx framework.ComplianceContext) {
	// Map deployments to nodes.
	networkGraph := ctx.Data().NetworkGraph()
	deploymentIDToNodes := make(map[string]*v1.NetworkNode, len(networkGraph.GetNodes()))
	for _, node := range networkGraph.GetNodes() {
		if node.GetEntity().GetType() != storage.NetworkEntityInfo_DEPLOYMENT {
			continue
		}
		deploymentIDToNodes[node.GetEntity().GetId()] = node
	}

	// Use the deployment node map to validate each deployment.
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		deploymentHasNetworkPolicies(ctx, deployment, deploymentIDToNodes)
	})
}

func deploymentHasNetworkPolicies(ctx framework.ComplianceContext, deployment *storage.Deployment, deploymentIDToNodes map[string]*v1.NetworkNode) {
	if isKubeSystem(deployment) {
		framework.PassNow(ctx, "Kubernetes system deployments are exempt from this requirement")
		return
	}

	hasIngress := deploymentHasSpecifiedNetworkPolicy(ctx, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, deploymentIDToNodes, deployment)
	hasEgress := deploymentHasSpecifiedNetworkPolicy(ctx, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE, deploymentIDToNodes, deployment)
	usesHostNamespace := deployment.GetHostNetwork()

	if hasIngress && hasEgress && !usesHostNamespace {
		framework.Passf(ctx, "Deployment has both ingress and egress network policies applied to it, and does not use host network namespace")
		return
	}
	if !hasIngress {
		framework.Failf(ctx, "No ingress network policies apply to Deployment %s (%s), hence all ingress connections are allowed", deployment.GetName(), deployment.GetId())
	}
	if !hasEgress {
		framework.Failf(ctx, "No egress network policies apply to Deployment %s (%s), hence all egress connections are allowed", deployment.GetName(), deployment.GetId())
	}
	if usesHostNamespace {
		framework.Failf(ctx, "Deployment %s (%s) uses host network, which allows it to subvert network policies", deployment.GetName(), deployment.GetId())
	}
}

func deploymentHasSpecifiedNetworkPolicy(ctx framework.ComplianceContext, policyType storage.NetworkPolicyType, deploymentIDToNodes map[string]*v1.NetworkNode, deployment *storage.Deployment) bool {
	for _, policyID := range deploymentIDToNodes[deployment.GetId()].GetPolicyIds() {
		policy := ctx.Data().NetworkPolicies()[policyID]
		if policy == nil || !policyIsOfType(policy.GetSpec(), policyType) {
			continue
		}
		return true
	}
	return false
}

func policyIsOfType(spec *storage.NetworkPolicySpec, policyType storage.NetworkPolicyType) bool {
	for _, ty := range spec.GetPolicyTypes() {
		if ty == policyType {
			return true
		}
	}
	return false
}

func isKubeSystem(deployment *storage.Deployment) bool {
	return deployment.GetNamespace() == "kube-system"
}

// AlertsForDeployments checks if any deployments has alerts for a given policy lifecycle.
func AlertsForDeployments(ctx framework.ComplianceContext, policyLifeCycle storage.LifecycleStage) {
	alerts := ctx.Data().Alerts()
	deploymentIDToAlerts := make(map[string][]*storage.ListAlert)
	for _, alert := range alerts {
		// resolved alerts is ok. We are interested in current env.
		if alert.State == storage.ViolationState_RESOLVED {
			continue
		}
		// enforcement is enabled.
		if alert.GetEnforcementCount() > 0 {
			continue
		}
		violations := deploymentIDToAlerts[alert.GetDeployment().GetId()]
		violations = append(violations, alert)
		deploymentIDToAlerts[alert.GetDeployment().GetId()] = violations
	}

	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		count := deploymentHasAlert(deployment, policyLifeCycle, deploymentIDToAlerts)
		if count > 0 {
			framework.Failf(ctx, "Deployment has active alert(s) in '%s' lifecycle not being enforced.", strings.ToLower(storage.LifecycleStage_name[int32(policyLifeCycle)]))
			return
		}
		framework.Passf(ctx, "Deployment has no active alert(s) in '%s' lifecycle.", strings.ToLower(storage.LifecycleStage_name[int32(policyLifeCycle)]))
	})
}

func deploymentHasAlert(deployment *storage.Deployment, policyLifeCycle storage.LifecycleStage, deploymentIDToAlerts map[string][]*storage.ListAlert) int {
	alerts := deploymentIDToAlerts[deployment.GetId()]
	count := 0
	for _, alert := range alerts {
		if policyLifeCycle == alert.GetLifecycleStage() {
			count++
		}
	}
	return count
}
