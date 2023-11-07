package common

import (
	"fmt"
	"os"
	"regexp"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

// CheckNotifierInUseByCluster checks if at least one enabled policy has a notifier configured.
func CheckNotifierInUseByCluster(ctx framework.ComplianceContext) {
	notifiers := set.NewStringSet()
	for _, notifier := range ctx.Data().Notifiers() {
		notifiers.Add(notifier.Id)
	}

	for _, policy := range ctx.Data().Policies() {
		if !IsPolicyEnabled(policy) {
			continue
		}

		for _, notifierID := range policy.GetNotifiers() {
			if notifiers.Contains(notifierID) {
				framework.Pass(ctx, "At least one enabled policy has a notifier configured.")
				return
			}
		}
	}

	framework.Fail(ctx, "There are no enabled policies with a notifier configured.")
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

// AnyPolicyInLifecycleStageEnforcedInterpretation provides an interpretation sentence for CheckAnyPolicyInLifecycleStageEnforced.
func AnyPolicyInLifecycleStageEnforcedInterpretation(stage storage.LifecycleStage) string {
	return fmt.Sprintf("StackRox checks that at least one policy is enabled and enforced in the lifecycle stage %q.", stage)
}

// CheckAnyPolicyInLifecycleStageEnforced checks if there is at least one
// policy of the given lifecycle stage that is enabled and enforced.
func CheckAnyPolicyInLifecycleStageEnforced(ctx framework.ComplianceContext, lifecycleStage storage.LifecycleStage) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		if IsPolicyEnabled(p) && IsPolicyEnforced(p) && PolicyIsInLifecycleStage(p, lifecycleStage) {
			framework.Passf(ctx, "At least one policy in lifecycle stage %q is enabled and enforced", lifecycleStage)
			return
		}
	}

	framework.Failf(ctx, "No policies in lifecycle stage %q are enabled and enforced", lifecycleStage)
}

// PolicyIsInLifecycleStage returns whether the given policy is in the given lifecycle stage.
func PolicyIsInLifecycleStage(policy *storage.Policy, targetStage storage.LifecycleStage) bool {
	for _, policyStage := range policy.GetLifecycleStages() {
		if policyStage == targetStage {
			return true
		}
	}

	return false
}

// AnyPoliciesEnforced checks if any policy in the given set is being enforced.
func AnyPoliciesEnforced(ctx framework.ComplianceContext, policyNames set.StringSet) int {
	policies := ctx.Data().Policies()
	count := 0

	for name := range policyNames {
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

// AnyPolicyInLifeCycleInterpretation provides an interpretation sentence for CheckAnyPolicyInLifeCycle.
func AnyPolicyInLifeCycleInterpretation(stage storage.LifecycleStage) string {
	return fmt.Sprintf("StackRox checks that at least one policy is enabled in the lifecycle stage %q.", stage)
}

// CheckAnyPolicyInLifeCycle checks if there are any enabled policies in the given lifecycle.
func CheckAnyPolicyInLifeCycle(ctx framework.ComplianceContext, policyLifeCycle storage.LifecycleStage) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		if IsPolicyEnabled(p) && PolicyIsInLifecycleStage(p, policyLifeCycle) {
			framework.Passf(ctx, "At least one policy in lifecycle stage %q is enabled", policyLifeCycle)
			return
		}
	}
	framework.Failf(ctx, "There are no enabled policies in lifecycle stage %q", policyLifeCycle)
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

	if AnyPoliciesEnforced(ctx, policySet) > 0 {
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
			framework.Failf(ctx, "Deployment %s (container %s) is using read-write filesystem", deployment.GetName(), container.GetName())
		} else {
			framework.Passf(ctx, "Deployment %s (container %s) is using read-only filesystem", deployment.GetName(), container.GetName())
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

// CheckNetworkPoliciesByDeploymentInterpretation is the interpretation text for CheckNetworkPoliciesByDeployment.
const CheckNetworkPoliciesByDeploymentInterpretation = `StackRox analyzes the Kubernetes network policies configured in your clusters.
Network policies allow you to block unexpected connections to, from, or between deployments.
This control checks that all deployments have ingress and egress network policies.`

// CheckNetworkPoliciesByDeployment ensures that every deployment in the cluster
// has ingress and egress network policies and does not use host network namespace.
// Use this with DeploymentKind control checks.
func CheckNetworkPoliciesByDeployment(ctx framework.ComplianceContext) {
	// Map deployments to nodes.
	networkGraph := ctx.Data().DeploymentsToNetworkPolicies()

	// Use the deployment node map to validate each deployment.
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		deploymentHasNetworkPolicies(ctx, deployment, networkGraph)
	})
}

// ClusterHasIngressNetworkPolicies ensures the cluster has ingress network policies and does
// not use host network namespace.
func ClusterHasIngressNetworkPolicies(ctx framework.ComplianceContext) {
	networkGraph := ctx.Data().DeploymentsToNetworkPolicies()
	// Use the deployment node map to validate each deployment.
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		deploymentHasIngressNetworkPolicies(ctx, deployment, networkGraph)
	})
}

// ClusterHasEgressNetworkPolicies ensures the cluster has egress network policies and does
// not use host network namespace.
func ClusterHasEgressNetworkPolicies(ctx framework.ComplianceContext) {
	networkGraph := ctx.Data().DeploymentsToNetworkPolicies()

	// Use the deployment node map to validate each deployment.
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		deploymentHasEgressNetworkPolicies(ctx, deployment, networkGraph)
	})
}

func deploymentHasNetworkPolicies(ctx framework.ComplianceContext, deployment *storage.Deployment, deploymentIDToNetworkPolicies map[string][]*storage.NetworkPolicy) {
	if isKubeSystem(deployment) {
		framework.SkipNow(ctx, "Kubernetes system deployments are exempt from this requirement")
	}

	hasIngress := deploymentHasSpecifiedNetworkPolicy(ctx, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, deploymentIDToNetworkPolicies, deployment)
	hasEgress := deploymentHasSpecifiedNetworkPolicy(ctx, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE, deploymentIDToNetworkPolicies, deployment)
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

func deploymentHasIngressNetworkPolicies(ctx framework.ComplianceContext, deployment *storage.Deployment, deploymentIDToNetworkPolicies map[string][]*storage.NetworkPolicy) {
	checkFailed := false

	if isKubeSystem(deployment) {
		framework.SkipNow(ctx, "Kubernetes system deployments are exempt from this requirement")
		return
	}

	hasIngress := deploymentHasSpecifiedNetworkPolicy(ctx, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE, deploymentIDToNetworkPolicies, deployment)
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

func deploymentHasEgressNetworkPolicies(ctx framework.ComplianceContext, deployment *storage.Deployment, deploymentIDToNodes map[string][]*storage.NetworkPolicy) {
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

func deploymentHasSpecifiedNetworkPolicy(ctx framework.ComplianceContext, policyType storage.NetworkPolicyType, deploymentsToNetworkPolicies map[string][]*storage.NetworkPolicy, deployment *storage.Deployment) bool {
	netPols, ok := deploymentsToNetworkPolicies[deployment.GetId()]
	if !ok {
		return false
	}

	for _, netPol := range netPols {
		if !policyIsOfType(netPol.GetSpec(), policyType) {
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

// CheckNoViolationsForDeployPhasePoliciesInterpretation is the interpretation text for CheckNoViolationsForDeployPhasePolicies.
const CheckNoViolationsForDeployPhasePoliciesInterpretation = `StackRox checks that there are no unresolved violations for policies in the lifecycle stage "DEPLOY".`

// CheckNoViolationsForDeployPhasePolicies checks that there are no active violations for deploy-phase policies.
func CheckNoViolationsForDeployPhasePolicies(ctx framework.ComplianceContext) {
	var violated bool
	alerts := ctx.Data().UnresolvedAlerts()
	for _, alert := range alerts {
		if alert.GetLifecycleStage() == storage.LifecycleStage_DEPLOY {
			framework.Failf(ctx, "Policy %q is violated by deployment %q in namespace %q of cluster %q",
				alert.GetPolicy().GetName(), alert.GetDeployment().GetName(), alert.GetDeployment().GetNamespace(), alert.GetDeployment().GetClusterName())
			violated = true
		}
	}
	if !violated {
		framework.Pass(ctx, "There are no active violations for deploy-phase policies")
	}
}

// CheckViolationsForPolicyByDeployment checks if the deployments have violations for a given policy.
func CheckViolationsForPolicyByDeployment(ctx framework.ComplianceContext, policy *storage.Policy) {
	alerts := ctx.Data().UnresolvedAlerts()
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
	if ctx.Data().CISKubernetesTriggered() {
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

var (
	secretRegexp = regexp.MustCompile("(?i)secret")
)

// CheckSecretsInEnv check if any policy is configured to alert on the string secret contained in env vars
func CheckSecretsInEnv(ctx framework.ComplianceContext) {
	var atLeastOnePassed bool
	var policiesEnabledNotEnforced []string
	for _, policy := range ctx.Data().Policies() {
		envKeyValues := policyfields.GetEnvKeyValues(policy)
		if policy.GetDisabled() {
			continue
		}
		enforced := IsPolicyEnforced(policy)
		var matchingPairFoundInPolicy bool
		for _, kvPair := range envKeyValues {
			if secretRegexp.MatchString(kvPair.Key) {
				matchingPairFoundInPolicy = true
				break
			}
		}
		if !matchingPairFoundInPolicy {
			continue
		}
		if enforced {
			atLeastOnePassed = true
		} else {
			policiesEnabledNotEnforced = append(policiesEnabledNotEnforced, policy.GetName())
		}
	}
	if atLeastOnePassed {
		framework.Pass(ctx, "At least one policy is enabled and enforced that detects secrets in environment variables")
	} else if len(policiesEnabledNotEnforced) > 0 {
		framework.Failf(ctx, "Enforcement is not set on at least one policy that detects secrets in environment variables (%v)", policiesEnabledNotEnforced)
	} else {
		framework.Fail(ctx, "No policy to detect secrets in environment variables")
	}
}

// CheckRuntimeSupportInCluster checks if runtime is enabled and collector
// is sending process and network data.
func CheckRuntimeSupportInCluster(ctx framework.ComplianceContext) {
	if ctx.Data().Cluster().GetCollectionMethod() != storage.CollectionMethod_NO_COLLECTION && ctx.Data().HasProcessIndicators() && len(ctx.Data().NetworkFlows()) > 0 {
		framework.PassNowf(ctx, "Runtime support is enabled (or collector service is running) for cluster %s. Network visualization for active network connections is possible.", ctx.Data().Cluster().GetName())
	}
	framework.Failf(ctx, "Runtime support is not enabled (or collector service is not running) for cluster %s. Network visualization for active network connections is not possible.", ctx.Data().Cluster().GetName())
}
