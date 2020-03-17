import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import services.NamespaceService
import services.NetworkPolicyService
import services.SecretService

import org.junit.experimental.categories.Category
import groups.SensorBounce
import util.Timer

class ReconciliationTest extends BaseSpecification {

    @Category(SensorBounce)
    def "Verify the Sensor reconciles after being restarted"() {
        when:
        "Get Sensor and counts"
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

        // Wait is builtin
        orchestrator.createDeployment(dep)
        assert Services.waitForDeployment(dep)
        // Testing out feature behind the ROX_POD_DEPLOY_SEPARATE feature flag
        assert Services.getPods().findAll { it.deploymentId == dep.getDeploymentUid() }.size() == 1

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
    }

}
