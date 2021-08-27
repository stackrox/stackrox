import static Services.getResourceViolationsWithTimeout

import common.Constants
import groups.BAT
import groups.RUNTIME
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ScopeOuterClass
import objects.Secret
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.PolicyService
import spock.lang.Stepwise
import spock.lang.Unroll

@Stepwise
class AuditLogAlertsTest extends BaseSpecification {
    @Unroll
    @Category([BAT, RUNTIME])
    def "Verify Audit Log Event Source Policies Trigger: #verb - #resourceType"() {
        given:
        "Running on an OpenShift 4 cluster"
        Assume.assumeTrue("Audit Log alerts are only supported on OpenShift 4", ClusterService.isOpenShift4())

        when:
        "An audit log event source policy is created"

        // add some randomness so that an older test run doesn't poison the result (because the audit log entries
        // may still exist on the cluster.)
        def resName = "e2e-test-rez" + UUID.randomUUID()
        def policy = PolicyOuterClass.Policy.newBuilder()
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
        def policyId = PolicyService.createNewPolicy(policy)
        assert policyId

        and:
        "The resource is created, accessed and deleted"
        if (resourceType == "SECRETS") {
            createGetAndDeleteSecret(resName, Constants.ORCHESTRATOR_NAMESPACE)
        } else if (resourceType == "CONFIGMAPS") {
            createGetAndDeleteConfigMap(resName, Constants.ORCHESTRATOR_NAMESPACE)
        }

        then:
        "Verify that policy was violated"
        def violations =  getResourceViolationsWithTimeout(resourceType, resName, policy.getName(), 60)
        // There should be exactly one violation because we are testing only verb at a time
        assert violations != null && violations.size() == 1

        cleanup:
        if (policyId) {
            PolicyService.deletePolicy(policyId)
        }

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

    def createGetAndDeleteSecret(String name, String namespace) {
        Secret testSecret = new Secret()
        testSecret.name = name
        testSecret.type = "generic"
        testSecret.namespace = namespace
        testSecret.data = [
                "value": Base64.getEncoder().encodeToString("sooper sekret".getBytes()),
        ]

        orchestrator.createSecret(testSecret)
        orchestrator.getSecret(name, namespace)
        orchestrator.deleteSecret(name, namespace)
    }

    def createGetAndDeleteConfigMap(String name, String namespace) {
        orchestrator.createConfigMap(name, ["value": "map me"], namespace)
        orchestrator.getConfigMap(name, namespace)
        orchestrator.deleteConfigMap(name, namespace)
    }
}
