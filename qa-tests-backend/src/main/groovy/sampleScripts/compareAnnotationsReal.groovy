package sampleScripts

import common.Constants
import io.stackrox.proto.storage.NodeOuterClass.Node
import io.stackrox.proto.storage.Rbac
import objects.K8sRole
import objects.K8sRoleBinding
import orchestratormanager.Kubernetes
import orchestratormanager.OrchestratorType
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import services.BaseService
import services.NodeService
import services.RbacService
import util.Env
import util.Helpers

// Repeat the same annotation comparisons found in K8sRbacTest and SummaryTest.

// Use basic authentication for API calls to central. Relies on:
// ROX_USERNAME (defaults to admin)
// ROX_ADMIN_PASSWORD (inferred from the most recent deploy/{k8s,openshift}/central-deploy/password)
// API_HOSTNAME & API_PORT
BaseService.useBasicAuth()
BaseService.setUseClientCert(false)

// Get a cluster client. Assumes you have a working kube configuration. Relies on:
// CLUSTER: Either `OPENSHIFT` or `K8S`. This is inferred from the most recent
//   `deploy/{k8s,openshift}/central-deploy` dir
Kubernetes orchestrator = OrchestratorType.create(
        Env.mustGetOrchestratorType(),
        Constants.ORCHESTRATOR_NAMESPACE
)

Logger log = LoggerFactory.getLogger("scripts")

// K8sRbacTest

def stackroxRoles = RbacService.getRoles()
def orchestratorRoles = orchestrator.getRoles() + orchestrator.getClusterRoles()

assert stackroxRoles.size() == orchestratorRoles.size()

for (Rbac.K8sRole stackroxRole : stackroxRoles) {
    log.info "Looking for orchestrator role to match SR role: ${stackroxRole.name} (${stackroxRole.namespace})"
    K8sRole role = orchestratorRoles.find {
        it.name == stackroxRole.name &&
                it.clusterRole == stackroxRole.clusterRole &&
                it.namespace == stackroxRole.namespace
    }
    assert role
    role.annotations.remove("kubectl.kubernetes.io/last-applied-configuration")
    Helpers.compareAnnotations(role.annotations, stackroxRole.annotationsMap)
}

def stackroxBindings = RbacService.getRoleBindings()
def orchestratorBindings = orchestrator.getRoleBindings() + orchestrator.getClusterRoleBindings()

stackroxBindings.each { Rbac.K8sRoleBinding b ->
    log.info "Looking for orchestrator binding to match SR binding: ${b.name} (${b.namespace})"
    K8sRoleBinding binding = orchestratorBindings.find {
        it.name == b.name && it.namespace == b.namespace
    }
    assert binding != null

    binding.annotations.remove("kubectl.kubernetes.io/last-applied-configuration")
    Helpers.compareAnnotations(binding.annotations, b.annotationsMap)
}

// SummaryTest

List<Node> stackroxNodes = NodeService.getNodes()
List<objects.Node> orchestratorNodes = orchestrator.getNodeDetails()

assert stackroxNodes.size() == orchestratorNodes.size()

for (Node stackroxNode : stackroxNodes) {
    objects.Node orchestratorNode = orchestratorNodes.find { it.uid == stackroxNode.id }
    Helpers.compareAnnotations(orchestratorNode.annotations, stackroxNode.getAnnotationsMap())
}
