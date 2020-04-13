import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import services.ClusterService
import services.DevelopmentService
import services.FeatureFlagService
import services.MetadataService
import services.NamespaceService
import services.NetworkPolicyService
import services.SecretService

import org.junit.experimental.categories.Category
import groups.SensorBounce
import util.Timer

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
    ]

    // MAX_ALLOWED_DELETIONS is the max number of deletions allowed for a resource.
    // It aims to detect overly aggressive reconciliation.
    private static final Integer MAX_ALLOWED_DELETIONS = 3

    private void verifyReconciliationStats(boolean verifyMin) {
        // Cannot verify this on a release build, since the API is not exposed.
        if (MetadataService.isReleaseBuild()) {
            return
        }
        def clusterId = ClusterService.getClusterId()
        def reconciliationStatsForCluster = DevelopmentService.
            getReconciliationStatsByCluster().getStatsList().find { it.clusterId == clusterId }
        assert reconciliationStatsForCluster
        assert reconciliationStatsForCluster.getReconciliationDone()
        println "Reconciliation stats: ${reconciliationStatsForCluster.deletedObjectsByTypeMap}"
        for (def entry: reconciliationStatsForCluster.getDeletedObjectsByTypeMap().entrySet()) {
            def expectedMinDeletions = EXPECTED_MIN_DELETIONS_BY_KEY.get(entry.getKey())
            assert expectedMinDeletions != null : "Please add object type " +
                "${entry.getKey()} to the map of known reconciled resources in ReconciliationTest.groovy"
            if (verifyMin) {
                assert entry.getValue() >= expectedMinDeletions: "Number of deletions too low for " +
                    "object type ${entry.getKey()} (got ${entry.getValue()})"
            }
            assert entry.getValue() <= MAX_ALLOWED_DELETIONS : "Overly aggressive reconciliation for " +
                "object type ${entry.getKey()} (got ${entry.getValue()})"
        }
    }

    @Category(SensorBounce)
    def "Verify the Sensor reconciles after being restarted"() {
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
                .setImage ("busybox")
                .addLabel ("app", "testing123")
                .setCommand(["sleep", "600"])

        def podDeploySeparate = FeatureFlagService.isFeatureFlagEnabled("ROX_POD_DEPLOY_SEPARATE")

        // Wait is builtin
        orchestrator.createDeployment(dep)
        assert Services.waitForDeployment(dep)
        if (podDeploySeparate) {
            assert Services.getPods().findAll { it.deploymentId == dep.getDeploymentUid() }.size() == 1
        }

        NetworkPolicy policy = new NetworkPolicy("do-nothing")
                .setNamespace(ns)
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def networkPolicyID = orchestrator.applyNetworkPolicy(policy)
        assert NetworkPolicyService.waitForNetworkPolicy(networkPolicyID)

        def sensorDeployment = new Deployment().setNamespace("stackrox").setName("sensor")
        orchestrator.deleteAndWaitForDeploymentDeletion(sensorDeployment)

        // Delete objects from k8s
        orchestrator.identity {
            deleteDeployment(dep)
            deleteSecret("testing123", ns)
            deleteNetworkPolicy(policy)
            deleteNamespace(ns)
            // Just wait for the namespace to be deleted which is indicative that all of them have been deleted
            waitForNamespaceDeletion(ns)

            createOrchestratorDeployment(sensor)
        }

        // Recreate sensor
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
            if (podDeploySeparate) {
                numPods = Services.getPods().findAll { it.deploymentId == dep.getDeploymentUid() }.size()
            }
            numNamespaces = NamespaceService.getNamespaces().findAll { it.metadata.name == ns }.size()
            numNetworkPolicies = NetworkPolicyService.getNetworkPolicies().findAll { it.id == networkPolicyID }.size()
            numSecrets = SecretService.getSecrets().findAll { it.id == secretID }.size()

            if (numDeployments + numPods + numNamespaces + numNetworkPolicies + numSecrets == 0) {
                break
            }
            println "Waiting for all resources to be reconciled"
        }
        assert numDeployments == 0
        if (podDeploySeparate) {
            assert numPods == 0
        }
        assert numNamespaces == 0
        assert numNetworkPolicies == 0
        assert numSecrets == 0

        verifyReconciliationStats(true)
    }

}
