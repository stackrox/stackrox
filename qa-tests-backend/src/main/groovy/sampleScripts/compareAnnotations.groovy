package sampleScripts

import common.Constants
import io.stackrox.proto.storage.Rbac
import objects.K8sRole
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import services.BaseService
import services.RbacService
import util.Env
import util.Helpers

// Repeat the same annotation comparisons found in K8sRbacTest and SummaryTest.

// Use basic authentication for API calls to central. Relies on:
// ROX_USERNAME (defaults to admin)
// ROX_PASSWORD (inferred from the most recent deploy/{k8s,openshift}/central-deploy/password)
// API_HOSTNAME & API_PORT
BaseService.useBasicAuth()
BaseService.setUseClientCert(false)

// Get a cluster client. Assumes you have a working kube configuration. Relies on:
// CLUSTER: Either `OPENSHIFT` or `K8S`. This is inferred from the most recent
//   `deploy/{k8s,openshift}/central-deploy` dir
OrchestratorMain orchestrator = OrchestratorType.create(
        Env.mustGetOrchestratorType(),
        Constants.ORCHESTRATOR_NAMESPACE
)

Logger log = LoggerFactory.getLogger("compare")

def stackroxRoles = RbacService.getRoles()
def orchestratorRoles = orchestrator.getRoles() + orchestrator.getClusterRoles()

Map<String, String> a = new HashMap<String, String>() {{
    put("same", "value")
    put("a_only_key", "value")
    put("different", "a")
}}

Map<String, String> b = new HashMap<String, String>() {{
    put("same", "value")
    put("b_only_key", "value")
    put("different", "b")
}}

Helpers.compareAnnotations(a, b)

assert stackroxRoles.size() == orchestratorRoles.size()
for (Rbac.K8sRole stackroxRole : stackroxRoles) {
    log.info "Looking for SR Role: ${stackroxRole.name} (${stackroxRole.namespace})"
    K8sRole role = orchestratorRoles.find {
        it.name == stackroxRole.name &&
                it.clusterRole == stackroxRole.clusterRole &&
                it.namespace == stackroxRole.namespace
    }
    assert role
    role.annotations.remove("kubectl.kubernetes.io/last-applied-configuration")
    // assert role.annotations == stackroxRole.annotationsMap
    Helpers.compareAnnotations(role.annotations, stackroxRole.annotations)
}
