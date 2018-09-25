package orchestratormanager

import objects.Deployment
import objects.NetworkPolicy

interface OrchestratorMain {
    def setup()
    def cleanup()

    def createDeployment(Deployment deployment)
    /*TODO:
        def getDeploymenton(String deploymentName)
        def updateDeploymenton()
    */
    def deleteDeployment(String deploymentName, String namespace)
    def deleteService(String serviceName, String namespace)
    def createClairifyDeployment()
    String getClairifyEndpoint()
    def createSecret(String name)
    def deleteSecret(String name, String namespace)
    String applyNetworkPolicy(NetworkPolicy policy)
    boolean deleteNetworkPolicy(NetworkPolicy policy)
    String generateYaml(Object orchestratorObject)
    def wasContainerKilled(String containerName, String namespace)
    def getDeploymentReplicaCount(Deployment deployment)
    def getDeploymentUnavailableReplicaCount(Deployment deployment)
    def getDeploymentNodeSelectors(Deployment deployment)
}
