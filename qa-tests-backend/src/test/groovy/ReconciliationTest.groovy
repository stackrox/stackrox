import static Services.getViolationsWithTimeout
import static util.Helpers.withRetry

import orchestratormanager.OrchestratorTypes

import io.fabric8.kubernetes.api.model.Pod

import io.stackrox.proto.storage.AlertOuterClass

import common.Constants
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import services.AlertService
import services.BaseService
import services.ClusterService
import services.DevelopmentService
import services.MetadataService
import services.NamespaceService
import services.NetworkPolicyService
import services.SecretService
import util.Env

import spock.lang.IgnoreIf
import spock.lang.Tag

class ReconciliationTest extends BaseSpecification {

    private static final Map<String, Integer> EXPECTED_MIN_DELETIONS_BY_KEY = [
        "*central.SensorEvent_Secret": 1,
        "*central.SensorEvent_Namespace": 1,
        "*central.SensorEvent_Pod": 1,
        "*central.SensorEvent_Role": 0,
        "*central.SensorEvent_NetworkPolicy": 1,
        "*central.SensorEvent_ServiceAccount": 0,
        "*central.SensorEvent_Binding": 0,
        "*central.SensorEvent_Deployment": 1,
        "*central.SensorEvent_Node": 0,
        "*central.SensorEvent_ComplianceOperatorProfile": 0,
        "*central.SensorEvent_ComplianceOperatorProfileV2": 0,
        "*central.SensorEvent_ComplianceOperatorRemediationV2": 0,
        "*central.SensorEvent_ComplianceOperatorResult": 0,
        "*central.SensorEvent_ComplianceOperatorRule": 0,
        "*central.SensorEvent_ComplianceOperatorRuleV2": 0,
        "*central.SensorEvent_ComplianceOperatorScanSettingBinding": 0,
        "*central.SensorEvent_ComplianceOperatorScanSettingBindingV2": 0,
        "*central.SensorEvent_ComplianceOperatorScan": 0,
        "*central.SensorEvent_ComplianceOperatorScanV2": 0,
        "*central.SensorEvent_ComplianceOperatorSuiteV2": 0,
        "*central.SensorEvent_ImageIntegration": 0,
    ]

    private Set<String> getPodsInCluster() {
        return orchestrator.getNamespaces().collectMany { String namespace ->
            List<Pod> allPods = orchestrator.getPodsByLabel(namespace, new HashMap<String, String>())
            allPods.collect { Pod pod -> namespace + ":" + pod.metadata.getName() }
        }
    }

    private Set<String> getDifference(Set<String> list1, Set<String> list2) {
        Set<String> result = list1.clone() as Set<String>
        result.removeAll(list2)
        return result
    }

    private void verifyReconciliationStats() {
        // Cannot verify this on a release build, since the API is not exposed.
        if (MetadataService.isReleaseBuild()) {
            return
        }
        def clusterId = ClusterService.getClusterId()
        def reconciliationStatsForCluster = null
        withRetry(30, 2) {
            BaseService.useBasicAuth()
            reconciliationStatsForCluster = DevelopmentService.
                getReconciliationStatsByCluster().getStatsList().find { it.clusterId == clusterId }
            assert reconciliationStatsForCluster
            assert reconciliationStatsForCluster.getReconciliationDone()
        }
        log.info "Reconciliation stats: ${reconciliationStatsForCluster.deletedObjectsByTypeMap}"
        for (def entry: reconciliationStatsForCluster.getDeletedObjectsByTypeMap().entrySet()) {
            def expectedMinDeletions = EXPECTED_MIN_DELETIONS_BY_KEY.get(entry.getKey())
            assert expectedMinDeletions != null : "Please add object type " +
                "${entry.getKey()} to the map of known reconciled resources in ReconciliationTest.groovy"
            assert entry.getValue() >= expectedMinDeletions: "Number of deletions too low for " +
                    "object type ${entry.getKey()} (got ${entry.getValue()})"
        }
    }

    @Tag("SensorBounce")
    @Tag("COMPATIBILITY")
    // RS-361 - Fails on OSD
    @IgnoreIf({ Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT })
    def "Verify the Sensor reconciles after being restarted"() {
        when:
        "Get Sensor and counts"

        Deployment sensorDeployment = new Deployment().setNamespace(Constants.STACKROX_NAMESPACE).setName("sensor")

        List<AlertOuterClass.ListAlert> violations = []
        Deployment busyboxDeployment
        String secretID
        String networkPolicyID

        def ns = "qa-reconciliation"
        // Deploy a new resource of each type
        // Not possible to test node in this circumstance
        // Requires manual testing

        // Wait is pretty much instantaneous
        def namespaceID = orchestrator.createNamespace(ns)
        NamespaceService.waitForNamespace(namespaceID, 10)

        Set<String> podsBeforeDeleting = [] as Set

        try {
            addStackroxImagePullSecret(ns)

            // Wait is builtin
            secretID = orchestrator.createSecret("testing123", ns)
            SecretService.waitForSecret(secretID, 10)

            busyboxDeployment = new Deployment()
                    .setNamespace(ns)
                    .setName("testing123")
                    .setImage("quay.io/rhacs-eng/qa:busybox")
                    .addPort(22)
                    .addLabel("app", "testing123")
                    .setCommand(["sleep", "600"])

            // Wait is builtin
            orchestrator.createDeployment(busyboxDeployment)
            assert Services.waitForDeployment(busyboxDeployment)
            assert Services.getPods().findAll { it.deploymentId == busyboxDeployment.getDeploymentUid() }.size() == 1

            violations = getViolationsWithTimeout("testing123",
                    "Secure Shell (ssh) Port Exposed", 90)
            assert violations.size() == 1

            NetworkPolicy policy = new NetworkPolicy("do-nothing")
                    .setNamespace(ns)
                    .addPodSelector()
                    .addPolicyType(NetworkPolicyTypes.INGRESS)
            networkPolicyID = orchestrator.applyNetworkPolicy(policy)
            assert NetworkPolicyService.waitForNetworkPolicy(networkPolicyID)

            podsBeforeDeleting = podsInCluster
            log.info "Pods in cluster before deleting:"
            for (pod in podsBeforeDeleting) {
                log.info pod
            }

            List<Pod> pods = orchestrator.getPodsByLabel(Constants.STACKROX_NAMESPACE, ["app": "sensor"])
            assert pods.size() == 1
            orchestrator.scaleDeployment(Constants.STACKROX_NAMESPACE, "sensor", 0)
            // In case the pod gets stuck in `Terminating` state after the scale down,
            // delete Sensor's pod (not deployment) without grace period.
            orchestrator.deletePod(Constants.STACKROX_NAMESPACE, pods[0].getMetadata().getName(), 0)

            orchestrator.waitForAllPodsToBeRemoved(Constants.STACKROX_NAMESPACE, ["app": "sensor"], 30, 5)

            orchestrator.identity {
                // Delete objects from k8s
                deleteDeployment(busyboxDeployment)
                deleteSecret("testing123", ns)
                deleteNetworkPolicy(policy)
            }
        } finally {
            orchestrator.deleteNamespace(ns)
            // Just wait for the namespace to be deleted which is indicative that all of them have been deleted
            orchestrator.waitForNamespaceDeletion(ns)
        }

        Set<String> podsBeforeRestarting = podsInCluster
        log.info "Pods in cluster before restarting:"
        for (pod in podsBeforeRestarting) {
            log.info pod
        }
        log.info "Pods that were likely deleted while sensor was down:"
        def deletedPods = getDifference(podsBeforeDeleting, podsBeforeRestarting)
        for (pod in deletedPods) {
            log.info pod
        }

        // Scale sensor up
        orchestrator.scaleDeployment(Constants.STACKROX_NAMESPACE, "sensor", 1)
        Services.waitForDeployment(sensorDeployment)

        def maxWaitForSync = 200
        def interval = 1

        then:
        "Verify that we don't have references to resources removed when sensor was gone"
        // Get the resources from central and make sure the values exist
        int retries = (int) (maxWaitForSync / interval)
        int numDeployments = -1
        int numPods = -1
        int numNamespaces = -1
        int numNetworkPolicies = -1
        int numSecrets = -1
        withRetry(retries, interval) {
            log.info "Waiting for all resources to be reconciled"
            numDeployments = Services.getDeployments().findAll { it.name == busyboxDeployment.getName() }.size()
            numPods = Services.getPods().findAll { it.deploymentId == busyboxDeployment.getDeploymentUid() }.size()
            numNamespaces = NamespaceService.getNamespaces().findAll { it.metadata.name == ns }.size()
            numNetworkPolicies = NetworkPolicyService.getNetworkPolicies().findAll { it.id == networkPolicyID }.size()
            numSecrets = SecretService.getSecrets().findAll { it.id == secretID }.size()

            assert numDeployments == 0
            assert numPods == 0
            assert numNamespaces == 0
            assert numNetworkPolicies == 0
            assert numSecrets == 0
        }

        // It is possible that more pods will be deleted in the observation period (e.g., Scanner being scaled down).
        // We want to make sure that the pods from the list are gone, so we do not assert on the total number of
        // deletions as this may cause flakes.
        log.info "All pods after reconciliation: ", Services.getPods()
        for (def name: deletedPods) {
            assert Services.getPods().findAll { it.name == name }.size() == 0,
                "Should not find the pod ${name} after reconciliation"
        }

        verifyReconciliationStats()

        // Verify Latest Tag alert is marked as stale
        def violation = AlertService.getViolation(violations[0].getId())
        assert violation.state == AlertOuterClass.ViolationState.RESOLVED
    }

}
