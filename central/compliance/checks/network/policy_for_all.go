package network

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func init() {
	framework.MustRegisterNewCheck(
		"all-deployments-have-ingress-policy",
		framework.DeploymentKind,
		nil,
		func(ctx framework.ComplianceContext) {
			checkAllDeploymentsHavePolicy(ctx, storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE)
		})

	framework.MustRegisterNewCheck(
		"all-deployments-have-egress-policy",
		framework.DeploymentKind,
		nil,
		func(ctx framework.ComplianceContext) {
			checkAllDeploymentsHavePolicy(ctx, storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE)
		})
}

func policyIsOfType(spec *storage.NetworkPolicySpec, policyType storage.NetworkPolicyType) bool {
	for _, ty := range spec.GetPolicyTypes() {
		if ty == policyType {
			return true
		}
	}
	return false
}

func checkAllDeploymentsHavePolicy(
	ctx framework.ComplianceContext,
	policyType storage.NetworkPolicyType) {

	networkGraph := ctx.Data().NetworkGraph()
	networkPolicies := ctx.Data().NetworkPolicies()

	var policyTypeName string
	switch policyType {
	case storage.NetworkPolicyType_INGRESS_NETWORK_POLICY_TYPE:
		policyTypeName = "ingress"
	case storage.NetworkPolicyType_EGRESS_NETWORK_POLICY_TYPE:
		policyTypeName = "egress"
	default:
		framework.Abortf(ctx, "unknown network policy type %v", policyType)
	}

	deploymentNodes := make(map[string]*v1.NetworkNode, len(networkGraph.GetNodes()))
	for _, node := range networkGraph.GetNodes() {
		if node.GetEntity().GetType() != storage.NetworkEntityInfo_DEPLOYMENT {
			continue
		}

		deploymentNodes[node.GetEntity().GetId()] = node
	}

	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		node := deploymentNodes[deployment.GetId()]
		if node == nil {
			framework.FailNowf(ctx, "Deployment %s (%s) is not present in network graph", deployment.GetName(), deployment.GetId())
		}

		for _, policyID := range node.GetPolicyIds() {
			policy := networkPolicies[policyID]
			if policy == nil || !policyIsOfType(policy.GetSpec(), policyType) {
				continue
			}
			framework.PassNowf(ctx, "%s network policy %s applies", policyTypeName, policy.GetName())
		}

		framework.Fail(ctx, "No ingress network policies apply, hence all ingress connections are allowed")
	})
}
