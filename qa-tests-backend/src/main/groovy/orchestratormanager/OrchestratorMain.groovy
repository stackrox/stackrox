package orchestratormanager

import io.kubernetes.client.models.V1beta1ValidatingWebhookConfiguration
import objects.DaemonSet
import objects.Deployment
import objects.Namespace
import objects.NetworkPolicy
import objects.Node
import objects.Service

interface OrchestratorMain {
    def setup()
    def cleanup()

    //Deployments
    io.fabric8.kubernetes.api.model.apps.Deployment getOrchestratorDeployment(String ns, String name)
    def createOrchestratorDeployment(io.fabric8.kubernetes.api.model.apps.Deployment dep)

    def createDeploymentNoWait(Deployment deployment)
    def createDeployment(Deployment deployment)
    def batchCreateDeployments(List<Deployment> deployments)
    def deleteDeployment(Deployment deployment)
    def waitForDeploymentDeletion(Deployment deploy)
    String getDeploymentId(Deployment deployment)
    def getDeploymentReplicaCount(Deployment deployment)
    def getDeploymentUnavailableReplicaCount(Deployment deployment)
    def getDeploymentNodeSelectors(Deployment deployment)
    def getDeploymentCount(String ns)
    Set<String> getDeploymentSecrets(Deployment deployment)

    //DaemonSets
    def createDaemonSet(DaemonSet daemonSet)
    def deleteDaemonSet(DaemonSet daemonSet)
    def getDaemonSetReplicaCount(DaemonSet daemonSet)
    def getDaemonSetNodeSelectors(DaemonSet daemonSet)
    def getDaemonSetUnavailableReplicaCount(DaemonSet daemonSet)
    def getDaemonSetCount(String ns)
    def waitForDaemonSetDeletion(String name)

    //Containers
    def deleteContainer(String containerName, String namespace)
    def wasContainerKilled(String containerName, String namespace)
    def isKubeProxyPresent()
    def isKubeDashboardRunning()
    def getContainerlogs(Deployment deployment)
    def getStaticPodCount(String ns)

    //Services
    def createService(Deployment deployment)
    def createService(Service service)
    def deleteService(String serviceName, String namespace)
    def waitForServiceDeletion(Service service)

    //Secrets
    def createSecret(String name, String namespace)
    def deleteSecret(String name, String namespace)
    def getSecretCount(String ns)

    //Namespaces
    String createNamespace(String ns)
    def deleteNamespace(String ns)
    def waitForNamespaceDeletion(String ns)
    def getAllNetworkPoliciesNamesByNamespace(Boolean ignoreUndoneStackroxGenerated)

    //NetworkPolicies
    String applyNetworkPolicy(NetworkPolicy policy)
    boolean deleteNetworkPolicy(NetworkPolicy policy)
    def getNetworkPolicyCount(String ns)

    //Nodes
    def getNodeCount()
    List<Node> getNodeDetails()
    def supportsNetworkPolicies()

    //Namespaces
    List<Namespace> getNamespaceDetails()

    //Misc
    def createClairifyDeployment()
    String getClairifyEndpoint()
    String generateYaml(Object orchestratorObject)
    String getNameSpace()
    String getSensorContainerName()
    def waitForSensor()

    V1beta1ValidatingWebhookConfiguration getAdmissionController()
    def deleteAdmissionController(String name)
    def createAdmissionController(V1beta1ValidatingWebhookConfiguration config)

    /*TODO:
        def getDeploymenton(String deploymentName)
        def updateDeploymenton()
    */
}
