package objects

import common.Constants
import groovy.transform.AutoClone
import groovy.util.logging.Slf4j
import orchestratormanager.OrchestratorType

@AutoClone
@Slf4j
class Deployment {
    String name
    String namespace = Constants.ORCHESTRATOR_NAMESPACE
    String image
    Map<String, String> labels = [:]
    Map<Integer, String> ports = [:]
    Integer targetport
    List<Volume> volumes = []
    List<VolumeMount> volumeMounts = []
    Map<String, String> secretNames = [:]
    List<String> imagePullSecret = []
    Map<String,String> annotation = [:]
    List<String> command = []
    List<String> args = []
    Integer replicas = 1
    Map<String, String> env = [:]
    List<String> envFromSecrets = []
    List<String> envFromConfigMaps = []
    Map<String, SecretKeyRef> envValueFromSecretKeyRef = [:]
    Map<String, ConfigMapKeyRef> envValueFromConfigMapKeyRef = [:]
    Map<String, String> envValueFromFieldRef = [:]
    Map<String, String> envValueFromResourceFieldRef = [:]
    Boolean isPrivileged = false
    Boolean readOnlyRootFilesystem = false
    Map<String , String> limits = [:]
    Map<String , String> request = [:]
    Boolean hostNetwork = false
    List<String> addCapabilities = []
    List<String> dropCapabilities = []

    // Misc
    String loadBalancerIP = null
    String routeHost = null
    String deploymentUid
    List<Pod> pods = []
    String containerName = null
    Boolean skipReplicaWait = false
    Boolean exposeAsService = false
    Boolean createLoadBalancer = false
    Boolean createRoute = false
    Boolean automountServiceAccountToken = true
    Boolean livenessProbeDefined = false
    Boolean readinessProbeDefined = false
    String serviceName
    String serviceAccountName

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

    static final List<String> TEST_IMAGES_TO_IGNORE_FOR_RATE_LIMIT_CHECK = [
            "quay.io/rhacs-eng/qa-multi-arch-busybox:latest",
            "busybox",
            "quay.io/rhacs-eng/qa-multi-arch-nginx:latest",
            "nginx",
            "non-existent:image",
    ]

    Deployment setImage(String imageName) {
        // This is an imperfect check that images used in test are not
        // potentially subject to docker.io rate limiting and thus the cause of
        // test flakes. Imperfect because some tests rely on latest images in
        // particular to trigger the 'latest tag' policy and undoing that
        // reliance is a longer term project (ROX-10041).
        if (!TEST_IMAGES_TO_IGNORE_FOR_RATE_LIMIT_CHECK.contains(imageName) &&
                (imageName =~ /^docker.io.*/ ||
                 !(imageName =~ /^[a-z]+\./))) {
            String nameAsTag = imageName.replaceAll(~"[./:]", "-")
            log.warn """\
                WARNING: ${imageName} may be subject to rate limiting.
                Consider making a duplicate at quay.io/rhacs-eng/qa:${nameAsTag}
                e.g. (needs write access - ask @eng-staff)
                docker pull ${imageName}
                docker tag ${imageName} quay.io/rhacs-eng/qa:${nameAsTag}
                docker push quay.io/rhacs-eng/qa:${nameAsTag}
                """.stripIndent()
        }
        this.image = imageName
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

    Deployment addVolume(String name, String path, boolean enableHostPath = false, boolean readOnly = false) {
        this.volumes.add(new Volume(name: name,
                hostPath: enableHostPath,
                mountPath: path))
        this.volumeMounts.add(new VolumeMount(name: name,
                mountPath: path,
                readOnly: readOnly
        ))
        return this
    }

    Deployment addVolume(Volume v, boolean readOnly = false) {
        this.volumes.add(v)
        this.volumeMounts.add(new VolumeMount(
                mountPath: v.mountPath,
                name: v.name,
                readOnly: readOnly
        ))
        return this
    }

    Deployment addVolumeFromConfigMap(ConfigMap configMap, String mountPath) {
        String volumeName = "${configMap.name}-volume"
        this.volumes.add(new Volume(name: volumeName,
            configMap: configMap))
        this.volumeMounts.add(new VolumeMount(
            mountPath: mountPath,
            name: volumeName,
            readOnly: true,
        ))
        return this
    }

    Deployment addSecretName(String volumeName, String secretName) {
        this.secretNames.put(volumeName, secretName)
        return this
    }

    Deployment addVolumeFromSecret(Secret secret, String mountPath) {
        String volumeName = "${secret.name}-volume"
        this.volumes.add(new Volume(name: volumeName,
                secret: secret))
        this.volumeMounts.add(new VolumeMount(
                name: volumeName,
                mountPath: mountPath,
                readOnly: true,
        ))
        this.addSecretName(volumeName, secret.name)
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

    Deployment setEnvFromSecrets(List<String> envFromSecrets) {
        this.envFromSecrets = envFromSecrets
        return this
    }

    Deployment setEnvFromConfigMaps(List<String> envFromConfigMaps) {
        this.envFromConfigMaps = envFromConfigMaps
        return this
    }

    Deployment addEnvValueFromSecretKeyRef(String envName, SecretKeyRef secretKeyRef) {
        this.envValueFromSecretKeyRef.put(envName, secretKeyRef)
        return this
    }

    Deployment addEnvValueFromConfigMapKeyRef(String envName, ConfigMapKeyRef configMapKeyRef) {
        this.envValueFromConfigMapKeyRef.put(envName, configMapKeyRef)
        return this
    }

    Deployment addEnvValueFromFieldRef(String envName, String fieldPath) {
        this.envValueFromFieldRef.put(envName, fieldPath)
        return this
    }

    Deployment addEnvValueFromResourceFieldRef(String envName, String resource) {
        this.envValueFromResourceFieldRef.put(envName, resource)
        return this
    }

    Deployment setPrivilegedFlag(boolean val) {
        this.isPrivileged = val
        return this
    }

    Deployment setReadOnlyRootFilesystem(boolean val) {
        this.readOnlyRootFilesystem = val
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

    Deployment setServiceAccountName(String n) {
        this.serviceAccountName = n
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

    Deployment setContainerName(String containerName) {
        this.containerName = containerName
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

    Deployment setCreateRoute(Boolean create) {
        this.createRoute = create
        return this
    }

    Deployment setAutomountServiceAccountToken(Boolean automount) {
        this.automountServiceAccountToken = automount
        return this
    }

    Deployment setLivenessProbeDefined(Boolean probeDefined) {
        this.livenessProbeDefined = probeDefined
        return this
    }

    Deployment setReadinessProbeDefined(Boolean probeDefined) {
        this.readinessProbeDefined = probeDefined
        return this
    }

    Deployment setServiceName(String name) {
        this.serviceName = name
        return this
    }

    Deployment setCapabilities(List<String> add, List<String> drop) {
        this.addCapabilities = add
        this.dropCapabilities = drop
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

class Job extends Deployment {
    @Override
    Job create() {
        OrchestratorType.orchestrator.createJob(this)
        return this
    }

    @Override
    def delete() {
        OrchestratorType.orchestrator.deleteJob(this)
    }
}
