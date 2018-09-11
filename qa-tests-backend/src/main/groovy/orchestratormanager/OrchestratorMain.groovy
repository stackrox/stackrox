package orchestratormanager

interface OrchestratorMain {
    def setup()
    def cleanup()

    def createDeployment(objects.Deployment deployment)
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
}
