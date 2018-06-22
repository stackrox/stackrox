package orchestratormanager

interface OrchestratorMain {
    def setup()
    def cleanup()

    String getDeploymentName()
    void setDeploymentName(String metaName)
    void addMetaLables(String labelName, String labelValue)
    void setReplicasNum(int replicasNum)
    void addTemplateLabels(String labelName, String labelValue)
    void setContainersImage(String containersImage)
    void setContainersName(String containersName)
    void addContainerPort(int port)

    boolean createDeployment()
    /*TODO:
        def getDeploymenton(String deploymentName)
        def updateDeploymenton()
    */
    def deleteDeployment(String deploymentName)
    def getPodOrContainerIds(String deploymentName, String namespace)

}
