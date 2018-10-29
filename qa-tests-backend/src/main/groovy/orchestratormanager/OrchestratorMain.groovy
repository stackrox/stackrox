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
    String getDeploymentId(Deployment deployment)
    def getDeploymentReplicaCount(Deployment deployment)
    def getDeploymentUnavailableReplicaCount(Deployment deployment)
    def getDeploymentNodeSelectors(Deployment deployment)

    //DaemonSets
    def createDaemonSet(DaemonSet daemonSet)
    def deleteDaemonSet(DaemonSet daemonSet)
    def getDaemonSetReplicaCount(DaemonSet daemonSet)
    def getDaemonSetNodeSelectors(DaemonSet daemonSet)
    def getDaemonSetUnavailableReplicaCount(DaemonSet daemonSet)

    //Containers
    String getpods()
    def wasContainerKilled(String containerName, String namespace)

    //Services
    def deleteService(String serviceName, String namespace)

    //Secrets
    def createSecret(String name)
    def deleteSecret(String name, String namespace)

    //NetworkPolicies
    String applyNetworkPolicy(NetworkPolicy policy)
    boolean deleteNetworkPolicy(NetworkPolicy policy)

    //Misc
    def createClairifyDeployment()
    String getClairifyEndpoint()
    String generateYaml(Object orchestratorObject)

    /*TODO:
        def getDeploymenton(String deploymentName)
        def updateDeploymenton()
    */
}
