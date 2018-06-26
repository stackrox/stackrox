package objects

class DockerEEDeployment {
    String deploymentName
    int replicasNum
    String containersImage
    Map<String, String> metaLabels     // --labels
    Map<String, String> templateLabels //--container-label
    List<Integer> containersPorts

    DockerEEDeployment() {
        this.deploymentName = "qaTest"
        this.replicasNum = 1
        this.containersImage = "nginx:latest"
        this.metaLabels = new HashMap<>()
        this.templateLabels = new HashMap<>()
        this.containersPorts = new LinkedList<>()
    }

    String getDeploymentName() {
        return deploymentName
    }
    void setDeploymentName(String deploymentName) {
        this.deploymentName = deploymentName
    }
    void addMetaLabels(String labelName, String labelValue) {
        this.metaLabels.put(labelName, labelValue)
    }
    void setReplicasNum(int replicasNum) {
        this.replicasNum = replicasNum
    }
    void addTemplateLabels(String labelName, String labelValue) {
        this.templateLabels.put(labelName, labelValue)
    }
    void setContainersImage(String containersImage) {
        this.containersImage = containersImage
    }

    void addContainerPort(int port) {
        this.containersPorts.add(port)
    }

    private void appendLabels(StringBuilder command) {
        if (this.metaLabels == null || this.metaLabels.size() == 0) {
            return
        }
        command.append("--labels [")
        Iterator<Map.Entry<String, String>> metaLabelIt = metaLabels.entrySet().iterator()
        boolean firstElement = true
        while (metaLabelIt.hasNext()) {
            if (firstElement) {
                firstElement = false
            } else {
                command.append(",")
            }
            Map.Entry<String, String> pair = (Map.Entry<String, String>)metaLabelIt.next()
            command.append(pair.getKey()).append("=").append("\"" + pair.getValue() + "\"")
        }
        command.append("] ")
    }

    private void appendContainerLabels(StringBuilder command) {
        if (this.templateLabels == null || this.templateLabels.size() == 0) {
            return
        }
        command.append("--container-label [")
        Iterator<Map.Entry<String, String>> templateLabelIt = templateLabels.entrySet().iterator()
        boolean firstElement = true
        while (templateLabelIt.hasNext()) {
            if (firstElement) {
                firstElement = false
            } else {
                command.append(",")
            }
            Map.Entry<String, String> pair = (Map.Entry<String, String>)templateLabelIt.next()
            command.append(pair.getKey()).append("=").append("\"" + pair.getValue() + "\"")
        }
        command.append("] ")
    }

    private void appendContainersPorts(StringBuilder command) {
        if (this.containersPorts == null || this.containersPorts.size() == 0) {
            return
        }
        Iterator<Integer> portIt = containersPorts.iterator()
        while (portIt.hasNext()) {
            int portNum = portIt.next()
            command.append("--publish ").append(portNum).append(":").append(portNum).append(" ")
        }
    }

    String generateCommand() {
        StringBuilder command = new StringBuilder()

        command.append("--name ").append(this.getDeploymentName()).append(" ")
        command.append("--replicas ").append(this.getReplicasNum()).append(" ")

        this.appendLabels(command)
        this.appendContainerLabels(command)
        this.appendContainersPorts(command)

        command.append(this.containersImage)
        return command.toString()
    }
}
