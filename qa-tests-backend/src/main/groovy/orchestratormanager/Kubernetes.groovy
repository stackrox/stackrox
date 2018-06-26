package orchestratormanager

import objects.KubernetesDeployment

class Kubernetes extends OrchestratorCommon implements OrchestratorMain {
    private final String namespace
    private final String cliCommand = "kubectl"
    private KubernetesDeployment kubernetesDeployment

    Kubernetes(String ns) {
        //TODO: create secret for DTR
        namespace = ns
        def containerID = this.getPodOrContainerIds("central", "stackrox")
        this.portForward(containerID[0])
    }

    Kubernetes() {
        Kubernetes("default")
    }

    def portForward(String podId) {
        String cmd = "${cliCommand} -n stackrox port-forward ${podId} 8000:443 &"
        def result = runCommand(cmd).standardOutput
        System.out.println(result)
    }

    def ensureNamespaceExists(String namespace) {
        def getNamespaces = "${cliCommand} get namespaces -o jsonpath={.items[*].metadata.name}"
        def namespaces = runCommand(getNamespaces).standardOutput.split(" ")

        if (!namespaces.contains(namespace)) {
            def createNamespace = "${cliCommand} create namespace ${namespace}"
            runCommand(createNamespace)
            println "Created namespace ${namespace}"
        }
    }

    void setDeploymentName(String metaName) {
        kubernetesDeployment.setDeploymentName(metaName)
    }
    String getDeploymentName() {
        return kubernetesDeployment.getDeploymentName()
    }
    void addMetaLabels(String labelName, String labelValue) {
        kubernetesDeployment.addMetaLabels(labelName, labelValue)
    }
    void setReplicasNum(int replicasNum) {
        kubernetesDeployment.setReplicasNum(replicasNum)
    }
    void addTemplateLabels(String labelName, String labelValue) {
        kubernetesDeployment.addTemplateLabels(labelName, labelValue)
    }
    void setContainersImage(String containersImage) {
        kubernetesDeployment.setContainersImage(containersImage)
    }
    void setContainersName(String containersName) {
        kubernetesDeployment.setContainersName(containersName)
    }
    void addContainerPort(int port) {
        kubernetesDeployment.addContainerPort(port)
    }

    def setup() {
        kubernetesDeployment = new KubernetesDeployment()
        ensureNamespaceExists(namespace)
    }

    def cleanup() {
        this.deleteDeployment(this.kubernetesDeployment.getDeploymentName())
    }

    boolean createDeployment() {
        String ns = this.namespace
        String jsonDeployment = kubernetesDeployment.generateDeployment()
        def deploymentName = kubernetesDeployment.getDeploymentName()
        def fileSuffix = ".json"
        def exitCode = 1 // False: 1, True: 0
        try {
            def inputFile = File.createTempFile(deploymentName, fileSuffix)
            inputFile.write(jsonDeployment)
            String cmd = "${cliCommand} -n ${ns} create -f ${inputFile}"
            println(cmd)
            exitCode = runCommand(cmd, null, true).exitValue
            inputFile.deleteOnExit()
        } catch (IOException ioException) {
            println(ioException.toString())
        }
        if (exitCode) {
            println "deployment failed..."
            return false
        }
        println "waiting for all pods to be in running state..."

        def replicasCmd = "${cliCommand} -n ${ns} " +
                "get deploy --field-selector metadata.name=${deploymentName} -o jsonpath={.items[0].spec.replicas}"
        def replicas = runCommand(replicasCmd).standardOutput.trim()
        def status = "0"
        def runningCmd = "${cliCommand} -n ${ns} get deploy " +
                "--field-selector metadata.name=${deploymentName} -o jsonpath={.items[0].status.availableReplicas}"
        while (status.trim() != replicas) {
            status = runCommand(runningCmd).standardOutput
        }
        println "Deployment created."

        return true
    }

    def getPodOrContainerIds(String deploymentName, String ns = namespace) {
        List<String> podIds = new ArrayList<>()
        def cmd = "${cliCommand} -n ${ns} get pods --no-headers --output=custom-columns=NAME:.metadata.name"
        def podNames = runCommand(cmd).standardOutput.split("\\r?\\n")
        for (String podName : podNames) {
            if (podName.startsWith(deploymentName)) {
                podIds.add(podName)
            }
        }
        return podIds.toArray()
    }

    def deleteDeployment(String deploymentName) {
        println "Removing deployment ${deploymentName}"
        String ns = namespace
        String cmd = "${cliCommand} -n ${ns}  delete deployment ${deploymentName}"
        def exitCode = runCommand(cmd).exitValue
        while (exitCode != 0) {
            exitCode = runCommand(cmd).exitValue
        }

        def status = deploymentName
        println "Waiting for deployment to terminate"
        while (status.contains(deploymentName)) {
            def checkCmd = "${cliCommand} -n ${ns}  get deployments"
            status = runCommand(checkCmd).standardOutput
        }

        println "Waiting for pods to terminate"
        while (status.contains(deploymentName)) {
            def checkCmd = "${cliCommand} -n ${ns}  get pods"
            status = runCommand(checkCmd).standardOutput
        }

        println "Deployment removed."
    }

}
