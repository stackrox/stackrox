package common

import (
	"os"
	"regexp"
	"strings"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// CheckNotifierInUseByCluster checks if any notifiers have been sent up for alerts.
func CheckNotifierInUseByCluster(ctx framework.ComplianceContext) {
	for _, notifier := range ctx.Data().Notifiers() {
		if notifier.GetEnabled() == true {
			framework.Pass(ctx, "At least one notifier is enabled.")
			return
		}
	}
	framework.Fail(ctx, "There are no enabled notifiers for alerts.")
}

// CheckImageScannerInUseByCluster checks if we have atleast one image scanner in use.
func CheckImageScannerInUseByCluster(ctx framework.ComplianceContext) {
	var scanners []string
	for _, integration := range ctx.Data().ImageIntegrations() {
		for _, category := range integration.GetCategories() {
			if category == storage.ImageIntegrationCategory_SCANNER {
				scanners = append(scanners, integration.Name)
			}
		}
	}

	if len(scanners) > 0 {
		framework.Pass(ctx, "Cluster has an image scanner in use")
		return
	}

	framework.Fail(ctx, "No image scanners are being used in the cluster")
}

// CheckImageScannerWasUsed checks if images were scanned atleast once.
func CheckImageScannerWasUsed(ctx framework.ComplianceContext) {
	for _, image := range ctx.Data().Images() {
		if image.GetSetCves() != nil {
			framework.Passf(ctx, "Image (%s) was scanned previously for vulnerabilities", image.GetName())
		} else {
			framework.Failf(ctx, "Image (%s) was never scanned for vulnerabilities", image.GetName())
		}
	}
}

// CheckFixedCVES checks if there are fixed CVEs in images.
// Check fails if we find fixed CVEs since we need to upgrade the image.
func CheckFixedCVES(ctx framework.ComplianceContext) {
	for _, image := range ctx.Data().Images() {
		if image.SetCves == nil {
			framework.Failf(ctx, "Image %s was never scanned for CVEs", image.GetName())
			continue
		}
		if image.GetFixableCves() > 0 {
			framework.Failf(ctx, "Image %s has %d fixed CVEs. An image upgrade is required.", image.GetName(), image.GetFixableCves())
		} else {
			framework.Passf(ctx, "Image %s has no fixed CVEs.", image.GetName())
		}
	}
}

// CheckBuildTimePolicyEnforced checks if there is atleast one build
// time policy that is enabled and enforced.
func CheckBuildTimePolicyEnforced(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		for _, stage := range p.GetLifecycleStages() {
			if stage == storage.LifecycleStage_BUILD && !p.Disabled && IsPolicyEnforced(p) {
				framework.Pass(ctx, "At least one build time policy is enabled and enforced")
				return
			}
		}
	}

	framework.Fail(ctx, "Unable to find a build time policy that is enabled and enforced")
}

// AnyPoliciesEnforced checks if any policy in the given list is being enforced.
func AnyPoliciesEnforced(ctx framework.ComplianceContext, policyNames []string) int {
	policies := ctx.Data().Policies()
	count := 0

	for _, name := range policyNames {
		p := policies[name]
		if p.GetDisabled() {
			continue
		}
		if IsPolicyEnforced(p) {
			count++
		}
	}

	return count
}

// CheckAnyPolicyInLifeCycle checks if there are any enabled policies in the given lifecycle.
func CheckAnyPolicyInLifeCycle(ctx framework.ComplianceContext, policyLifeCycle storage.LifecycleStage) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		for _, stage := range p.GetLifecycleStages() {
			if stage == policyLifeCycle && !p.Disabled {
				framework.Passf(ctx, "At least one policy in lifecycle %q is enabled", strings.ToLower(storage.LifecycleStage_name[int32(policyLifeCycle)]))
				return
			}
		}
	}
	framework.Failf(ctx, "There are no enabled policies in lifecycle %q", strings.ToLower(storage.LifecycleStage_name[int32(policyLifeCycle)]))
}

// CheckAnyPolicyInCategoryEnforced checks if there are any policies in the given category
// which are enabled and enforced.
func CheckAnyPolicyInCategoryEnforced(ctx framework.ComplianceContext, category string) {
	categoryPolicies := ctx.Data().PolicyCategories()
	policySet := categoryPolicies[category]
	if policySet.Cardinality() == 0 {
		framework.Failf(ctx, "No policies are in place to detect %q category issues", category)
		return
	}
	policies := policySet.AsSlice()
	if AnyPoliciesEnforced(ctx, policies) > 0 {
		framework.Passf(ctx, "Policies are in place to detect and enforce %q category issues.", category)
	} else {
		framework.Failf(ctx, "No policies are being enforced in %q category issues.", category)
	}
}

// CheckDeploymentHasReadOnlyFSByDeployment checks if the deployment has read-only File System.
func CheckDeploymentHasReadOnlyFSByDeployment(ctx framework.ComplianceContext) {
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		deploymentHasReadOnlyRootFS(ctx, deployment)
	})
}

func deploymentHasReadOnlyRootFS(ctx framework.ComplianceContext, deployment *storage.Deployment) {
	for _, container := range deployment.GetContainers() {
		securityContext := container.GetSecurityContext()
		readOnlyRootFS := securityContext.GetReadOnlyRootFilesystem()
		if !readOnlyRootFS {
			framework.Failf(ctx, "Deployment %s is using read-write filesystem", deployment.GetName())
		} else {
			framework.Passf(ctx, "Deployment %s is using read-only filesystem", deployment.GetName())
		}
	}
}

// IsPolicyEnabled returns true if the policy is enabled.
func IsPolicyEnabled(p *storage.Policy) bool {
	return !p.Disabled
}

// IsPolicyEnforced returns true if the policy has one or more enforcement actions.
func IsPolicyEnforced(p *storage.Policy) bool {
	for _, act := range p.GetEnforcementActions() {
		if act != storage.EnforcementAction_UNSET_ENFORCEMENT {
			return true
		}
	}
	return false
}

// CheckNetworkPoliciesByDeployment ensures that every deployment in the cluster
// has ingress and egress network policies and does not use host network namespace.
// Use this with DeploymentKind control checks.
func CheckNetworkPoliciesByDeployment(ctx framework.ComplianceContext) {
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

// ClusterHasIngressNetworkPolicies ensures the cluster has ingress network policies and does
// not use host network namespace.
func ClusterHasIngressNetworkPolicies(ctx framework.ComplianceContext) {
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
		deploymentHasIngressNetworkPolicies(ctx, deployment, deploymentIDToNodes)
	})
}

// ClusterHasEgressNetworkPolicies ensures the cluster has egress network policies and does
// not use host network namespace.
func ClusterHasEgressNetworkPolicies(ctx framework.ComplianceContext) {
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
		deploymentHasEgressNetworkPolicies(ctx, deployment, deploymentIDToNodes)
	})
}

func deploymentHasNetworkPolicies(ctx framework.ComplianceContext, deployment *storage.Deployment, deploymentIDToNodes map[string]*v1.NetworkNode) {
	if isKubeSystem(deployment) {
		framework.SkipNow(ctx, "Kubernetes system deployments are exempt from this requirement")
	}

	hasIngress := deploymentHasSpecifiedNetworkPolicy(ctx, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, deploymentIDToNodes, deployment)
	hasEgress := deploymentHasSpecifiedNetworkPolicy(ctx, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE, deploymentIDToNodes, deployment)
	usesHostNamespace := deployment.GetHostNetwork()

	if hasIngress && hasEgress && !usesHostNamespace {
		framework.Pass(ctx, "Deployment has both ingress and egress network policies applied to it, and does not use host network namespace")
		return
	}
	if !hasIngress {
		framework.Fail(ctx, "No ingress network policies apply to this deployment, hence all ingress connections are allowed")
	}
	if !hasEgress {
		framework.Fail(ctx, "No egress network policies apply to this deployment, hence all egress connections are allowed")
	}
	if usesHostNamespace {
		framework.Fail(ctx, "Deployment uses host network, which allows it to subvert network policies")
	}
}

func deploymentHasIngressNetworkPolicies(ctx framework.ComplianceContext, deployment *storage.Deployment, deploymentIDToNodes map[string]*v1.NetworkNode) {
	checkFailed := false

	if isKubeSystem(deployment) {
		framework.SkipNow(ctx, "Kubernetes system deployments are exempt from this requirement")
		return
	}

	hasIngress := deploymentHasSpecifiedNetworkPolicy(ctx, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, deploymentIDToNodes, deployment)
	usesHostNamespace := deployment.GetHostNetwork()

	if usesHostNamespace {
		framework.Fail(ctx, "Deployment uses host network, which allows it to subvert network policies")
		checkFailed = true
	}

	if !hasIngress {
		framework.Fail(ctx, "No ingress network policies apply to the deployment, hence all ingress connections are allowed")
		checkFailed = true
	}

	if !checkFailed {
		framework.Pass(ctx, "Deployment has ingress network policies applied to it, and does not use host network namespace")
	}
}

func deploymentHasEgressNetworkPolicies(ctx framework.ComplianceContext, deployment *storage.Deployment, deploymentIDToNodes map[string]*v1.NetworkNode) {
	checkFailed := false

	if isKubeSystem(deployment) {
		framework.SkipNow(ctx, "Kubernetes system deployments are exempt from this requirement")
		return
	}

	hasIngress := deploymentHasSpecifiedNetworkPolicy(ctx, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE, deploymentIDToNodes, deployment)
	usesHostNamespace := deployment.GetHostNetwork()

	if usesHostNamespace {
		framework.Fail(ctx, "Deployment uses host network, which allows it to subvert network policies")
		checkFailed = true
	}

	if !hasIngress {
		framework.Fail(ctx, "No egress network policies apply to the deployment, hence all egress connections are allowed")
		checkFailed = true
	}

	if !checkFailed {
		framework.Pass(ctx, "Deployment has egress network policies applied to it, and does not use host network namespace")
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

// CheckViolationsForPolicyByDeployment checks if the deployments have violations for a given policy.
func CheckViolationsForPolicyByDeployment(ctx framework.ComplianceContext, policy *storage.Policy) {
	alerts := ctx.Data().Alerts()
	deploymentIDToAlerts := make(map[string][]*storage.ListAlert)
	for _, alert := range alerts {
		// resolved alerts is ok. We are interested in current env.
		if alert.State == storage.ViolationState_RESOLVED {
			continue
		}

		// enforcement actions taken.
		if alert.GetEnforcementCount() > 0 {
			continue
		}
		violations := deploymentIDToAlerts[alert.GetDeployment().GetId()]
		violations = append(violations, alert)
		deploymentIDToAlerts[alert.GetDeployment().GetId()] = violations
	}

	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		checkFailed := false
		alerts := deploymentIDToAlerts[deployment.GetId()]
		for _, alert := range alerts {
			if policy.GetName() == alert.GetPolicy().GetName() {
				framework.Failf(ctx, "Deployment has active violations for %q policy not being enforced.", policy.GetName())
				checkFailed = true
			}
		}
		if !checkFailed {
			framework.Passf(ctx, "Deployment has no active violations for %q policy.", policy.GetName())
		}
	})
}

// CISBenchmarksSatisfied checks if either Docker or Kube benchmarks were run.
func CISBenchmarksSatisfied(ctx framework.ComplianceContext) {
	if ctx.Data().CISDockerTriggered() || ctx.Data().CISKubernetesTriggered() {
		framework.Pass(ctx, "CIS Benchmarks have been run.")
		return
	}

	framework.Fail(ctx, "No CIS Benchmarks have been run.")
}

// CheckSecretFilePerms determines if any container in a deployment has a secret vol that is not mounted with perm 0600
func CheckSecretFilePerms(ctx framework.ComplianceContext) {
	deployments := ctx.Data().Deployments()
	for _, deployment := range deployments {
		secretFilePath := ""
		for _, container := range deployment.Containers {
			for _, vol := range container.Volumes {
				if vol.Type == "secret" {
					secretFilePath = vol.GetDestination() + vol.Name
					info, err := os.Lstat(secretFilePath)
					if err != nil {
						log.Error(err)
						continue
					}
					perm := info.Mode().Perm()
					if perm != 0600 {
						// since this control is clusterkind, returning on first failed condition
						// not all the evidence is recorded
						framework.Failf(ctx, "Deployment has secret file in %d mode instead of 0600", perm)
						return
					}
				}
			}
		}
	}
	framework.Pass(ctx, "Deployment is not using any secret volume mounts")
}

// CheckSecretsInEnv check if any policy is configured to alert on the string secret contained in env vars
func CheckSecretsInEnv(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, policy := range policies {
		matchSecret, err := regexp.MatchString("(?i)secret", policy.GetFields().GetEnv().GetKey())
		if err != nil {
			log.Error(err)
		}
		enabled := false
		if matchSecret && err == nil &&
			policy.GetFields().GetEnv().GetValue() != "" && !policy.GetDisabled() {
			enabled = true
		}

		enforced := false
		if (matchSecret) && err == nil &&
			policy.GetFields().GetEnv().GetValue() != "" && !policy.GetDisabled() &&
			IsPolicyEnforced(policy) {
			enforced = true
		}

		if enabled && enforced {
			framework.Pass(ctx, "Detecting secrets in env is enabled and enforced")
			return
		}
		if enabled && !enforced {
			framework.Fail(ctx, "Detecting secrets in env is enabled and not enforced")
			return
		}
	}
	framework.Fail(ctx, "No policy to detect secrets in env")
}
