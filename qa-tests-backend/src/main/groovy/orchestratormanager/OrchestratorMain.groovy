package orchestratormanager

import objects.DaemonSet
import objects.Deployment
import objects.NetworkPolicy

interface OrchestratorMain {
    def setup()
    def cleanup()

    //Deployments
    def createDeployment(Deployment deployment)
    def batchCreateDeployments(List<Deployment> deployments)
    def deleteDeployment(Deployment deployment)
    def waitForDeploymentDeletion(Deployment deploy)
    String getDeploymentId(Deployment deployment)
    def getDeploymentReplicaCount(Deployment deployment)
    def getDeploymentUnavailableReplicaCount(Deployment deployment)
    def getDeploymentNodeSelectors(Deployment deployment)
    def getDeploymentCount()
    Set<String> getDeploymentSecrets(Deployment deployment)

    //DaemonSets
    def createDaemonSet(DaemonSet daemonSet)
    def deleteDaemonSet(DaemonSet daemonSet)
    def getDaemonSetReplicaCount(DaemonSet daemonSet)
    def getDaemonSetNodeSelectors(DaemonSet daemonSet)
    def getDaemonSetUnavailableReplicaCount(DaemonSet daemonSet)
    def getDaemonSetCount()
    def waitForDaemonSetDeletion(String name)

    //Containers
    def wasContainerKilled(String containerName, String namespace)
    def isKubeProxyPresent()
    def isKubeDashboardRunning()
    def getContainerlogs(Deployment deployment)

    //Services
    def deleteService(String serviceName, String namespace)
    def createService(Deployment deployment)

    //Secrets
    def createSecret(String name)
    def deleteSecret(String name, String namespace)
    def getSecretCount()

    //NetworkPolicies
    String applyNetworkPolicy(NetworkPolicy policy)
    boolean deleteNetworkPolicy(NetworkPolicy policy)

    //Nodes
    def getNodeCount()
    def supportsNetworkPolicies()

    //Misc
    def createClairifyDeployment()
    String getClairifyEndpoint()
    String generateYaml(Object orchestratorObject)
    String getNameSpace()

    /*TODO:
        def getDeploymenton(String deploymentName)
        def updateDeploymenton()
    */
}
