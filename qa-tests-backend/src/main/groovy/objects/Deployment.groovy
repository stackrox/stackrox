package objects

import common.Constants
import orchestratormanager.OrchestratorType

class Deployment {
    String name
    String namespace = Constants.ORCHESTRATOR_NAMESPACE
    String image
    Map<String, String> labels = [:]
    Map<Integer, String> ports = [:]
    Integer targetport
    Map<String, String> volumes = [:]
    Map<String, String> volumeMounts = [:]
    Map<String, String> secretNames = [:]
    List<String> imagePullSecret = []
    Map<String,String> annotation = [:]
    List<String> command = []
    List<String> args = []
    Integer replicas = 1
    Map<String, String> env = [:]
    Boolean isPrivileged = false
    Map<String , String> limits = [:]
    Map<String , String> request = [:]
    Boolean hostNetwork = false

    // Misc
    String loadBalancerIP = null
    String deploymentUid
    List<Pod> pods = []
    Boolean skipReplicaWait = false
    Boolean exposeAsService = false
    Boolean createLoadBalancer = false
    String serviceName

    Deployment setName(String n) {
        this.name = n
        // This label will be the selector used to select this deployment.
        this.addLabel("name", n)
        return this
    }

    Deployment setNamespace(String n) {
        this.namespace = n
        return this
    }

    Deployment setImage(String i) {
        this.image = i
        return this
    }

    Deployment addLabel(String k, String v) {
        this.labels[k] = v
        return this
    }

    Deployment addPort(Integer p, String protocol = "TCP") {
        this.ports.put(p, protocol)
        return this
    }

    Deployment setTargetPort(int port) {
        this.targetport = port
        return this
    }

    Deployment addVolume(String v, String p, boolean enableHostPath = false) {
        enableHostPath ? this.volumes.put(v, p) : this.volumes.put(v, null)
        this.volumeMounts.put(v, p)
        return this
    }

    Deployment addSecretName(String v, String s) {
        this.secretNames.put(v, s)
        return this
    }

    Deployment addImagePullSecret(String sec) {
        this.imagePullSecret.add(sec)
        return this
    }

    Deployment addAnnotation(String key, String val) {
        this.annotation[key] = val
        return this
    }

    Deployment setCommand(List<String> command) {
        this.command = command
        return this
    }

    Deployment setArgs(List<String> args) {
        this.args = args
        return this
    }

    Deployment setReplicas(Integer n) {
        this.replicas = n
        return this
    }

    Deployment setEnv(Map<String, String> env) {
        this.env = env
        return this
    }

    Deployment setPrivilegedFlag(boolean val) {
        this.isPrivileged = val
        return this
    }

    Deployment addLimits(String key, String val) {
        this.limits.put(key, val)
        return this
    }

    Deployment addRequest(String key, String val) {
        this.request.put(key, val)
        return this
    }

    Deployment setHostNetwork(boolean val) {
        this.hostNetwork = val
        return this
    }

    Deployment addPod(String podName, String podUid, List<String> containerIds, String podIP) {
        this.pods.add(
                new Pod(
                        name: podName,
                        namespace: this.namespace,
                        uid: podUid,
                        containerIds: containerIds,
                        podIP: podIP
                )
        )
        return this
    }

    Deployment setSkipReplicaWait(Boolean skip) {
        this.skipReplicaWait = skip
        return this
    }

    Deployment setExposeAsService(Boolean expose) {
        this.exposeAsService = expose
        return this
    }

    Deployment setCreateLoadBalancer(Boolean lb) {
        this.createLoadBalancer = lb
        return this
    }

    Deployment setServiceName(String name) {
        this.serviceName = name
        return this
    }

    Deployment create() {
        OrchestratorType.orchestrator.createDeployment(this)
        return this
    }

    def delete() {
        OrchestratorType.orchestrator.deleteDeployment(this)
    }
}

class DaemonSet extends Deployment {
    @Override
    DaemonSet create() {
        OrchestratorType.orchestrator.createDaemonSet(this)
        return this
    }

    @Override
    def delete() {
        OrchestratorType.orchestrator.deleteDaemonSet(this)
    }
}
