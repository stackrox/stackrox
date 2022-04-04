import static Services.getViolationsWithTimeout

import io.stackrox.proto.storage.AlertOuterClass
import services.AlertService
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import services.ClusterService
import services.DevelopmentService
import services.MetadataService
import services.NamespaceService
import services.NetworkPolicyService
import services.SecretService

import spock.lang.Retry
import org.junit.Assume
import org.junit.experimental.categories.Category
import groups.SensorBounce
import util.Timer

@Retry(count = 0)
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
        "*central.SensorEvent_ComplianceOperatorResult": 0,
        "*central.SensorEvent_ComplianceOperatorRule": 0,
        "*central.SensorEvent_ComplianceOperatorScanSettingBinding": 0,
        "*central.SensorEvent_ComplianceOperatorScan": 0,
    ]

    // DEFAULT_MAX_ALLOWED_DELETIONS is the default max number of deletions allowed for a resource.
    // It aims to detect overly aggressive reconciliation.
    private static final Integer DEFAULT_MAX_ALLOWED_DELETIONS = 3

    // MAX_ALLOWED_DELETIONS_BY_KEY is the max number of deletions allowed per resource.
    // It aims to detect overly aggressive reconciliation.
    private static final Map<String, Integer> MAX_ALLOWED_DELETIONS_BY_KEY = [
        // We create and delete an entire namespace, so we may see a lot of secrets being deleted, esp in OpenShift.
        "*central.SensorEvent_Secret": 5,
    ]

    private static void verifyReconciliationStats(boolean verifyMin) {
        // Cannot verify this on a release build, since the API is not exposed.
        if (MetadataService.isReleaseBuild()) {
            return
        }
        def clusterId = ClusterService.getClusterId()
        def reconciliationStatsForCluster = null
        withRetry(30, 2) {
            reconciliationStatsForCluster = DevelopmentService.
                getReconciliationStatsByCluster().getStatsList().find { it.clusterId == clusterId }
            assert reconciliationStatsForCluster
            assert reconciliationStatsForCluster.getReconciliationDone()
        }
        println "Reconciliation stats: ${reconciliationStatsForCluster.deletedObjectsByTypeMap}"
        for (def entry: reconciliationStatsForCluster.getDeletedObjectsByTypeMap().entrySet()) {
            def expectedMinDeletions = EXPECTED_MIN_DELETIONS_BY_KEY.get(entry.getKey())
            assert expectedMinDeletions != null : "Please add object type " +
                "${entry.getKey()} to the map of known reconciled resources in ReconciliationTest.groovy"
            if (verifyMin) {
                assert entry.getValue() >= expectedMinDeletions: "Number of deletions too low for " +
                    "object type ${entry.getKey()} (got ${entry.getValue()})"
            }
            def maxAllowedDeletions = MAX_ALLOWED_DELETIONS_BY_KEY.getOrDefault(
                entry.getKey(), DEFAULT_MAX_ALLOWED_DELETIONS)
            assert entry.getValue() <= maxAllowedDeletions: "Overly aggressive reconciliation for " +
                "object type ${entry.getKey()} (got ${entry.getValue()})"
        }
    }

    @Category(SensorBounce)
    def "Verify the Sensor reconciles after being restarted"() {
        // RS-361 - Fails on OSD. Need help troubleshooting. Disabling for now.
        Assume.assumeFalse(ClusterService.isOpenShift3())
        Assume.assumeFalse(ClusterService.isOpenShift4())

        when:
        "Get Sensor and counts"

        // Verify initial reconciliation stats (from the reconciliation that must have happened
        // whenever the sensor first connected).
        verifyReconciliationStats(false)

        def sensor = orchestrator.getOrchestratorDeployment("stackrox", "sensor")

        def ns = "reconciliation"
        // Deploy a new resource of each type
        // Not possible to test node in this circumstance
        // Requires manual testing

        // Wait is pretty much instantaneous
        def namespaceID = orchestrator.createNamespace(ns)
        NamespaceService.waitForNamespace(namespaceID, 10)

        // Wait is builtin
        def secretID = orchestrator.createSecret("testing123", ns)
        SecretService.waitForSecret(secretID, 10)

        Deployment dep = new Deployment()
                .setNamespace(ns)
                .setName ("testing123")
                .setImage ("quay.io/rhacs-eng/qa:busybox")
                .addPort (22)
                .addLabel ("app", "testing123")
                .setCommand(["sleep", "600"])

        // Wait is builtin
        orchestrator.createDeployment(dep)
        assert Services.waitForDeployment(dep)
        assert Services.getPods().findAll { it.deploymentId == dep.getDeploymentUid() }.size() == 1

        def violations = getViolationsWithTimeout("testing123",
                "Secure Shell (ssh) Port Exposed", 30)
        assert violations.size() == 1

        NetworkPolicy policy = new NetworkPolicy("do-nothing")
                .setNamespace(ns)
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def networkPolicyID = orchestrator.applyNetworkPolicy(policy)
        assert NetworkPolicyService.waitForNetworkPolicy(networkPolicyID)

        def sensorDeployment = new Deployment().setNamespace("stackrox").setName("sensor")
        orchestrator.deleteAndWaitForDeploymentDeletion(sensorDeployment)

        def labels = ["app":"sensor"]
        orchestrator.waitForAllPodsToBeRemoved("stackrox", labels)

        orchestrator.identity {
            // Delete objects from k8s
            deleteDeployment(dep)
            deleteSecret("testing123", ns)
            deleteNetworkPolicy(policy)
            deleteNamespace(ns)
            // Just wait for the namespace to be deleted which is indicative that all of them have been deleted
            waitForNamespaceDeletion(ns)

            // Recreate sensor
            try {
                createOrchestratorDeployment(sensor)
            } catch (Exception e) {
                println "Error re-creating the sensor: " + e
                throw e
            }
        }

        Services.waitForDeployment(sensorDeployment)

        def maxWaitForSync = 100
        def interval = 1

        then:
        "Verify that we don't have references to resources removed when sensor was gone"
        // Get the resources from central and make sure the values exist
        int retries = maxWaitForSync / interval
        Timer t = new Timer(retries, interval)
        int numDeployments, numPods, numNamespaces, numNetworkPolicies, numSecrets
        while (t.IsValid()) {
            numDeployments = Services.getDeployments().findAll { it.name == dep.getName() }.size()
            numPods = Services.getPods().findAll { it.deploymentId == dep.getDeploymentUid() }.size()
            numNamespaces = NamespaceService.getNamespaces().findAll { it.metadata.name == ns }.size()
            numNetworkPolicies = NetworkPolicyService.getNetworkPolicies().findAll { it.id == networkPolicyID }.size()
            numSecrets = SecretService.getSecrets().findAll { it.id == secretID }.size()

            if (numDeployments + numPods + numNamespaces + numNetworkPolicies + numSecrets == 0) {
                break
            }
            println "Waiting for all resources to be reconciled"
        }
        assert numDeployments == 0
        assert numPods == 0
        assert numNamespaces == 0
        assert numNetworkPolicies == 0
        assert numSecrets == 0

        verifyReconciliationStats(true)

        // Verify Latest Tag alert is marked as stale
        def violation = AlertService.getViolation(violations[0].getId())
        assert violation.state == AlertOuterClass.ViolationState.RESOLVED
    }

}
