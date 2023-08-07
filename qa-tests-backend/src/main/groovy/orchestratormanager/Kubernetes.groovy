package orchestratormanager

import static util.Helpers.evaluateWithRetry
import static util.Helpers.withK8sClientRetry
import static util.Helpers.withRetry

import java.nio.file.Paths
import java.util.concurrent.CompletableFuture
import java.util.concurrent.TimeUnit
import java.util.concurrent.TimeoutException

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j
import io.fabric8.kubernetes.api.model.Capabilities
import io.fabric8.kubernetes.api.model.ConfigMap as K8sConfigMap
import io.fabric8.kubernetes.api.model.ConfigMapEnvSource
import io.fabric8.kubernetes.api.model.ConfigMapKeySelectorBuilder
import io.fabric8.kubernetes.api.model.ConfigMapVolumeSource
import io.fabric8.kubernetes.api.model.Container
import io.fabric8.kubernetes.api.model.ContainerPort
import io.fabric8.kubernetes.api.model.ContainerStatus
import io.fabric8.kubernetes.api.model.EnvFromSource
import io.fabric8.kubernetes.api.model.EnvVar
import io.fabric8.kubernetes.api.model.EnvVarBuilder
import io.fabric8.kubernetes.api.model.EnvVarSourceBuilder
import io.fabric8.kubernetes.api.model.ExecAction
import io.fabric8.kubernetes.api.model.HostPathVolumeSource
import io.fabric8.kubernetes.api.model.IntOrString
import io.fabric8.kubernetes.api.model.LabelSelector
import io.fabric8.kubernetes.api.model.LocalObjectReference
import io.fabric8.kubernetes.api.model.Namespace
import io.fabric8.kubernetes.api.model.NamespaceBuilder
import io.fabric8.kubernetes.api.model.ObjectFieldSelectorBuilder
import io.fabric8.kubernetes.api.model.ObjectMeta
import io.fabric8.kubernetes.api.model.Pod
import io.fabric8.kubernetes.api.model.PodBuilder
import io.fabric8.kubernetes.api.model.PodList
import io.fabric8.kubernetes.api.model.PodSpec
import io.fabric8.kubernetes.api.model.PodTemplateSpec
import io.fabric8.kubernetes.api.model.Probe
import io.fabric8.kubernetes.api.model.Quantity
import io.fabric8.kubernetes.api.model.ResourceFieldSelectorBuilder
import io.fabric8.kubernetes.api.model.ResourceRequirements
import io.fabric8.kubernetes.api.model.Secret as K8sSecret
import io.fabric8.kubernetes.api.model.SecretEnvSource
import io.fabric8.kubernetes.api.model.SecretKeySelectorBuilder
import io.fabric8.kubernetes.api.model.SecretVolumeSource
import io.fabric8.kubernetes.api.model.SecurityContext
import io.fabric8.kubernetes.api.model.Service
import io.fabric8.kubernetes.api.model.ServiceAccount
import io.fabric8.kubernetes.api.model.ServiceBuilder
import io.fabric8.kubernetes.api.model.ServicePort
import io.fabric8.kubernetes.api.model.ServiceSpec
import io.fabric8.kubernetes.api.model.Status
import io.fabric8.kubernetes.api.model.Volume
import io.fabric8.kubernetes.api.model.VolumeMount
import io.fabric8.kubernetes.api.model.admissionregistration.v1.ValidatingWebhookConfiguration
import io.fabric8.kubernetes.api.model.apps.DaemonSet as K8sDaemonSet
import io.fabric8.kubernetes.api.model.apps.DaemonSetBuilder
import io.fabric8.kubernetes.api.model.apps.DaemonSetList
import io.fabric8.kubernetes.api.model.apps.DaemonSetSpec
import io.fabric8.kubernetes.api.model.apps.Deployment as K8sDeployment
import io.fabric8.kubernetes.api.model.apps.DeploymentBuilder
import io.fabric8.kubernetes.api.model.apps.DeploymentList
import io.fabric8.kubernetes.api.model.apps.DeploymentSpec
import io.fabric8.kubernetes.api.model.apps.StatefulSet as K8sStatefulSet
import io.fabric8.kubernetes.api.model.apps.StatefulSetList
import io.fabric8.kubernetes.api.model.batch.v1.Job as K8sJob
import io.fabric8.kubernetes.api.model.batch.v1.JobList
import io.fabric8.kubernetes.api.model.batch.v1.JobSpec
import io.fabric8.kubernetes.api.model.networking.v1.NetworkPolicyBuilder
import io.fabric8.kubernetes.api.model.networking.v1.NetworkPolicyEgressRuleBuilder
import io.fabric8.kubernetes.api.model.networking.v1.NetworkPolicyIngressRuleBuilder
import io.fabric8.kubernetes.api.model.networking.v1.NetworkPolicyPeerBuilder
import io.fabric8.kubernetes.api.model.policy.v1beta1.HostPortRange
import io.fabric8.kubernetes.api.model.policy.v1beta1.PodSecurityPolicy
import io.fabric8.kubernetes.api.model.policy.v1beta1.PodSecurityPolicyBuilder
import io.fabric8.kubernetes.api.model.rbac.ClusterRole
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding
import io.fabric8.kubernetes.api.model.rbac.PolicyRule
import io.fabric8.kubernetes.api.model.rbac.Role
import io.fabric8.kubernetes.api.model.rbac.RoleBinding
import io.fabric8.kubernetes.api.model.rbac.RoleRef
import io.fabric8.kubernetes.api.model.rbac.Subject
import io.fabric8.kubernetes.client.KubernetesClient
import io.fabric8.kubernetes.client.KubernetesClientBuilder
import io.fabric8.kubernetes.client.KubernetesClientException
import io.fabric8.kubernetes.client.dsl.Deletable
import io.fabric8.kubernetes.client.dsl.ExecListener
import io.fabric8.kubernetes.client.dsl.ExecWatch
import io.fabric8.kubernetes.client.dsl.MixedOperation
import io.fabric8.kubernetes.client.dsl.Resource
import io.fabric8.kubernetes.client.dsl.RollableScalableResource
import io.fabric8.kubernetes.client.dsl.ScalableResource
import org.apache.commons.exec.CommandLine

import common.YamlGenerator
import objects.ConfigMap
import objects.ConfigMapKeyRef
import objects.DaemonSet
import objects.Deployment
import objects.Job
import objects.K8sPolicyRule
import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.K8sSubject
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Node
import objects.Secret
import objects.SecretKeyRef
import util.Env
import util.Timer

@CompileStatic
@Slf4j
class Kubernetes implements OrchestratorMain {
    final int sleepDurationSeconds = 5
    final int maxWaitTimeSeconds = 90
    final int lbWaitTimeSeconds = 600
    final int intervalTime = 1
    String namespace
    KubernetesClient client

    MixedOperation<K8sDaemonSet, DaemonSetList, Resource<K8sDaemonSet>> daemonsets

    MixedOperation<K8sDeployment, DeploymentList,
            RollableScalableResource<K8sDeployment>> deployments

    MixedOperation<K8sStatefulSet, StatefulSetList,
            RollableScalableResource<K8sStatefulSet>> statefulsets

    MixedOperation<K8sJob, JobList,
            ScalableResource<K8sJob>> jobs

    Kubernetes(String ns) {
        this.namespace = ns
        this.client = new KubernetesClientBuilder().build()
        // On OpenShift, the namespace config is typically non-null (set to the default project), which causes all
        // "any namespace" requests to be scoped to the default project.
        this.client.configuration.namespace = null
        this.client.configuration.setRequestTimeout(32*1000)
        this.client.configuration.setConnectionTimeout(20*1000)
        this.deployments = this.client.apps().deployments()
        this.daemonsets = this.client.apps().daemonSets()
        this.statefulsets = this.client.apps().statefulSets()
        this.jobs = this.client.batch().v1().jobs()
    }

    Kubernetes() {
        this("default")
    }

    def ensureNamespaceExists(String ns) {
        Namespace namespace = newNamespace(ns)
        try {
            client.namespaces().create(namespace)
            log.info "Created namespace ${ns}"
            // defaultPspForNamespace(ns)
            provisionDefaultServiceAccount(ns)
        } catch (KubernetesClientException kce) {
            if (kce.code != 409) {
                throw kce
            }
            log.debug("Namespace ${ns} already exists")
        }
    }

    def setup() {
        ensureNamespaceExists(this.namespace)
    }

    def cleanup() {
    }

    /*
        Deployment Methods
    */

    def createDeployment(Deployment deployment) {
        ensureNamespaceExists(deployment.namespace)
        createDeploymentNoWait(deployment)
        waitForDeploymentAndPopulateInfo(deployment)
    }

    boolean updateDeploymentNoWait(Deployment deployment, int maxRetries=0) {
        K8sDeployment k8sdeployment = deployments.inNamespace(deployment.namespace).withName(deployment.name).get()
        if (k8sdeployment) {
            log.debug "Deployment ${deployment.name} with version ${k8sdeployment.metadata.resourceVersion} " +
                    "found in namespace ${deployment.namespace}. Updating..."
        } else {
            log.debug "Deployment ${deployment.name} NOT found in namespace ${deployment.namespace}. Creating..."
        }
        return createDeploymentNoWait(deployment, maxRetries)
    }

    def updateDeployment(Deployment deployment) {
        if (deployments.inNamespace(deployment.namespace).withName(deployment.name).get()) {
            log.debug "Deployment ${deployment.name} found in namespace ${deployment.namespace}. Updating..."
        } else {
            log.debug "Deployment ${deployment.name} NOT found in namespace ${deployment.namespace}. Creating..."
        }
        // Our createDeployment actually uses createOrReplace so it should work for these purposes
        return createDeployment(deployment)
    }

    def batchCreateDeployments(List<Deployment> deployments) {
        for (Deployment deployment : deployments) {
            ensureNamespaceExists(deployment.namespace)
            createDeploymentNoWait(deployment)
        }
        for (Deployment deployment : deployments) {
            waitForDeploymentAndPopulateInfo(deployment)
        }
    }

    def waitForAllPodsToBeRemoved(String ns, Map<String, String>labels, int retries = 30, int intervalSeconds = 5) {
        LabelSelector selector = new LabelSelector()
        selector.matchLabels = labels
        Timer t = new Timer(retries, intervalSeconds)
        PodList list
        while (t.IsValid()) {
            list = client.pods().inNamespace(ns).withLabelSelector(selector).list()
            if (list.items.size() == 0) {
                return true
            }
        }

        log.debug "Timed out waiting for the following pods to be removed"
        for (Pod pod : list.getItems()) {
            log.debug "\t- ${pod.metadata.name}"
        }
        return false
    }

    boolean podReady(Pod pod) {
        def deleted = pod.metadata.deletionTimestamp as boolean
        return !deleted && pod.status?.containerStatuses?.every { it.ready }
    }

    def waitForPodsReady(String ns, Map<String, String> labels, int minReady = 1, int retries = 30,
                         int intervalSeconds = 5) {
        LabelSelector selector = new LabelSelector()
        selector.matchLabels = labels
        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            def list = client.pods().inNamespace(ns).withLabelSelector(selector).list()
            def readyPods = list.items.findAll { Pod p -> podReady(p) }
            def readyPodNames = readyPods.collect { Pod p -> p.metadata.name }
            def numReady = readyPodNames.size()
            if (numReady >= minReady) {
                log.debug "Encountered ${numReady} ready pods matching ${labels}: ${readyPodNames}"
                return true
            }
        }

        log.debug "Timed out waiting for pods to become ready"
        return false
    }

    // waitForPodRestart waits until the restartCount is greater then the given prevRestartCount
    def waitForPodRestart(String ns, String name, int prevRestartCount, int retries, int intervalSeconds) {
        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            def pod = client.pods().inNamespace(ns).withName(name).get()
            if (pod.status.containerStatuses.get(0).restartCount > prevRestartCount) {
                log.debug "Restarted container ${ns}/${name}"
                return true
            }
        }
        throw new OrchestratorManagerException("Timed out waiting killing pod ${ns}/${name}")
    }

    List<Pod> getPods(String ns, String appName) {
        return getPodsByLabel(ns, ["app": appName])
    }

    List<Pod> getPodsByLabel(String ns, Map<String, String> label) {
        def selector = new LabelSelector()
        selector.matchLabels = label
        PodList list = evaluateWithRetry(2, 3) {
            return client.pods().inNamespace(ns).withLabelSelector(selector).list()
        }
        return list.getItems()
    }

    Boolean deletePod(String ns, String podName, Long gracePeriodSecs) {
        Deletable podClient = client.pods().inNamespace(ns).withName(podName)
        if (gracePeriodSecs != null) {
            podClient = podClient.withGracePeriod(gracePeriodSecs)
        }
        return podClient.delete()
    }

    def deleteAllPods(String ns, Map<String, String> labels) {
        log.debug "Delete all pods in ${ns} with labels ${labels}"
        client.pods().inNamespace(ns).withLabels(labels).delete()
    }

    void deleteAllPodsAndWait(String ns, Map<String, String> labels) {
        log.debug "Will delete all pods in ${ns} with labels ${labels} and wait for deletion"

        List<Pod> beforePods = evaluateWithRetry(2, 3) {
            client.pods().inNamespace(ns).withLabels(labels).list().getItems()
        }
        beforePods.each { pod ->
            evaluateWithRetry(2, 3) {
                client.pods().inNamespace(ns).withName(pod.metadata.name).delete()
            }
        }

        Timer t = new Timer(30, 5)
        Boolean allDeleted = false
        while (!allDeleted && t.IsValid()) {
            allDeleted = true
            beforePods.each { deleted ->
                Pod pod = evaluateWithRetry(2, 3) {
                    client.pods().inNamespace(ns).withName(deleted.metadata.name).get()
                }
                if (pod == null) {
                    log.debug "${deleted.metadata.name} is deleted"
                }
                else {
                    log.debug "${deleted.metadata.name} is not deleted"
                    allDeleted = false
                }
            }
        }
        if (!allDeleted) {
            throw new OrchestratorManagerException("Gave up trying to delete all pods")
        }
    }

    Boolean deletePodAndWait(String ns, String name, int retries, int intervalSeconds) {
        deletePod(ns, name, null)
        log.debug "Deleting pod ${name}"

        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            log.debug "Waiting for pod deletion ${name}"
            def pod = client.pods().inNamespace(ns).withName(name).get()
            if (pod == null) {
                log.debug "Deleted pod ${name}"
                return true
            }
        }
        throw new OrchestratorManagerException("Could not delete pod ${ns}/${name}")
    }

    Boolean restartPodByLabelWithExecKill(String ns, Map<String, String> labels) {
        Pod pod = getPodsByLabel(ns, labels).get(0)
        int prevRestartCount = pod.status.containerStatuses.get(0).restartCount
        def cmds = ["sh", "-c", "kill -15 1"] as String[]
        execInContainerByPodName(pod.metadata.name, pod.metadata.namespace, cmds)
        log.debug "Killed pod ${pod.metadata.name}"
        return waitForPodRestart(pod.metadata.namespace, pod.metadata.name, prevRestartCount, 25, 5)
    }

    def restartPodByLabels(String ns, Map<String, String> labels, int retries, int intervalSecond) {
        Pod pod = getPodsByLabel(ns, labels).get(0)

        deletePodAndWait(ns, pod.metadata.name, retries, intervalSecond)
        return waitForPodsReady(ns, labels, 1, retries, intervalSecond)
    }

    def getAndPrintPods(String ns, String name) {
        log.debug "Status of ${name}'s pods:"
        for (Pod pod : getPodsByLabel(ns, ["deployment": name])) {
            log.debug "\t- ${pod.metadata.name}"
            for (ContainerStatus status : pod.status.containerStatuses) {
                log.debug "\t  Container status: ${status.state}"
            }
        }
    }

    String getPodLog(String ns, String name) {
        log.debug "reading logs from ${ns}/${name}"
        return client.pods().inNamespace(ns)
                .withName(name).getLog()
    }

    def copyFileToPod(String fromPath, String ns, String podName, String toPath) {
        client.pods()
                .inNamespace(ns)
                .withName(podName)
                .file(toPath)
                .upload(Paths.get(fromPath))
    }

    def addPodAnnotationByApp(String ns, String appName, String key, String value) {
        Pod pod = getPodsByLabel(ns, ["app": appName]).get(0)
        client.pods().inNamespace(ns).withName(pod.metadata.name).edit {
            n -> new PodBuilder(n).editMetadata().addToAnnotations(key, value).endMetadata().build()
        }
    }

    def waitForDeploymentDeletion(Deployment deploy, int retries = 30, int intervalSeconds = 5) {
        Timer t = new Timer(retries, intervalSeconds)

        K8sDeployment d
        while (t.IsValid()) {
            d = this.deployments.inNamespace(deploy.namespace).withName(deploy.name).get()
            if (d == null) {
                log.debug "${deploy.name}: deployment removed."
                return
            }
            getAndPrintPods(deploy.namespace, deploy.name)
        }
        log.debug "Timed out waiting for deployment ${deploy.name} to be deleted"
    }

    def deleteAndWaitForDeploymentDeletion(Deployment... deployments) {
        for (Deployment deployment : deployments) {
            this.deleteDeployment(deployment)
        }
        for (Deployment deployment : deployments) {
            this.waitForDeploymentDeletion(deployment)
        }
    }

    def deleteDeployment(Deployment deployment) {
        if (deployment.exposeAsService) {
            this.deleteService(deployment.name, deployment.namespace)
        }
        if (deployment.createRoute) {
            this.deleteRoute(deployment.name, deployment.namespace)
        }
        // Retry deletion due to race condition in sdk and controller
        // See https://github.com/fabric8io/kubernetes-client/issues/1477
        Boolean deleted = false
        Timer t = new Timer(10, 1)
        while (t.IsValid()) {
            try {
                this.deployments.inNamespace(deployment.namespace).withName(deployment.name).delete()
                deleted = true
                break
            } catch (KubernetesClientException ex) {
                log.warn("Failed to delete deployment:", ex)
            }
        }
        if (deleted) {
            log.debug "Removed the deployment: ${deployment.name}"
        }
        else {
            log.warn "Failed to deleted the deployment: ${deployment.name} after repeated attempts"
        }
    }

    def createOrchestratorDeployment(K8sDeployment dep) {
        dep.setApiVersion("")
        dep.metadata.setResourceVersion("")
        return this.deployments.inNamespace(dep.metadata.namespace).create(dep)
    }

    K8sDeployment getOrchestratorDeployment(String ns, String name) {
        return this.deployments.inNamespace(ns).withName(name).get()
    }

    String getDeploymentId(Deployment deployment) {
        return this.deployments.inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()?.metadata?.uid
    }

    def getDeploymentReplicaCount(Deployment deployment) {
        K8sDeployment d = this.deployments.inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            log.debug "${deployment.name}: Replicas=${d.getSpec().getReplicas()}"
            return d.getSpec().getReplicas()
        }
    }

    def getDeploymentUnavailableReplicaCount(Deployment deployment) {
        K8sDeployment d = this.deployments
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            log.debug "${deployment.name}: Unavailable Replicas=${d.getStatus().getUnavailableReplicas()}"
            return d.getStatus().getUnavailableReplicas()
        }
    }

    def getDeploymentNodeSelectors(Deployment deployment) {
        K8sDeployment d = this.deployments
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            log.debug "${deployment.name}: Host=${d.getSpec().getTemplate().getSpec().getNodeSelector()}"
            return d.getSpec().getTemplate().getSpec().getNodeSelector()
        }
    }

    Set<String> getDeploymentSecrets(Deployment deployment) {
        K8sDeployment d = this.deployments
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        Set<String> secretSet = [] as Set
        if (d != null) {
            d.getSpec()?.getTemplate()?.getSpec()?.getVolumes()?.each { secretSet.add(it.secret.secretName) }
            d.getSpec()?.getTemplate()?.getSpec()?.getContainers()?.getAt(0)?.getEnvFrom()?.each {
                // Only care about secrets for now.
                if (it.getSecretRef() != null) {
                    secretSet.add(it.secretRef.name)
                }
            }
        }
        return secretSet
    }

    List<String> getDeploymentCount() {
        return this.deployments.list().getItems().collect { it.metadata.name }
    }

    List<String> getDeploymentCount(String ns) {
        return this.deployments.inNamespace(ns).list().getItems().collect { it.metadata.name }
    }

    List<String> getDeployments(String ns) {
        return evaluateWithRetry(2, 3) {
            this.deployments.inNamespace(ns).list().getItems().collect { it.metadata.name }
        }
    }

    def createPortForward(int port, Deployment deployment, String podName = "") {
        if (deployment.pods.size() == 0) {
            throw new KubernetesClientException(
                    "Error creating port-forward: Could not get pod details from deployment.")
        }
        if (deployment.pods.size() > 1 && podName == "") {
            throw new KubernetesClientException(
                    "Error creating port-forward: Deployment contains more than 1 pod, but no pod was specified.")
        }
        return deployment.pods.size() == 1 ?
                this.client.pods()
                        .inNamespace(deployment.namespace)
                        .withName(deployment.pods.get(0).name)
                        .portForward(port) :
                this.client.pods()
                        .inNamespace(deployment.namespace)
                        .withName(podName)
                        .portForward(port)
    }

    EnvVar getDeploymentEnv(String ns, String name, String key) {
        def deployment = client.apps().deployments().inNamespace(ns).withName(name).get()
        if (deployment == null) {
            throw new OrchestratorManagerException("Did not find deployment ${ns}/${name}")
        }

        List<EnvVar> envVars = client.apps().deployments().inNamespace(ns).withName(name).get().spec.template
                .spec.containers.get(0).env
        int index = envVars.findIndexOf { EnvVar it -> it.name == key }
        if (index < 0) {
            throw new OrchestratorManagerException("Did not find env variable ${key} in ${ns}/${name}")
        }
        return envVars.get(index)
    }

    def updateDeploymentEnv(String ns, String name, String key, String value) {
        log.debug "Update env var in ${ns}/${name}: ${key} = ${value}"
        List<EnvVar> envVars = client.apps().deployments().inNamespace(ns).withName(name).get().spec.template
                .spec.containers.get(0).env

        int index = envVars.findIndexOf { EnvVar it -> it.name == key }
        if (index > -1) {
            log.debug "Env var ${key} found on index: ${index}"
            envVars.get(index).value = value
        }
        else {
            log.debug "Env var ${key} not found. Adding it now"
            envVars.add(new EnvVarBuilder().withName(key).withValue(value).build())
        }

        withRetry(2, 3) {
            client.apps().deployments().inNamespace(ns).withName(name)
                .edit { d -> new DeploymentBuilder(d)
                    .editSpec()
                    .editTemplate()
                    .editSpec()
                    .editContainer(0)
                    .withEnv(envVars)
                    .endContainer()
                    .endSpec()
                    .endTemplate()
                    .endSpec()
                .build() }
        }
    }

    def scaleDeployment(String ns, String name, Integer replicas) {
        Exception mostRecentException
        Timer t = new Timer(30, 5)
        while (t.IsValid()) {
            try {
                client.apps().deployments().inNamespace(ns).withName(name).scale(replicas)
                mostRecentException = null
                break
            } catch (Exception e) {
                log.warn("Failed to scale the deployment", e)
                mostRecentException = e
            }
        }
        if (mostRecentException) {
            log.warn("Giving up trying to scale the deployment ${name} to ${replicas}")
            throw mostRecentException
        }
        else {
            log.info("Scaled the deployment ${name} to ${replicas}")
        }
    }

    /*
        DaemonSet Methods
    */

    def createDaemonSet(DaemonSet daemonSet) {
        ensureNamespaceExists(daemonSet.namespace)
        createDaemonSetNoWait(daemonSet)
        waitForDaemonSetAndPopulateInfo(daemonSet)
    }

    def deleteDaemonSet(DaemonSet daemonSet) {
        this.daemonsets.inNamespace(daemonSet.namespace).withName(daemonSet.name).delete()
        log.debug "${daemonSet.name}: daemonset removed."
    }

    boolean containsDaemonSetContainer(String ns, String name, String containerName) {
        return client.apps().daemonSets().inNamespace(ns).withName(name).get().spec.template
            .spec.containers.findIndexOf { it.name == containerName } > -1
    }

    def updateDaemonSetEnv(String ns, String name, String containerName, String key, String value) {
        log.debug "Update env var in ${ns}/${name}/${containerName}: ${key} = ${value}"
        List<Container> containers = client.apps().daemonSets().inNamespace(ns).withName(name).get().spec.template
            .spec.containers
        int containerIndex = containers.findIndexOf { it.name == containerName }
        if (containerIndex == -1) {
            throw new RuntimeException("Could not update env var. No container named ${containerName} in ${ns}/${name}")
        }
        log.debug "Container ${ns}/${name}/${containerName} found on index: ${containerIndex}"
        List<EnvVar> envVars = containers.get(containerIndex).env
        log.debug "Current env vars of ${ns}/${name}/${containerName}: ${envVars}"

        int index = envVars.findIndexOf { EnvVar it -> it.name == key }
        if (index > -1) {
            log.debug "Env var ${key} found on index: ${index}"
            envVars.get(index).value = value
        }
        else {
            log.debug "Env var ${key} not found. Adding it now"
            envVars.add(new EnvVarBuilder().withName(key).withValue(value).build())
        }

        client.apps().daemonSets().inNamespace(ns).withName(name)
            .edit { d -> new DaemonSetBuilder(d)
                .editSpec()
                .editTemplate()
                .editSpec()
                .editContainer(containerIndex)
                .withEnv(envVars)
                .endContainer()
                .endSpec()
                .endTemplate()
                .endSpec()
                .build() }
    }

    boolean deploymentReady(String ns, String name) {
        def depl = client.apps().deployments().inNamespace(ns).withName(name).get()
        if (depl == null) {
            return false
        }
        return depl.status.readyReplicas > 0
    }

    boolean daemonSetReady(String ns, String name) {
        def daemonSet = client.apps().daemonSets().inNamespace(ns).withName(name).get()
        return daemonSet.status.numberReady >= daemonSet.status.desiredNumberScheduled
    }

    // daemonSetEnvVarUpdated returns true if all pods are ready and the env var has a given value for all pods
    boolean daemonSetEnvVarUpdated(String ns, String name, String containerName,
                                   String envVarName, String envVarValue) {
        def pods = client.pods().inNamespace(ns).withLabel("app", name).list().getItems()
        int podsPassing = 0
        for (Pod pod : pods) {
            log.debug "Found pod \"${pod.getMetadata().name}\" with ${pod.getSpec().containers.size()} containers"
            int containerIndex = pod.getSpec().containers.findIndexOf { it.name == containerName }
            if (containerIndex == -1) {
                log.debug "Pod ${pod.getMetadata().name}: could not find container ${containerName}"
                return false
            }
            List<EnvVar> envVars = pod.getSpec().containers.get(containerIndex).env
            int index = envVars.findIndexOf { EnvVar it -> it.name == envVarName }
            if (index == -1) {
                log.debug "Pod ${pod.getMetadata().name}: " +
                    "could not find env variable ${envVarName} in container ${containerName}"
                return false
            }
            def value = envVars.get(index).value
            log.debug "Pod ${pod.getMetadata().name}: " +
                "Env var ${envVarName} found on index: ${index} with value ${value}"
            if (value != envVarValue) {
                log.debug "Pod ${pod.getMetadata().name}: " +
                    "Expected value ${envVarValue} does not match current ${value}"
                return false
            }
            log.debug "Pod ${pod.getMetadata().name}: All conditions have been met"
            podsPassing++
        }
        return podsPassing == pods.size()
    }

    def createJob(Job job) {
        ensureNamespaceExists(job.namespace)

        job.getNamespace() != null ?: job.setNamespace(this.namespace)

        K8sJob k8sJob = new K8sJob(
                metadata: new ObjectMeta(
                        name: job.name,
                        namespace: job.namespace,
                        labels: job.labels
                ),
                spec: new JobSpec(
                        template: new PodTemplateSpec(
                                metadata: new ObjectMeta(
                                        name: job.name,
                                        namespace: job.namespace,
                                        labels: job.labels
                                ),
                                spec: generatePodSpec(job)
                        )
                )
        )
        // Jobs cannot be Always
        k8sJob.spec.template.spec.restartPolicy = "Never"

        try {
            log.debug "Told the orchestrator to create job " + job.getName()
            return this.jobs.inNamespace(job.namespace).createOrReplace(k8sJob)
        } catch (Exception e) {
            log.warn("Error creating k8s job", e)
        }
        return null
    }

    def deleteJob(Job job) {
        this.jobs.inNamespace(job.namespace).withName(job.name).delete()
        log.debug "${job.name}: job removed."
    }

    def waitForDaemonSetDeletion(String name, String ns = namespace) {
        Timer t = new Timer(30, 5)

        while (t.IsValid()) {
            if (this.daemonsets.inNamespace(ns).withName(name).get() == null) {
                log.debug "Daemonset ${name} has been deleted"
                return
            }
        }
        log.debug "Timed out waiting for daemonset ${name} to stop"
    }

    def getDaemonSetReplicaCount(DaemonSet daemonSet) {
        K8sDaemonSet d = this.daemonsets
                .inNamespace(daemonSet.namespace)
                .withName(daemonSet.name)
                .get()
        if (d != null) {
            log.debug "${daemonSet.name}: Replicas=${d.getStatus().getDesiredNumberScheduled()}"
            return d.getStatus().getDesiredNumberScheduled()
        }
        return null
    }

    def getDaemonSetUnavailableReplicaCount(DaemonSet daemonSet) {
        K8sDaemonSet d = this.daemonsets
                .inNamespace(daemonSet.namespace)
                .withName(daemonSet.name)
                .get()
        if (d != null) {
            log.debug "${daemonSet.name}: Unavailable Replicas=${d.getStatus().getNumberUnavailable()}"
            return d.getStatus().getNumberUnavailable() == null ? 0 : d.getStatus().getNumberUnavailable()
        }
        return null
    }

    def getDaemonSetNodeSelectors(DaemonSet daemonSet) {
        K8sDaemonSet d = this.daemonsets
                .inNamespace(daemonSet.namespace)
                .withName(daemonSet.name)
                .get()
        if (d != null) {
            log.debug "${daemonSet.name}: Host=${d.getSpec().getTemplate().getSpec().getNodeSelector()}"
            return d.getSpec().getTemplate().getSpec().getNodeSelector()
        }
        return null
    }

    List<String> getDaemonSetCount(String ns) {
        return this.daemonsets.inNamespace(ns).list().getItems().collect { it.metadata.name }
    }

    List<String> getDaemonSetCount() {
        return this.daemonsets.list().getItems().collect { it.metadata.name }
    }

    String getDaemonSetId(DaemonSet daemonSet) {
        return this.daemonsets.inNamespace(daemonSet.namespace)
                .withName(daemonSet.name)
                .get()?.metadata?.uid
    }

    /*
        StatefulSet Methods
    */

    List<String> getStatefulSetCount() {
        return this.statefulsets.list().getItems().collect { it.metadata.name }
    }

    List<String> getStatefulSetCount(String ns) {
        return this.statefulsets.inNamespace(ns).list().getItems().collect { it.metadata.name }
    }

    /*
        Container Methods
    */

    def deleteContainer(String containerName, String namespace = this.namespace) {
        withRetry(2, 3) {
            client.pods().inNamespace(namespace).withName(containerName).delete()
        }
    }

    def wasContainerKilled(String containerName, String namespace = this.namespace) {
        Timer t = new Timer(20, 3)

        Pod pod
        while (t.IsValid()) {
            try {
                pod = client.pods().inNamespace(namespace).withName(containerName).get()
                if (pod == null) {
                    log.debug "Could not query K8S for pod details, assuming pod was killed"
                    return true
                }
                log.debug "Pod Deletion Timestamp: ${pod.metadata.deletionTimestamp}"
                if (pod.metadata.deletionTimestamp != null ) {
                    return true
                }
            } catch (Exception e) {
                log.warn("wasContainerKilled: error fetching pod details - retrying", e)
            }
        }
        log.warn "wasContainerKilled: did not determine container was killed before 60s timeout"
        log.warn "container details were found:\n${containerName}: ${pod}"
        return false
    }

    def isKubeDashboardRunning() {
        return evaluateWithRetry(2, 3) {
            PodList pods = client.pods().inAnyNamespace().list()
            List<Pod> kubeDashboards = pods.getItems().findAll {
                it.getSpec().getContainers().find {
                    it.getImage().contains("kubernetes-dashboard")
                }
            }
            return kubeDashboards.size() > 0
        }
    }

    String getContainerlogs(String ns, String podName, String containerName) {
        return client.pods().inNamespace(ns).withName(podName).inContainer(containerName).getLog()
    }

    Set<String> getStaticPodCount(String ns = null) {
        return evaluateWithRetry(2, 3) {
            // This method assumes that a static pod name will contain the node name that the pod is running on
            def nodeNames = client.nodes().list().items.collect { it.metadata.name }
            Set<String> staticPods = [] as Set
            PodList podList = ns == null ? client.pods().list() : client.pods().inNamespace(ns).list()
            podList.items.each {
                for (String node : nodeNames) {
                    if (it.metadata.name.contains(node)) {
                        staticPods.add(it.metadata.name[0..it.metadata.name.indexOf(node) - 2])
                    }
                }
            }
            return staticPods
        }
    }

    /*
        Service Methods
    */

    def createService(Deployment deployment) {
        withRetry(2, 3) {
            Service service = new Service(
                    metadata: new ObjectMeta(
                            name: deployment.serviceName ? deployment.serviceName : deployment.name,
                            namespace: deployment.namespace,
                            labels: deployment.labels
                    ),
                    spec: new ServiceSpec(
                            ports: deployment.getPorts().collect {
                                k, v ->
                                    new ServicePort(
                                            name: k as String,
                                            port: k as Integer,
                                            protocol: v,
                                            targetPort: new IntOrString(deployment.targetport) ?:
                                                    new IntOrString(k as Integer)
                                    )
                            },
                            selector: deployment.labels,
                            type: deployment.createLoadBalancer ? "LoadBalancer" : "ClusterIP"
                    )
            )
            def created = client.services().inNamespace(deployment.namespace).createOrReplace(service)
            if (created == null) {
                log.debug deployment.serviceName ?: deployment.name + " service not created"
                assert created
            }
            log.debug deployment.serviceName ?: deployment.name + " service created"
            if (deployment.createLoadBalancer) {
                deployment.loadBalancerIP = waitForLoadBalancer(deployment.serviceName ?:
                        deployment.name, deployment.namespace)
            }
        }
    }

    def createService(objects.Service s) {
        withRetry(2, 3) {
            Service service = new Service(
                    metadata: new ObjectMeta(
                            name: s.name,
                            namespace: s.namespace,
                            labels: s.labels
                    ),
                    spec: new ServiceSpec(
                            ports: s.getPorts().collect {
                                k, v ->
                                    new ServicePort(
                                            name: k as String,
                                            port: k as Integer,
                                            protocol: v,
                                            targetPort:
                                                    new IntOrString(s.targetport) ?: new IntOrString(k as Integer)
                                    )
                            },
                            selector: s.labels,
                            type: s.type.toString()
                    )
            )
            client.services().inNamespace(s.namespace).createOrReplace(service)
        }
        log.debug "${s.name}: Service created"
        if (objects.Service.Type.LOADBALANCER == s.type) {
            s.loadBalancerIP = waitForLoadBalancer(s.name, s.namespace)
        }
    }

    def getServiceIP(String serviceName, String ns) {
        return client.services()
                .inNamespace(ns)
                .withName(serviceName)
                .get()
                .getSpec()
                .getClusterIP()
    }

    def deleteService(String name, String namespace = this.namespace) {
        withRetry(2, 3) {
            log.debug "${name}: Service deleting..."
            client.services().inNamespace(namespace).withName(name).delete()
        }
        log.debug "${name}: Service deleted"
    }

    def waitForServiceDeletion(objects.Service service) {
        boolean beenDeleted = false

        int retries = (maxWaitTimeSeconds / sleepDurationSeconds).intValue()
        Timer t = new Timer(retries, sleepDurationSeconds)
        while (!beenDeleted && t.IsValid()) {
            Service s = client.services().inNamespace(service.namespace).withName(service.name).get()
            beenDeleted = true

            log.debug "Waiting for service ${service.name} to be deleted"
            if (s != null) {
                beenDeleted = false
            }
        }

        if (beenDeleted) {
            log.debug service.name + ": service removed."
        } else {
            log.debug "Timed out waiting for service ${service.name} to be removed"
        }
    }

    def addOrUpdateServiceLabel(String serviceName, String ns, String name, String value) {
        Map<String, String> label = [:]
        label.put(name, value)
        evaluateWithRetry(2, 3) {
            client.services().inNamespace(ns).withName(serviceName).edit {
                s ->
                    new ServiceBuilder(s).editMetadata().addToLabels(label).endMetadata().build()
            }
        }
    }

    def waitForLoadBalancer(Deployment deployment) {
        "Creating a load balancer"
        if (deployment.createLoadBalancer) {
            deployment.loadBalancerIP = waitForLoadBalancer(deployment.serviceName ?:
                                        deployment.name, deployment.namespace)
        }
    }

    /**
     * This is an overloaded method for creating load balancer for a given service or deployment
     *
     * @param service
     */
    String waitForLoadBalancer(String serviceName, String namespace) {
        Service service
        String loadBalancerIP
        int iterations = (lbWaitTimeSeconds / intervalTime).intValue()
        log.debug "Waiting for LB external IP for " + serviceName
        Timer t = new Timer(iterations, intervalTime)
        while (t.IsValid()) {
            service = client.services().inNamespace(namespace).withName(serviceName).get()
            if (service?.status?.loadBalancer?.ingress?.size()) {
                loadBalancerIP = service.status.loadBalancer.ingress.get(0).
                                  ip ?: service.status.loadBalancer.ingress.get(0).hostname
                log.debug "LB IP: " + loadBalancerIP
                break
            }
        }
        if (loadBalancerIP == null) {
            log.debug "Could not get loadBalancer IP in ${t.SecondsSince()} seconds and ${iterations} iterations"
        }
        return loadBalancerIP
    }

    /*
        Route Methods
    */

    def createRoute(String routeName, String namespace) {
        throw new RuntimeException("K8s does not support routes")
    }

    def deleteRoute(String routeName, String namespace) {
        throw new RuntimeException("K8s does not support routes")
    }

    String waitForRouteHost(String serviceName, String namespace) {
        throw new RuntimeException("K8s does not support routes")
    }

    /*
        Secrets Methods
    */
    K8sSecret waitForSecretCreation(String secretName, String namespace = this.namespace) {
        int retries = (maxWaitTimeSeconds / sleepDurationSeconds).intValue()
        Timer t = new Timer(retries, sleepDurationSeconds)
        while (t.IsValid()) {
            K8sSecret secret = client.secrets().inNamespace(namespace).withName(secretName).get()
            if (secret != null) {
                log.debug secretName + ": secret created."
                return secret
            }
        }
        log.debug "Timed out waiting for secret ${secretName} to be created"
        return null
    }

    String createImagePullSecret(String name, String username, String password,
                                 String namespace, String server) {
        return createImagePullSecret(new Secret(
            name: name,
            server: server,
            username: username,
            password: password,
            namespace: namespace
        ))
    }

    String createImagePullSecret(Secret secret) {
        def namespace = secret.namespace ?: this.namespace

        def auth = secret.username + ":" + secret.password
        def b64Password = Base64.getEncoder().encodeToString(auth.getBytes())
        def dockerConfigJSON =  "{\"auths\":{\"" + secret.server + "\": {\"auth\": \"" + b64Password + "\"}}}"
        Map<String, String> data = new HashMap<String, String>()
        data.put(".dockerconfigjson", Base64.getEncoder().encodeToString(dockerConfigJSON.getBytes()))

        K8sSecret k8sSecret = new K8sSecret(
                apiVersion: "v1",
                kind: "Secret",
                type: "kubernetes.io/dockerconfigjson",
                data: data,
                metadata: new ObjectMeta(
                        name: secret.name,
                        namespace: namespace
                )
        )

        K8sSecret createdSecret = client.secrets().inNamespace(namespace).createOrReplace(k8sSecret)
        if (createdSecret != null) {
            createdSecret = waitForSecretCreation(secret.name, namespace)
            return createdSecret.metadata.uid
        }
        throw new RuntimeException("Couldn't create secret")
    }

    String createSecret(Secret secret) {
        K8sSecret k8sSecret = new K8sSecret(
                apiVersion: "v1",
                kind: "Secret",
                data: secret.data,
                type: secret.type,
                metadata: new ObjectMeta(
                        name: secret.name,
                )
        )

        def sec = client.secrets().inNamespace(secret.namespace).createOrReplace(k8sSecret)
        log.debug secret.name + ": Secret created."
        return sec.metadata.uid
    }

    String createSecret(String name, String namespace = this.namespace) {
        return evaluateWithRetry(2, 3) {
            Map<String, String> data = new HashMap<String, String>()
            data.put("username", "YWRtaW4=")
            data.put("password", "MWYyZDFlMmU2N2Rm")

            K8sSecret secret = new K8sSecret(
                    apiVersion: "v1",
                    kind: "Secret",
                    type: "Opaque",
                    data: data,
                    metadata: new ObjectMeta(
                            name: name
                    )
            )

            try {
                K8sSecret createdSecret = client.secrets().inNamespace(namespace).createOrReplace(secret)
                if (createdSecret != null) {
                    createdSecret = waitForSecretCreation(name, namespace)
                    return createdSecret.metadata.uid
                }
            } catch (Exception e) {
                log.warn("Error creating secret", e)
            }
            return null
        }
    }

    def updateSecret(K8sSecret secret) {
        withRetry(2, 3) {
            client.secrets().inNamespace(secret.metadata.namespace).createOrReplace(secret)
        }
    }

    def deleteSecret(String name, String namespace = this.namespace) {
        withRetry(2, 3) {
            client.secrets().inNamespace(namespace).withName(name).delete()
        }
        sleep(sleepDurationSeconds * 1000)
        log.debug name + ": Secret removed."
    }

    int getSecretCount(String ns) {
        return evaluateWithRetry(2, 3) {
            return client.secrets().inNamespace(ns).list().getItems().findAll {
                !it.type.startsWith("kubernetes.io/service-account-token")
            }.size()
        }
    }

    int getSecretCount() {
        return evaluateWithRetry(2, 3) {
            return client.secrets().list().getItems().findAll {
                !it.type.startsWith("kubernetes.io/service-account-token")
            }.size()
        }
    }

    K8sSecret getSecret(String name, String namespace) {
        return evaluateWithRetry(2, 3) {
            return client.secrets().inNamespace(namespace).withName(name).get()
        }
    }

    /*
        Network Policy Methods
    */

    String applyNetworkPolicy(NetworkPolicy policy) {
        return evaluateWithRetry(2, 3) {
            io.fabric8.kubernetes.api.model.networking.v1.NetworkPolicy networkPolicy =
                    createNetworkPolicyObject(policy)

            log.debug "${networkPolicy.metadata.name}: NetworkPolicy created:"
            log.debug YamlGenerator.toYaml(networkPolicy)
            io.fabric8.kubernetes.api.model.networking.v1.NetworkPolicy createdPolicy =
                    client.network().networkPolicies()
                            .inNamespace(networkPolicy.metadata.namespace ?
                                    networkPolicy.metadata.namespace :
                                    this.namespace).createOrReplace(networkPolicy)
            policy.uid = createdPolicy.metadata.uid
            return createdPolicy.metadata.uid
        }
    }

    boolean deleteNetworkPolicy(NetworkPolicy policy) {
        return evaluateWithRetry(2, 3) {
            Boolean status = client.network().networkPolicies()
                    .inNamespace(policy.namespace ? policy.namespace : this.namespace)
                    .withName(policy.name)
                    .delete()
            if (status) {
                log.debug "${policy.name}: NetworkPolicy removed."
                return true
            }
            log.debug "${policy.name}: Failed to remove NetworkPolicy."
            return false
        }
    }

    def getNetworkPolicyCount(String ns) {
        return evaluateWithRetry(2, 3) {
            return client.network().networkPolicies().inNamespace(ns).list().items.size()
        }
    }

    def getAllNetworkPoliciesNamesByNamespace(Boolean ignoreUndoneStackroxGenerated = false) {
        return evaluateWithRetry(2, 3) {
            Map<String, List<String>> networkPolicies = [:]
            client.network().networkPolicies().inAnyNamespace().list().items.each {
                boolean skip = false
                if (ignoreUndoneStackroxGenerated) {
                    if (it.spec.podSelector.matchLabels?.get("network-policies.stackrox.io/disable") == "nomatch") {
                        skip = true
                    }
                }
                skip ?: networkPolicies.containsKey(it.metadata.namespace) ?
                        networkPolicies.get(it.metadata.namespace).add(it.metadata.name) :
                        networkPolicies.put(it.metadata.namespace, [it.metadata.name])
            }
            return networkPolicies
        }
    }

    /*
        Node Methods
     */

    def getNodeCount() {
        return evaluateWithRetry(2, 3) {
            return client.nodes().list().getItems().size()
        }
    }

    List<Node> getNodeDetails() {
        return evaluateWithRetry(2, 3) {
            return client.nodes().list().items.collect {
                new Node(
                        uid: it.metadata.uid,
                        name: it.metadata.name,
                        labels: it.metadata.labels,
                        annotations: it.metadata.annotations,
                        internalIps: it.status.addresses.findAll { it.type == "InternalIP" }*.address,
                        externalIps: it.status.addresses.findAll { it.type == "ExternalIP" }*.address,
                        containerRuntimeVersion: it.status.nodeInfo.containerRuntimeVersion,
                        kernelVersion: it.status.nodeInfo.kernelVersion,
                        osImage: it.status.nodeInfo.osImage,
                        kubeletVersion: it.status.nodeInfo.kubeletVersion,
                        kubeProxyVersion: it.status.nodeInfo.kubeProxyVersion
                )
            }
        }
    }

    def isGKE() {
        return evaluateWithRetry(2, 3) {
            List<Node> gkeNodes = client.nodes().list().getItems().findAll {
                it.getStatus().getNodeInfo().getKubeletVersion().contains("gke")
            } as List<Node>
            return gkeNodes.size() > 0
        }
    }

    /*
        Namespace Methods
     */

    List<objects.Namespace> getNamespaceDetails() {
        return evaluateWithRetry(2, 3) {
            return client.namespaces().list().items.collect {
                new objects.Namespace(
                        uid: it.metadata.uid,
                        name: it.metadata.name,
                        labels: it.metadata.labels,
                        deploymentCount: getDeploymentCount(it.metadata.name) +
                                getDaemonSetCount(it.metadata.name) +
                                getStaticPodCount(it.metadata.name) +
                                getStatefulSetCount(it.metadata.name) +
                                getJobCount(it.metadata.name),
                        secretsCount: getSecretCount(it.metadata.name),
                        networkPolicyCount: getNetworkPolicyCount(it.metadata.name)
                )
            }
        }
    }

    def addNamespaceAnnotation(String ns, String key, String value) {
        client.namespaces().withName(ns).edit {
            n -> new NamespaceBuilder(n).editMetadata().addToAnnotations(key, value).endMetadata().build()
        }
    }

    def removeNamespaceAnnotation(String ns, String key) {
        client.namespaces().withName(ns).edit {
            n -> new NamespaceBuilder(n).editMetadata().removeFromAnnotations(key).endMetadata().build()
        }
    }

    List<String> getNamespaces() {
        return evaluateWithRetry(2, 3) {
            return client.namespaces().list().items.collect {
                it.metadata.name
            }
        }
    }

    /*
        Service Accounts
     */

    List<K8sServiceAccount> getServiceAccounts() {
        return evaluateWithRetry(1, 2) {
            List<K8sServiceAccount> serviceAccounts = []
            client.serviceAccounts().inAnyNamespace().list().items.each {
                // Ingest the K8s service account to a K8sServiceAccount() in a manner similar to the SR product.
                def annotations = it.metadata.annotations
                if (annotations) {
                    annotations.remove("kubectl.kubernetes.io/last-applied-configuration")
                }
                serviceAccounts.add(new K8sServiceAccount(
                        name: it.metadata.name,
                        namespace: it.metadata.namespace,
                        labels: it.metadata.labels ? it.metadata.labels : [:],
                        annotations: annotations ?: [:],
                        secrets: it.secrets,
                        imagePullSecrets: it.imagePullSecrets*.name,
                        automountToken: it.automountServiceAccountToken == null
                                ? true : it.automountServiceAccountToken,
                ))
            }
            return serviceAccounts
        }
    }

    def createServiceAccount(K8sServiceAccount serviceAccount) {
        withRetry(1, 2) {
            ServiceAccount sa = new ServiceAccount(
                    metadata: new ObjectMeta(
                            name: serviceAccount.name,
                            namespace: serviceAccount.namespace,
                            labels: serviceAccount.labels,
                            annotations: serviceAccount.annotations
                    ),
                    secrets: serviceAccount.secrets,
                    imagePullSecrets: serviceAccount.imagePullSecrets.collect {
                        String name -> new LocalObjectReference(name) }
            )
            client.serviceAccounts().inNamespace(sa.metadata.namespace).createOrReplace(sa)
        }
    }

    def deleteServiceAccount(K8sServiceAccount serviceAccount) {
        withRetry(1, 2) {
            client.serviceAccounts().inNamespace(serviceAccount.namespace).withName(serviceAccount.name).delete()
        }
    }

    def addServiceAccountImagePullSecret(String accountName, String secretName, String namespace = this.namespace) {
        withRetry(1, 2) {
            ServiceAccount serviceAccount = client.serviceAccounts()
                    .inNamespace(namespace)
                    .withName(accountName)
                    .get()

            Set<LocalObjectReference> imagePullSecretsSet = new HashSet<>(serviceAccount.getImagePullSecrets())
            imagePullSecretsSet.add(new LocalObjectReference(secretName))
            List<LocalObjectReference> imagePullSecretsList = []
            imagePullSecretsList.addAll(imagePullSecretsSet)
            serviceAccount.setImagePullSecrets(imagePullSecretsList)

            client.serviceAccounts().inNamespace(namespace).withName(accountName).createOrReplace(serviceAccount)
        }
    }

    def removeServiceAccountImagePullSecret(String accountName, String secretName, String namespace = this.namespace) {
        ServiceAccount serviceAccount = client.serviceAccounts()
                .inNamespace(namespace)
                .withName(accountName)
                .get()

        Set<LocalObjectReference> imagePullSecretsSet = new HashSet<>(serviceAccount.getImagePullSecrets())
        imagePullSecretsSet.remove(new LocalObjectReference(secretName))
        List<LocalObjectReference> imagePullSecretsList = []
        imagePullSecretsList.addAll(imagePullSecretsSet)
        serviceAccount.setImagePullSecrets(imagePullSecretsList)

        client.serviceAccounts().inNamespace(namespace).withName(accountName).createOrReplace(serviceAccount)
    }

    def provisionDefaultServiceAccount(String forNamespace) {
        if (forNamespace == this.namespace) {
            return
        }

        log.info """Copy image pull secrets from the default service account
                    in the test orchestration namespace (${this.namespace})
                    for use by the default service account in ${forNamespace} namespace.""".stripIndent()

        ServiceAccount orchestrationServiceAccount = client.serviceAccounts()
                    .inNamespace(this.namespace)
                    .withName("default")
                    .get()
        assert orchestrationServiceAccount, "Expect to find a default service account"

        List<LocalObjectReference> imagePullSecrets = orchestrationServiceAccount.getImagePullSecrets()

        imagePullSecrets.forEach {
            LocalObjectReference imagePullSecret ->
            K8sSecret secret = client.secrets().inNamespace(this.namespace).withName(imagePullSecret.name).get()
            assert secret, "the default SA has a non existing image pull secret - ${imagePullSecret.name}"

            K8sSecret copy = new K8sSecret(
                        apiVersion: "v1",
                        kind: "Secret",
                        type: secret.type,
                        data: secret.data,
                        metadata: new ObjectMeta(
                                name: secret.metadata.name,
                        )
            )
            client.secrets().inNamespace(forNamespace).createOrReplace(copy)
            assert waitForSecretCreation(copy.metadata.name, forNamespace), "could not copy the secret"
        }

        createServiceAccount(new K8sServiceAccount(
                name: "default",
                namespace: forNamespace,
                imagePullSecrets: imagePullSecrets*.name
        ))
        assert client.serviceAccounts()
                .inNamespace(this.namespace)
                .withName("default")
                .get()
                ?.imagePullSecrets == imagePullSecrets
    }

    /*
        Roles
     */

    List<K8sRole> getRoles() {
        return evaluateWithRetry(1, 2) {
            List<K8sRole> roles = []
            client.rbac().roles().inAnyNamespace().list().items.each {
                roles.add(new K8sRole(
                        name: it.metadata.name,
                        namespace: it.metadata.namespace,
                        clusterRole: false,
                        labels: it.metadata.labels ? it.metadata.labels : [:],
                        annotations: it.metadata.annotations ? it.metadata.annotations : [:],
                        rules: it?.rules ? it.rules.collect {
                            new K8sPolicyRule(
                                    verbs: it.verbs,
                                    apiGroups: it.apiGroups,
                                    resources: it.resources,
                                    nonResourceUrls: it.nonResourceURLs,
                                    resourceNames: it.resourceNames
                            )
                        } : [],
                ))
            }
            return roles
        }
    }

    def createRole(K8sRole role) {
        withRetry(1, 2) {
            Role r = new Role(
                    metadata: new ObjectMeta(
                            name: role.name,
                            namespace: role.namespace,
                            labels: role.labels,
                            annotations: role.annotations
                    ),
                    rules: role.rules.collect { K8sPolicyRule r ->
                        new PolicyRule(
                                verbs: r.verbs,
                                apiGroups: r.apiGroups,
                                resources: r.resources,
                                nonResourceURLs: r.nonResourceUrls,
                                resourceNames: r.resourceNames
                        )
                    }
            )
            role.uid = client.rbac().roles().inNamespace(role.namespace).createOrReplace(r).metadata.uid
        }
    }

    def deleteRole(K8sRole role) {
        withRetry(1, 2) {
            client.rbac().roles().inNamespace(role.namespace).withName(role.name).delete()
        }
    }

    /*
        RoleBindings
     */

    List<K8sRoleBinding> getRoleBindings() {
        return evaluateWithRetry(2, 3) {
            List<K8sRoleBinding> bindings = []
            client.rbac().roleBindings().inAnyNamespace().list().items.each {
                def b = new K8sRoleBinding(
                        new K8sRole(
                                name: it.metadata.name,
                                namespace: it.metadata.namespace,
                                clusterRole: false,
                                labels: it.metadata.labels ? it.metadata.labels : [:],
                                annotations: it.metadata.annotations ? it.metadata.annotations : [:]
                        ),
                        it.subjects.collect {
                    new K8sSubject(kind: it.kind, name: it.name, namespace: it.namespace ?: "")
                        }
                )
                def uid = it.roleRef.kind == "Role" ?
                        client.rbac().roles()
                                .inNamespace(it.metadata.namespace)
                                .withName(it.roleRef.name).get()?.metadata?.uid :
                        client.rbac().clusterRoles().withName(it.roleRef.name).get()?.metadata?.uid
                b.roleRef.uid = uid ?: ""
                bindings.add(b)
            }
            return bindings
        }
    }

    def createRoleBinding(K8sRoleBinding roleBinding) {
        withRetry(1, 2) {
            RoleBinding r = new RoleBinding(
                    metadata: new ObjectMeta(
                            name: roleBinding.name,
                            namespace: roleBinding.namespace,
                            labels: roleBinding.labels,
                            annotations: roleBinding.annotations
                    ),
                    subjects: roleBinding.subjects.collect {
                        new Subject(kind: it.kind, name: it.name, namespace: it.namespace)
                    },
                    roleRef: new RoleRef(
                            name: roleBinding.roleRef.name,
                            kind: roleBinding.roleRef.clusterRole ? "ClusterRole" : "Role"
                    )
            )
            client.rbac().roleBindings().inNamespace(roleBinding.namespace).createOrReplace(r)
        }
    }

    def deleteRoleBinding(K8sRoleBinding roleBinding) {
        withRetry(1, 2) {
            client.rbac().roleBindings()
                    .inNamespace(roleBinding.namespace)
                    .withName(roleBinding.name)
                    .delete()
        }
    }

    /*
        ClusterRoles
     */

    List<K8sRole> getClusterRoles() {
        return evaluateWithRetry(2, 3) {
            List<K8sRole> clusterRoles = []
            client.rbac().clusterRoles().list().items.each {
                clusterRoles.add(new K8sRole(
                        name: it.metadata.name,
                        namespace: "",
                        clusterRole: true,
                        labels: it.metadata.labels ? it.metadata.labels : [:],
                        annotations: it.metadata.annotations ? it.metadata.annotations : [:],
                        rules: it?.rules ? it.rules.collect {
                            new K8sPolicyRule(
                                    verbs: it.verbs,
                                    apiGroups: it.apiGroups,
                                    resources: it.resources,
                                    nonResourceUrls: it.nonResourceURLs,
                                    resourceNames: it.resourceNames
                            )
                        } : [],
                ))
            }
            return clusterRoles
        }
    }

    def createClusterRole(K8sRole role) {
        withRetry(2, 3) {
            ClusterRole r = new ClusterRole(
                    metadata: new ObjectMeta(
                            name: role.name,
                            labels: role.labels,
                            annotations: role.annotations
                    ),
                    rules: role.rules.collect {
                        new PolicyRule(
                                verbs: it.verbs,
                                apiGroups: it.apiGroups,
                                resources: it.resources,
                                nonResourceURLs: it.nonResourceUrls,
                                resourceNames: it.resourceNames
                        )
                    }
            )
            role.uid = client.rbac().clusterRoles().createOrReplace(r).metadata.uid
        }
    }

    def deleteClusterRole(K8sRole role) {
        withRetry(2, 3) {
            client.rbac().clusterRoles().withName(role.name).delete()
        }
    }

    /*
        ClusterRoleBindings
     */

    List<K8sRoleBinding> getClusterRoleBindings() {
        return evaluateWithRetry(2, 3) {
            List<K8sRoleBinding> clusterBindings = []
            client.rbac().clusterRoleBindings().list().items.each {
                def b = new K8sRoleBinding(
                        new K8sRole(
                                name: it.metadata.name,
                                namespace: "",
                                clusterRole: true,
                                labels: it.metadata.labels ? it.metadata.labels : [:],
                                annotations: it.metadata.annotations ? it.metadata.annotations : [:]
                        ),
                        it.subjects.collect {
                    new K8sSubject(kind: it.kind, name: it.name, namespace: it.namespace ?: "")
                        }
                )
                def uid = client.rbac().clusterRoles().withName(it.roleRef.name).get()?.metadata?.uid ?:
                        client.rbac().roles()
                                .inNamespace(it.metadata.namespace)
                                .withName(it.roleRef.name).get()?.metadata?.uid
                b.roleRef.uid = uid ?: ""
                clusterBindings.add(b)
            }
            return clusterBindings
        }
    }

    def createClusterRoleBinding(K8sRoleBinding roleBinding) {
        withRetry(2, 3) {
            ClusterRoleBinding r = new ClusterRoleBinding(
                    metadata: new ObjectMeta(
                            name: roleBinding.name,
                            labels: roleBinding.labels,
                            annotations: roleBinding.annotations
                    ),
                    subjects: roleBinding.subjects.collect {
                        new Subject(kind: it.kind, name: it.name, namespace: it.namespace)
                    },
                    roleRef: new RoleRef(
                            name: roleBinding.roleRef.name,
                            kind: roleBinding.roleRef.clusterRole ? "ClusterRole" : "Role"
                    )
            )
            client.rbac().clusterRoleBindings().createOrReplace(r)
        }
    }

    def deleteClusterRoleBinding(K8sRoleBinding roleBinding) {
        withRetry(2, 3) {
            client.rbac().clusterRoleBindings().withName(roleBinding.name).delete()
        }
    }

    /*
        PodSecurityPolicies
    */

    protected K8sRole generatePspRole() {
        def rules = [new K8sPolicyRule(
                apiGroups: ["policy"],
                resources: ["podsecuritypolicies"],
                resourceNames: ["allow-all-for-test"],
                verbs: ["use"]
        ),]
        return new K8sRole(
                name: "allow-all-for-test",
//                namespace: namespace,
                clusterRole: true,
                rules: rules
        )
    }

    protected K8sRoleBinding generatePspRoleBinding(String namespace) {
        def roleBinding =  new K8sRoleBinding(
                name: "allow-all-for-test-" + namespace,
                namespace: namespace,
                roleRef: generatePspRole(),
                subjects: [new K8sSubject(
                        name: "default",
                        namespace: namespace,
                        kind: "ServiceAccount"
                )]
        )
        return roleBinding
    }

    protected defaultPspForNamespace(String namespace) {
        if (Env.get("POD_SECURITY_POLICIES") != "false") {
            PodSecurityPolicy psp = new PodSecurityPolicyBuilder().withNewMetadata()
                .withName("allow-all-for-test")
                .endMetadata()
                .withNewSpec()
                .withPrivileged(true)
                .withAllowPrivilegeEscalation(true)
                .withAllowedCapabilities("*")
                .withVolumes("*")
                .withHostNetwork(true)
                .withHostPorts(new HostPortRange(65535, 0))
                .withHostIPC(true)
                .withHostPID(true)
                .withNewRunAsUser().withRule("RunAsAny").endRunAsUser()
                .withNewSeLinux().withRule("RunAsAny").endSeLinux()
                .withNewSupplementalGroups().withRule("RunAsAny").endSupplementalGroups()
                .withNewFsGroup().withRule("RunAsAny").endFsGroup()
                .endSpec()
                .build()
            client.policy().v1beta1().podSecurityPolicies().createOrReplace(psp)
            createClusterRole(generatePspRole())
            createClusterRoleBinding(generatePspRoleBinding(namespace))
        }
    }

    /*
        Jobs
     */

    List<String> getJobCount(String ns) {
        return evaluateWithRetry(2, 3) {
            return client.batch().v1().jobs().inNamespace(ns).list().getItems().collect { it.metadata.name }
        }
    }

    List<String> getJobCount() {
        return evaluateWithRetry(2, 3) {
            return client.batch().v1().jobs().list().getItems().collect { it.metadata.name }
        }
    }

    /*
        ConfigMaps
    */

    def createConfigMap(ConfigMap configMap) {
        createConfigMap(
                configMap.getName(),
                configMap.getData(),
                configMap.getNamespace()
        )
    }

    def createConfigMap(String name, Map<String,String> data, String namespace = this.namespace) {
        K8sConfigMap configMap = new K8sConfigMap(
                apiVersion: "v1",
                kind: "ConfigMap",
                data: data,
                metadata: new ObjectMeta(
                        name: name
                )
        )

        def config = client.configMaps().inNamespace(namespace).createOrReplace(configMap)
        log.debug name + ": ConfigMap created."
        return config.metadata.uid
    }

    ConfigMap getConfigMap(String name, String namespace) {
        return evaluateWithRetry(2, 3) {
            K8sConfigMap conf = client.configMaps().inNamespace(namespace).withName(name).get()
            return new ConfigMap(
                    name: conf.metadata.name,
                    namespace: conf.metadata.namespace,
                    data: conf.data
            )
        }
    }

    def deleteConfigMap(String name, String namespace) {
        withRetry(2, 3) {
            client.configMaps().inNamespace(namespace).withName(name).delete()
        }
        sleep(sleepDurationSeconds * 1000)
        log.debug name + ": ConfigMap removed."
    }

    /*
        Misc/Helper Methods
    */

    boolean execInContainerByPodName(String name, String namespace, String[] splitCmd, int retries = 1) {
        // Wait for container 0 to be running first.
        def timer = new Timer(retries, 1)
        while (timer.IsValid()) {
            def p = client.pods().inNamespace(namespace).withName(name).get()
            if (p == null || p.status.containerStatuses.size() == 0) {
                log.debug "First container in pod ${name} not yet running ..."
                continue
            }
            def status = p.status.containerStatuses.get(0)
            if (status.state.running != null) {
                log.debug "First container in pod ${name} is running"
                break
            }
            log.debug "First container in pod ${name} not yet running ..."
        }

        final CompletableFuture<ExecStatus> completion = new CompletableFuture<>()
        final outputStream = new ByteArrayOutputStream()
        final errorStream = new ByteArrayOutputStream()
        ExecStatus execStatus = ExecStatus.UNKNOWN

        final listener = new ExecListener() {
            ExecStatus status = ExecStatus.UNKNOWN

            @Override
            void onFailure(Throwable t, ExecListener.Response failureResponse) {
                log.warn("Command failed with response {}", failureResponse, t)
                completion.completeExceptionally(t)
            }

            @Override
            void onClose(int code, String reason) {
                // We ignore both code and reason because the code is websocket code, not the program exit code.
                log.debug("Websocket response: $code $reason")
                completion.complete(status)
            }

            @Override
            void onExit(int code, Status s) {
                // Do not print the full status object here as it triggers prow
                // error highlighting in CI. There is sufficient debug printed
                // elsewhere.
                log.debug("Command exited with $code")
                switch (code) {
                    case 0:
                        status = ExecStatus.SUCCESS
                        break
                    case -1:
                        status = ExecStatus.UNKNOWN
                        break
                    default:
                        status = ExecStatus.FAILURE
                }
            }
        }

        log.debug("Exec-ing the following command in pod {}: {}", name, splitCmd)
        try {
            final ExecWatch execCmd = client.pods()
                    .inNamespace(namespace)
                    .withName(name)
                    .writingOutput(outputStream)
                    .writingError(errorStream)
                    .usingListener(listener)
                    .exec(splitCmd)
            try {
                execStatus = completion.get(30, TimeUnit.SECONDS)
            } catch (TimeoutException ex) {
                // Note that timeout here does not abort the command once it started on the pod.
                execStatus = ExecStatus.TIMEOUT
                log.warn("Timeout occurred when exec-ing command in pod", ex)
            } finally {
                execCmd.close()
                log.debug("""\
                    Command status: {}
                    Stdout:
                    {}
                    Stderr:
                    {}""".stripIndent(),
                        execStatus,
                        outputStream,
                        errorStream)
            }
        } catch (Exception e) {
            log.warn("Error exec-ing command in pod", e)
        }
        return execStatus == ExecStatus.SUCCESS
    }

    boolean execInContainerByPodName(String name, String namespace, String cmd, int retries = 1) {
        String[] splitCmd = CommandLine.parse(cmd).with {
            final List<String> result = new ArrayList()
            result.add(it.getExecutable())
            result.addAll(it.getArguments())
            return result as String[]
        }
        return execInContainerByPodName(name, namespace, splitCmd, retries)
    }

    private enum ExecStatus {
        UNKNOWN,
        SUCCESS,
        /** E.g. no-zero exit code */
        FAILURE,
        TIMEOUT
    }

    def execInContainer(Deployment deployment, String cmd) {
        return execInContainerByPodName(deployment.pods.get(0).name, deployment.namespace, cmd, 30)
    }

    String generateYaml(Object orchestratorObject) {
        if (orchestratorObject instanceof NetworkPolicy) {
            return YamlGenerator.toYaml(createNetworkPolicyObject(orchestratorObject))
        }

        return ""
    }

    String getNameSpace() {
        return this.namespace
    }

    String getSensorContainerName() {
        return evaluateWithRetry(2, 3) {
            return client.pods().inNamespace("stackrox").list().items.find {
                it.metadata.name.startsWith("sensor")
            }.metadata.name
        }
    }

    def waitForSensor() {
        def start = System.currentTimeMillis()
        def running = client.apps().deployments()
                .inNamespace("stackrox")
                .withName("sensor")
                .get().status.readyReplicas < 1
        while (!running && (System.currentTimeMillis() - start) < 30000) {
            log.debug "waiting for sensor to come back online. Trying again in 1s..."
            sleep 1000
            running = client.apps().deployments()
                    .inNamespace("stackrox")
                    .withName("sensor")
                    .get().status.readyReplicas < 1
        }
        if (!running) {
            log.debug "Failed to detect sensor came back up within 30s... Future tests may be impacted."
        }
    }

    int getAllDeploymentTypesCount(String ns) {
        return getDeploymentCount(ns).size() +
                getDaemonSetCount(ns).size() +
                getStaticPodCount(ns).size() +
                getStatefulSetCount(ns).size() +
                getJobCount(ns).size()
    }

    /*
        Private K8S Support functions
    */

    def createDeploymentNoWait(Deployment deployment, int maxNumRetries=0) {
        deployment.getNamespace() != null ?: deployment.setNamespace(this.namespace)

        // Create service if needed
        if (deployment.exposeAsService) {
            createService(deployment)
        }

        K8sDeployment d = new K8sDeployment(
                metadata: new ObjectMeta(
                        name: deployment.name,
                        namespace: deployment.namespace,
                        labels: deployment.labels,
                        annotations: deployment.annotation,
                ),
                spec: new DeploymentSpec(
                        selector: new LabelSelector(null, deployment.labels),
                        replicas: deployment.replicas,
                        minReadySeconds: 15,
                        template: new PodTemplateSpec(
                                metadata: new ObjectMeta(
                                        name: deployment.name,
                                        namespace: deployment.namespace,
                                        labels: deployment.labels + ["deployment": deployment.name],
                                ),
                                spec: generatePodSpec(deployment)
                        )

                )
        )

        try {
            withK8sClientRetry(maxNumRetries, 1) {
                client.apps().deployments().inNamespace(deployment.namespace).createOrReplace(d)
                log.debug "Told the orchestrator to createOrReplace " + deployment.name
            }
            if (deployment.exposeAsService && deployment.createLoadBalancer) {
                waitForLoadBalancer(deployment)
            }
            if (deployment.createRoute) {
                createRoute(deployment.name, deployment.namespace)
                deployment.routeHost = waitForRouteHost(deployment.name, deployment.namespace)
            }
            return true
        } catch (Exception e) {
            log.warn("Error creating k8s deployment: ",  e)
            return false
        }
    }

    def waitForDeploymentAndPopulateInfo(Deployment deployment) {
        try {
            deployment.deploymentUid = waitForDeploymentStart(
                    deployment.getName(),
                    deployment.getNamespace(),
                    deployment.skipReplicaWait
            )
            updateDeploymentDetails(deployment)
        } catch (Exception e) {
            log.warn("Error while waiting for deployment/populating deployment info: ", e)
        }
        if (!deployment.skipReplicaWait && !deployment.deploymentUid) {
            throw new OrchestratorManagerException("The deployment did not start or reach replica ready state")
        }
    }

    def waitForDeploymentStart(String deploymentName, String namespace, Boolean skipReplicaWait = false) {
        Timer t = new Timer(60, 3)
        while (t.IsValid()) {
            log.debug "Waiting for ${deploymentName} to start"
            K8sDeployment d = null
            try {
                d = this.deployments.inNamespace(namespace).withName(deploymentName).get()
            } catch (Exception e) {
                log.warn("Error getting k8s deployment", e)
            }
            getAndPrintPods(namespace, deploymentName)
            if (d == null) {
                log.debug "${deploymentName} not found yet"
                continue
            } else if (skipReplicaWait) {
                // If skipReplicaWait is set, we still want to sleep for a few seconds to allow the deployment
                // to work its way through the system.
                sleep(sleepDurationSeconds * 1000)
                log.debug "${deploymentName}: deployment created (skipped replica wait)."
                return
            }
            if (d.getStatus().getReadyReplicas() == d.getSpec().getReplicas()) {
                log.debug "All ${d.getSpec().getReplicas()} replicas found " +
                        "in ready state for ${deploymentName}"
                log.debug "Took ${t.SecondsSince()} seconds for k8s deployment ${deploymentName}"
                return d.getMetadata().getUid()
            }
            log.debug "${d.getStatus().getReadyReplicas() ?: 0}/" +
                    "${d.getSpec().getReplicas()} are in the ready state for ${deploymentName}"
        }
    }

    def createDaemonSetNoWait(DaemonSet daemonSet) {
        daemonSet.getNamespace() != null ?: daemonSet.setNamespace(this.namespace)

        K8sDaemonSet ds = new K8sDaemonSet(
                metadata: new ObjectMeta(
                        name: daemonSet.name,
                        namespace: daemonSet.namespace,
                        labels: daemonSet.labels
                ),
                spec: new DaemonSetSpec(
                        minReadySeconds: 15,
                        selector: new LabelSelector(null, daemonSet.labels),
                        template: new PodTemplateSpec(
                                metadata: new ObjectMeta(
                                        name: daemonSet.name,
                                        namespace: daemonSet.namespace,
                                        labels: daemonSet.labels
                                ),
                                spec: generatePodSpec(daemonSet)
                        )
                )
        )

        try {
            this.daemonsets.inNamespace(daemonSet.namespace).createOrReplace(ds)
            log.debug "Told the orchestrator to create " + daemonSet.getName()
        } catch (Exception e) {
            log.warn("Error creating k8s deployment", e)
        }
    }

    def waitForDaemonSetAndPopulateInfo(DaemonSet daemonSet) {
        try {
            daemonSet.deploymentUid = waitForDaemonSetCreation(
                    daemonSet.getName(),
                    daemonSet.getNamespace(),
                    daemonSet.skipReplicaWait
            )
            updateDeploymentDetails(daemonSet)
        } catch (Exception e) {
            log.warn("Error while waiting for daemonset/populating daemonset info: ", e)
        }
    }

    def waitForDaemonSetCreation(String name, String namespace, Boolean skipReplicaWait = false) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            log.debug "Waiting for ${name} to start"
            K8sDaemonSet d = this.daemonsets.inNamespace(namespace).withName(name).get()
            getAndPrintPods(namespace, name)
            if (d == null) {
                log.debug "${name} not found yet"
                continue
            } else if (skipReplicaWait) {
                // If skipReplicaWait is set, we still want to sleep for a few seconds to allow the deployment
                // to work its way through the system.
                sleep(sleepDurationSeconds * 1000)
                log.debug "${name}: daemonset created (skipped replica wait)."
                return
            }
            if (d.getStatus().getCurrentNumberScheduled() == d.getStatus().getDesiredNumberScheduled()) {
                log.debug "All ${d.getStatus().getDesiredNumberScheduled()} replicas found in ready state for ${name}"
                return d.getMetadata().getUid()
            }
            log.debug "${d.getStatus().getCurrentNumberScheduled()}/" +
                    "${d.getStatus().getDesiredNumberScheduled()} are in the ready state for ${name}"
        }
    }

    PodSpec generatePodSpec(Deployment deployment) {
        List<ContainerPort> depPorts = deployment.ports.collect {
            k, v -> new ContainerPort(
                    k as Integer,
                    null,
                    null,
                    "port" + (k as String),
                    v as String
            )
        }

        List<LocalObjectReference> imagePullSecrets = new LinkedList<>()
        for (String str : deployment.getImagePullSecret()) {
            LocalObjectReference obj = new LocalObjectReference(name: str)
            imagePullSecrets.add(obj)
        }

        List<EnvVar> envVars = deployment.env.collect {
            k, v -> new EnvVar(k, v, null)
        }

        deployment.envValueFromSecretKeyRef.forEach {
            String k, SecretKeyRef v -> envVars.add(new EnvVarBuilder()
                    .withName(k)
                    .withValueFrom(new EnvVarSourceBuilder()
                            .withSecretKeyRef(
                                    new SecretKeySelectorBuilder().withKey(v.key).withName(v.name).build())
                            .build())
                    .build())
        }

        deployment.envValueFromConfigMapKeyRef.forEach {
            String k, ConfigMapKeyRef v -> envVars.add(new EnvVarBuilder()
                    .withName(k)
                    .withValueFrom(new EnvVarSourceBuilder()
                            .withConfigMapKeyRef(
                                    new ConfigMapKeySelectorBuilder().withKey(v.key).withName(v.name).build())
                            .build())
                    .build())
        }

        deployment.envValueFromFieldRef.forEach {
            String k, String fieldPath -> envVars.add(new EnvVarBuilder()
                    .withName(k)
                    .withValueFrom(new EnvVarSourceBuilder()
                            .withFieldRef(
                                    new ObjectFieldSelectorBuilder().withFieldPath(fieldPath).build())
                            .build())
                    .build())
        }

        deployment.envValueFromResourceFieldRef.forEach {
            String k, String resource -> envVars.add(new EnvVarBuilder()
                    .withName(k)
                    .withValueFrom(new EnvVarSourceBuilder()
                            .withResourceFieldRef(
                                    new ResourceFieldSelectorBuilder().withResource(resource).build())
                            .build())
                    .build())
        }

        List<EnvFromSource> envFrom = new LinkedList<>()
        for (String secret : deployment.getEnvFromSecrets()) {
            envFrom.add(new EnvFromSource(null, null, new SecretEnvSource(secret, false)))
        }
        for (String configMapName : deployment.getEnvFromConfigMaps()) {
            envFrom.add(new EnvFromSource(new ConfigMapEnvSource(configMapName, false), null, null))
        }

        List<Volume> volumes = []
        deployment.volumes.each {
            v -> Volume vol = new Volume(
                    name: v.name,
                    configMap: v.configMap ? new ConfigMapVolumeSource(
                            name: v.configMap.name
                        ) :
                        null,
                    hostPath: v.hostPath ? new HostPathVolumeSource(
                            path: v.mountPath,
                            type: "Directory") :
                            null,
                    secret: deployment.secretNames.get(v.name) ?
                            new SecretVolumeSource(secretName: deployment.secretNames.get(v.name)) :
                            null
            )
            volumes.add(vol)
        }

        List<VolumeMount> volMounts = []
        deployment.volumeMounts.each {
            v -> VolumeMount volMount = new VolumeMount(
                    mountPath: v.mountPath,
                    name: v.name,
                    readOnly: v.readOnly
            )
            volMounts.add(volMount)
        }

        Map<String , Quantity> limits = new HashMap<>()
        for (String key:deployment.limits.keySet()) {
            Quantity quantity = new Quantity(deployment.limits.get(key))
            limits.put(key, quantity)
        }

        Map<String , Quantity> requests = new HashMap<>()
        for (String key:deployment.request.keySet()) {
            Quantity quantity = new Quantity(deployment.request.get(key))
            requests.put(key, quantity)
        }

        Container container = new Container(
                name: deployment.containerName ? deployment.containerName : deployment.name,
                image: deployment.image,
                command: deployment.command,
                args: deployment.args,
                ports: depPorts,
                volumeMounts: volMounts,
                env: envVars,
                envFrom: envFrom,
                resources: new ResourceRequirements([], limits, requests),
                securityContext: new SecurityContext(privileged: deployment.isPrivileged,
                                                     readOnlyRootFilesystem: deployment.readOnlyRootFilesystem,
                                                     capabilities: new Capabilities(add: deployment.addCapabilities,
                                                                                    drop: deployment.dropCapabilities)),
        )
        if (deployment.livenessProbeDefined) {
            Probe livenessProbe = new Probe(
                exec: new ExecAction(command: ["touch", "/tmp/healthy"]),
                periodSeconds: 5,
            )
            container.setLivenessProbe(livenessProbe)
        }
        if (deployment.readinessProbeDefined) {
            Probe readinessProbe = new Probe(
                exec: new ExecAction(command: ["touch", "/tmp/ready"]),
                periodSeconds: 5,
            )
            container.setReadinessProbe(readinessProbe)
        }

        PodSpec podSpec = new PodSpec(
                containers: [container],
                volumes: volumes,
                imagePullSecrets: imagePullSecrets,
                hostNetwork: deployment.hostNetwork,
                serviceAccountName: deployment.serviceAccountName
        )
        if (!deployment.automountServiceAccountToken) {
            podSpec.automountServiceAccountToken = deployment.automountServiceAccountToken
        }
        return podSpec
    }

    def updateDeploymentDetails(Deployment deployment) {
        // Filtering pod query by using the "name=<name>" because it should always be present in the deployment
        // object - IF this is ever missing, it may cause problems fetching pod details
        PodList deployedPods = evaluateWithRetry(2, 3) {
            return client.pods().inNamespace(deployment.namespace).withLabel("name", deployment.name).list()
        }
        log.debug("Updating deployment ${deployment.name} with ${deployedPods.getItems().size()} pods")
        for (Pod pod : deployedPods.getItems()) {
            List<String> containerIDs = pod.getStatus().getContainerStatuses()*.getContainerID() ?: []
            deployment.addPod(
                    pod.getMetadata().getName(),
                    pod.getMetadata().getUid(),
                    containerIDs,
                    pod.getStatus().getPodIP()
            )
        }
    }

    protected io.fabric8.kubernetes.api.model.networking.v1.NetworkPolicy createNetworkPolicyObject(
            NetworkPolicy policy) {
        def networkPolicy = new NetworkPolicyBuilder()
                .withApiVersion("networking.k8s.io/v1")
                .withKind("NetworkPolicy")
                .withNewMetadata()
                .withName(policy.name)

        if (policy.namespace) {
            networkPolicy.withNamespace(policy.namespace)
        }

        if (policy.labels != null) {
            networkPolicy.withLabels(policy.labels)
        }

        networkPolicy = networkPolicy.endMetadata().withNewSpec()

        if (policy.metadataPodSelector != null) {
            networkPolicy.withNewPodSelector().withMatchLabels(policy.metadataPodSelector).endPodSelector()
        }

        if (policy.types != null) {
            List<String> polTypes = []
            for (NetworkPolicyTypes type : policy.types) {
                polTypes.add(type.toString())
            }
            networkPolicy.withPolicyTypes(polTypes)
        }

        if (policy.ingressPodSelector != null) {
            networkPolicy.withIngress(
                    new NetworkPolicyIngressRuleBuilder().withFrom(
                            new NetworkPolicyPeerBuilder()
                                    .withNewPodSelector()
                                    .withMatchLabels(policy.ingressPodSelector)
                                    .endPodSelector().build()
                    ).build()
            )
        }

        if (policy.egressPodSelector != null) {
            networkPolicy.withEgress(
                    new NetworkPolicyEgressRuleBuilder().withTo(
                            new NetworkPolicyPeerBuilder()
                                    .withNewPodSelector()
                                    .withMatchLabels(policy.egressPodSelector)
                                    .endPodSelector().build()
                    ).build()
            )
        }

        if (policy.ingressNamespaceSelector != null) {
            networkPolicy.withIngress(
                    new NetworkPolicyIngressRuleBuilder().withFrom(
                            new NetworkPolicyPeerBuilder()
                                    .withNewNamespaceSelector()
                                    .withMatchLabels(policy.ingressNamespaceSelector)
                                    .endNamespaceSelector().build()
                    ).build()
            )
        }

        if (policy.egressNamespaceSelector != null) {
            networkPolicy.withEgress(
                    new NetworkPolicyEgressRuleBuilder().withTo(
                            new NetworkPolicyPeerBuilder()
                                    .withNewNamespaceSelector()
                                    .withMatchLabels(policy.egressNamespaceSelector)
                                    .endNamespaceSelector().build()
                    ).build()
            )
        }

        return networkPolicy.endSpec().build()
    }

    ValidatingWebhookConfiguration getAdmissionController() {
        log.debug "get admission controllers stub"
    }

    def deleteAdmissionController(String name) {
        log.debug "delete admission controllers stub: ${name}"
    }

    def createAdmissionController(ValidatingWebhookConfiguration config) {
        log.debug "create admission controllers stub: ${config}"
    }

    /**
     * Creates namespace.
     * Note that createNamespace does not provision service account.
     */
    String createNamespace(String ns) {
        return evaluateWithRetry(2, 3) {
            Namespace namespace = newNamespace(ns)
            def namespaceId = client.namespaces().createOrReplace(namespace).metadata.getUid()
            // defaultPspForNamespace(ns)
            return namespaceId
        }
    }

    static Namespace newNamespace(String ns) {
        Namespace namespace = new Namespace()
        ObjectMeta meta = new ObjectMeta()
        meta.setNamespace(ns)
        meta.setName(ns)
        namespace.setMetadata(meta)
        return namespace
    }

    def deleteNamespace(String ns, Boolean waitForDeletion = true) {
        withRetry(2, 3) {
            client.namespaces().withName(ns).delete()
        }
        if (waitForDeletion) {
            waitForNamespaceDeletion(ns)
        }
    }

    def waitForNamespaceDeletion(String ns, int retries = 20, int intervalSeconds = 3) {
        log.debug "Waiting for namespace ${ns} to be deleted"
        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            if (client.namespaces().withName(ns).get() == null ) {
                log.debug "K8s found that namespace ${ns} was deleted"
                return true
            }
            log.debug "Retrying in ${intervalSeconds}..."
        }
        log.info "K8s did not detect that namespace ${ns} was deleted"
        return false
    }
}
