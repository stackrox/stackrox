package orchestratormanager

interface OrchestratorMain {
    def setup()
    def cleanup()

    def createDeployment(objects.Deployment deployment)
    /*TODO:
        def getDeploymenton(String deploymentName)
        def updateDeploymenton()
    */
    def deleteDeployment(String deploymentName)
}
