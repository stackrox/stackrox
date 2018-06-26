package orchestratormanager

import objects.DockerEEDeployment

class DockerEE extends OrchestratorCommon implements OrchestratorMain {
    private DockerEEDeployment dockerEEDeployment
    private final String workerName
    private final String cliCommand = "docker"
    private final List<String> envp = new LinkedList<>()

    DockerEE() {
        envp.add("DOCKER_CERT_PATH=" + System.getenv("DOCKER_CERT_PATH"))
        envp.add("DOCKER_HOST=" + System.getenv("DOCKER_HOST"))
        envp.add("DOCKER_API_VERSION=" + System.getenv("DOCKER_API_VERSION"))
        envp.add("DOCKER_TLS_VERIFY=" + System.getenv("DOCKER_TLS_VERIFY"))

        //Workaround for docker cli commands hanging
        println "Initializing docker cli..."
        def initializeCmd = "${cliCommand} ps"
        def attempt = 1
        def ev = runCommand(initializeCmd, envp).exitValue
        while (attempt <= 10 && ev != 0) {
            ev = runCommand(initializeCmd).exitValue
            attempt++
        }

        def dUsername = System.getenv("DOCKER_USERNAME")
        def dPassword = System.getenv("DOCKER_PASSWORD")
        runCommand("docker login -u ${dUsername} -p ${dPassword}", envp)

        //fetch and store worker1 name
        def workerNameCmd = "${cliCommand} node ls -f role=worker --format {{.Hostname}}"
        workerName = runCommand(workerNameCmd, envp, false).standardOutput.split("\\r?\\n")[0]
    }
    DockerEE(String ns) {
        DockerEE()
    }

    def setup() {
        this.dockerEEDeployment = new DockerEEDeployment()
    }
    def cleanup() {
        this.deleteDeployment(this.dockerEEDeployment.getDeploymentName())
    }

    String getDeploymentName() {
        return dockerEEDeployment.getDeploymentName()
    }
    void setDeploymentName(String deploymentName) {
        dockerEEDeployment.setDeploymentName(deploymentName)
    }
    void addMetaLabels(String labelName, String labelValue) {
        dockerEEDeployment.addMetaLabels(labelName, labelValue)
    }
    void setReplicasNum(int replicasNum) {
        dockerEEDeployment.setReplicasNum(replicasNum)
    }
    void addTemplateLabels(String labelName, String labelValue) {
        dockerEEDeployment.addTemplateLabels(labelName, labelValue)
    }
    void setContainersImage(String containersImage) {
        dockerEEDeployment.setContainersImage(containersImage)
    }
    void setContainersName(String containersName) {
        dockerEEDeployment.setContainersName(containersName)
    }
    void addContainerPort(int port) {
        dockerEEDeployment.addContainerPort(port)
    }

    boolean createDeployment() {
        def dockerServiceName = dockerEEDeployment.getDeploymentName()
        String command = dockerEEDeployment.generateCommand()
        String cmd = "${cliCommand} service create ${command}"
        println(cmd)
        def exitCode = runCommand(cmd, envp).exitValue

        if (exitCode) {
            println "deployment failed..."
            return false
        }

        println "waiting for all containers to be in running state..."

        def replicasCmd = "${cliCommand} service inspect ${dockerServiceName} " +
                "--format {{.Spec.Mode.Replicated.Replicas}}"
        def replicas = runCommand(replicasCmd, envp).standardOutput.trim()
        def status = "0/${replicas}"
        def expectedStatus = "${replicas}/${replicas}"
        def runningCmd = "${cliCommand} service ls -f name=${dockerServiceName} --format {{.Replicas}}"
        while (status.trim() != expectedStatus) {
            status = runCommand(runningCmd, envp).standardOutput
        }
        println "Service created."

        return true
    }

    def deleteDeployment(String deploymentName) {
        println "Removing service ${deploymentName}"
        String cmd = "${cliCommand} service rm ${deploymentName}"
        runCommand(cmd, envp)

        def status = deploymentName
        println "Waiting for service to terminate..."
        cmd = "${cliCommand} service ls"
        while (status.contains(deploymentName)) {
            status = runCommand(cmd, envp).standardOutput
        }
        println "Service removed."
    }

    @Override
    def getPodOrContainerIds(String deploymentName, String namespace) {
        def cmd = "${cliCommand} ps --format {{.ID}} -f name=${deploymentName}"

        return runCommand(cmd, envp).standardOutput.split("\\r?\\n")
    }
}
