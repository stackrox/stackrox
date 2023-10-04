import static Services.getAllResourceViolationsWithTimeout
import static Services.getResourceViolationsWithTimeout

import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ScopeOuterClass

import common.Constants
import objects.Secret
import services.AlertService
import services.ClusterService
import services.PolicyService

import spock.lang.Requires
import spock.lang.Stepwise
import spock.lang.Tag
import spock.lang.Unroll
import util.Env

// Audit Log alerts are only supported on OpenShift 4
@Requires({ Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT })
@Stepwise
class AuditLogAlertsTest extends BaseSpecification {
    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT = 60

    @Unroll
    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("PZ")
    def "Verify Audit Log Event Source Policies Trigger: #verb - #resourceType"() {
        when:
        "Audit log collection is enabled"
        def previouslyDisabled = ClusterService.getCluster().getDynamicConfig().getDisableAuditLogs()
        if (previouslyDisabled) {
            assert ClusterService.updateAuditLogDynamicConfig(false)
        }

        and:
        "An audit log event source policy is created"

        // add some randomness so that an older test run doesn't poison the result (because the audit log entries
        // may still exist on the cluster.)
        def resName = "e2e-test-rez" + UUID.randomUUID()
        def policy = createAuditLogSourcePolicy(resName, verb, resourceType)
        def policyId = PolicyService.createNewPolicy(policy)
        assert policyId
        sleep(5000) // wait 5s for the policy top propagate to sensor

        and:
        "The resource is created, accessed and deleted"
        if (resourceType == "SECRETS") {
            createGetAndDeleteSecret(resName, Constants.ORCHESTRATOR_NAMESPACE)
        } else if (resourceType == "CONFIGMAPS") {
            createGetAndDeleteConfigMap(resName, Constants.ORCHESTRATOR_NAMESPACE)
        }

        then:
        "Verify that policy was violated"
        def violations =  getResourceViolationsWithTimeout(resourceType, resName,
                policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)
        // There should be exactly one violation because we are testing only verb at a time
        assert violations != null && violations.size() == 1

        cleanup:
        if (policyId) {
            PolicyService.deletePolicy(policyId)
        }
        // set the feature back to what it was
        assert ClusterService.updateAuditLogDynamicConfig(previouslyDisabled)

        where:
        "Data inputs are:"

        resourceType | verb

        "SECRETS"    | "CREATE"
        "SECRETS"    | "GET"
        "SECRETS"    | "DELETE"
        "CONFIGMAPS" | "CREATE"
        "CONFIGMAPS" | "GET"
        "CONFIGMAPS" | "DELETE"
    }

    @Unroll
    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("PZ")
    def "Verify collection continues even after ACS components restarts: #component"() {
        when:
        "Audit log collection is enabled"
        def previouslyDisabled = ClusterService.getCluster().getDynamicConfig().getDisableAuditLogs()
        if (previouslyDisabled) {
            assert ClusterService.updateAuditLogDynamicConfig(false)
        }

        and:
        "An audit log event source policy is created"

        // add some randomness so that an older test run doesn't poison the result (because the audit log entries
        // may still exist on the cluster.)
        def resName = "e2e-test-rez" + UUID.randomUUID()
        def policy = createAuditLogSourcePolicy(resName, "GET", "CONFIGMAPS")
        def policyId = PolicyService.createNewPolicy(policy)
        assert policyId
        sleep(5000) // wait 5s for the policy top propagate to sensor

        and:
        "A violation is generated and resolved"
        createGetAndDeleteConfigMap(resName, Constants.ORCHESTRATOR_NAMESPACE)
        def violations =  getResourceViolationsWithTimeout("CONFIGMAPS", resName,
                policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)
        // There should be exactly one violation
        assert violations != null && violations.size() == 1

        AlertService.resolveAlert(violations[0].getId())

        and:
        "${component} is restarted thus collection is restarted"
        orchestrator.restartPodByLabels("stackrox", [app: component], 30, 5)

        and:
        "Another violation is generated"
        def altResourceName = resName + "-2"
        createGetAndDeleteConfigMap(altResourceName, Constants.ORCHESTRATOR_NAMESPACE)

        then:
        "Verify that only the access after restart triggers a violation"
        def allViolations =  getAllResourceViolationsWithTimeout("CONFIGMAPS",
                policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)

        // There should only be one violation - the new one
        assert allViolations != null &&
                allViolations.size() == 1 &&
                allViolations[0].resource.name == altResourceName

        cleanup:
        if (policyId) {
            PolicyService.deletePolicy(policyId)
        }
        // set the feature back to what it was
        assert ClusterService.updateAuditLogDynamicConfig(previouslyDisabled)

        where:
        "Data inputs are:"

        component   | _

        "sensor"    | _
        "collector" | _
        // Note: central restart isn't being tested here because unfortunately killing the central pod stops
        // the port forward and fails the rest of the test suite.
    }

    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("PZ")
    def "Verify collection continues when it is disabled and then re-enabled"() {
        when:
        "Audit log collection is enabled"
        def previouslyDisabled = ClusterService.getCluster().getDynamicConfig().getDisableAuditLogs()
        if (previouslyDisabled) {
            assert ClusterService.updateAuditLogDynamicConfig(false)
        }

        and:
        "An audit log event source policy is created"

        // add some randomness so that an older test run doesn't poison the result (because the audit log entries
        // may still exist on the cluster.)
        def resName = "e2e-test-rez" + UUID.randomUUID()
        def policy = createAuditLogSourcePolicy(resName, "GET", "CONFIGMAPS")
        def policyId = PolicyService.createNewPolicy(policy)
        assert policyId
        sleep(5000) // wait 5s for the policy top propagate to sensor

        and:
        "A violation is generated and resolved"
        createGetAndDeleteConfigMap(resName, Constants.ORCHESTRATOR_NAMESPACE)
        def violations =  getResourceViolationsWithTimeout("CONFIGMAPS", resName,
                policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)
        // There should be exactly one violation
        assert violations != null && violations.size() == 1

        AlertService.resolveAlert(violations[0].getId())

        and:
        "Feature is disabled and then re-enabled"
        assert ClusterService.updateAuditLogDynamicConfig(true)
        sleep(5000) // wait 5s for it to propagate to sensor before re-enabling
        assert ClusterService.updateAuditLogDynamicConfig(false)
        sleep(5000) // wait 5s for it to propagate again

        and:
        "Another violation is generated"
        def altResourceName = resName + "-2"
        createGetAndDeleteConfigMap(altResourceName, Constants.ORCHESTRATOR_NAMESPACE)

        then:
        "Verify that only the access after restart triggers a violation"
        def allViolations =  getAllResourceViolationsWithTimeout("CONFIGMAPS",
                policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)

        // There should only be one violation - the new one
        assert allViolations != null &&
                allViolations.size() == 1 &&
                allViolations[0].resource.name == altResourceName

        cleanup:
        if (policyId) {
            PolicyService.deletePolicy(policyId)
        }
        // set the feature back to what it was
        assert ClusterService.updateAuditLogDynamicConfig(previouslyDisabled)
    }

    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("PZ")
    def "Verify collection stops when feature is is disabled"() {
        when:
        "Audit log collection is disabled"
        def previouslyDisabled = ClusterService.getCluster().getDynamicConfig().getDisableAuditLogs()
        assert ClusterService.updateAuditLogDynamicConfig(true)

        and:
        "An audit log event source policy is created"

        // add some randomness so that an older test run doesn't poison the result (because the audit log entries
        // may still exist on the cluster.)
        def resName = "e2e-test-rez" + UUID.randomUUID()
        def policy = createAuditLogSourcePolicy(resName, "GET", "CONFIGMAPS")
        def policyId = PolicyService.createNewPolicy(policy)
        assert policyId
        sleep(5000) // wait 5s for the policy to propagate to sensor

        and:
        "The resource is accessed"
        createGetAndDeleteConfigMap(resName, Constants.ORCHESTRATOR_NAMESPACE)

        then:
        "Verify that no violations were generated"
        def violations =  getResourceViolationsWithTimeout("CONFIGMAPS", resName,
                policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)
        assert violations == null || violations.size() == 0

        cleanup:
        if (policyId) {
            PolicyService.deletePolicy(policyId)
        }
        // set the feature back to what it was
        assert ClusterService.updateAuditLogDynamicConfig(previouslyDisabled)
    }

    def createAuditLogSourcePolicy(String resName, String verb, String resourceType) {
        return PolicyOuterClass.Policy.newBuilder()
                .setName("e2e-test-detect-${verb}-${resourceType}")
                .addLifecycleStages(PolicyOuterClass.LifecycleStage.RUNTIME)
                .setEventSource(PolicyOuterClass.EventSource.AUDIT_LOG_EVENT)
                .addCategories("Test")
                .setDisabled(false)
                .setSeverity(PolicyOuterClass.Severity.CRITICAL_SEVERITY)
                .addScope(
                        ScopeOuterClass.Scope.newBuilder().setNamespace(Constants.ORCHESTRATOR_NAMESPACE).build()
                )
                .addPolicySections(
                        PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                PolicyOuterClass.PolicyGroup.newBuilder()
                                        .setFieldName("Kubernetes Resource")
                                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(resourceType))
                        ).addPolicyGroups(
                                PolicyOuterClass.PolicyGroup.newBuilder()
                                        .setFieldName("Kubernetes API Verb")
                                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(verb))
                        ).addPolicyGroups(
                                PolicyOuterClass.PolicyGroup.newBuilder()
                                        .setFieldName("Kubernetes Resource Name")
                                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(resName))
                        )
                ).build()
    }

    def createGetAndDeleteSecret(String name, String namespace) {
        Secret testSecret = new Secret()
        testSecret.name = name
        testSecret.type = "generic"
        testSecret.namespace = namespace
        testSecret.data = [
                "value": Base64.getEncoder().encodeToString("sooper sekret".getBytes()),
        ]
        // some breather needed on few arches
        if (Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x") {
            sleep(5000)
        }
        orchestrator.createSecret(testSecret)
        orchestrator.getSecret(name, namespace)
        orchestrator.deleteSecret(name, namespace)
    }

    def createGetAndDeleteConfigMap(String name, String namespace) {
        // some breather needed on few arches
        if (Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x") {
            sleep(5000)
        }
        orchestrator.createConfigMap(name, ["value": "map me"], namespace)
        orchestrator.getConfigMap(name, namespace)
        orchestrator.deleteConfigMap(name, namespace)
    }
}
