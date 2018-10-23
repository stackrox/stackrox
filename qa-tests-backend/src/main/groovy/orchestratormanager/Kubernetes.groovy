package orchestratormanager

import common.YamlGenerator
import io.kubernetes.client.ApiClient
import io.kubernetes.client.ApiException
import io.kubernetes.client.Configuration
import io.kubernetes.client.apis.CoreV1Api
import io.kubernetes.client.apis.ExtensionsV1beta1Api
import io.kubernetes.client.custom.IntOrString
import io.kubernetes.client.models.ExtensionsV1beta1DeploymentList
import io.kubernetes.client.models.V1Capabilities
import io.kubernetes.client.models.V1LabelSelector
import io.kubernetes.client.models.V1LocalObjectReference
import io.kubernetes.client.models.V1ObjectMeta
import io.kubernetes.client.models.V1Namespace
import io.kubernetes.client.models.ExtensionsV1beta1Deployment
import io.kubernetes.client.models.ExtensionsV1beta1DeploymentSpec
import io.kubernetes.client.models.V1ContainerPort
import io.kubernetes.client.models.V1PodTemplateSpec
import io.kubernetes.client.models.V1PodSpec
import io.kubernetes.client.models.V1SecretVolumeSource
import io.kubernetes.client.models.V1Volume
import io.kubernetes.client.models.V1Container
import io.kubernetes.client.models.V1PodList
import io.kubernetes.client.models.V1Pod
import io.kubernetes.client.models.V1DeleteOptions
import io.kubernetes.client.models.V1SecurityContext
import io.kubernetes.client.models.V1Service
import io.kubernetes.client.models.V1Secret
import io.kubernetes.client.models.V1ServicePort
import io.kubernetes.client.models.V1ServiceSpec
import io.kubernetes.client.models.V1VolumeMount
import io.kubernetes.client.models.V1Status
import io.kubernetes.client.models.V1beta1NetworkPolicy
import io.kubernetes.client.models.V1beta1NetworkPolicyEgressRule
import io.kubernetes.client.models.V1beta1NetworkPolicyIngressRule
import io.kubernetes.client.models.V1beta1NetworkPolicyPeer
import io.kubernetes.client.models.V1beta1NetworkPolicySpec
import io.kubernetes.client.util.Config
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes

import java.util.stream.Collectors

class Kubernetes extends OrchestratorCommon implements OrchestratorMain {
    private final String namespace
    private final int sleepDuration = 5000
    private final int maxWaitTime = 30000

    final private CoreV1Api api
    final private ExtensionsV1beta1Api beta1

    Kubernetes(String ns) {
        this.namespace = ns
        ApiClient client = Config.defaultClient()
        Configuration.setDefaultApiClient(client)

        this.api = new CoreV1Api()
        this.beta1 = new ExtensionsV1beta1Api()

        ensureNamespaceExists()
    }

    Kubernetes() {
        Kubernetes("default")
    }

    def ensureNamespaceExists() {
        V1Namespace namespace = new V1Namespace().apiVersion("v1").metadata(new V1ObjectMeta().name(this.namespace))
        try {
            this.api.createNamespace(namespace, null)
            println "Created namespace ${namespace}"
        } catch (ApiException e) {
            // 409 is already exists
            if (e.code != 409) {
                throw e
            }
        }
    }

    @Override
    def setup() {
    }

    @Override
    def cleanup() {
    }

    def waitForDeploymentCreation(String deploymentName, String namespace, Boolean skipReplicaWait = false) {
        int waitTime = 0

        while (waitTime < maxWaitTime) {
            ExtensionsV1beta1DeploymentList dList
            dList = beta1.listNamespacedDeployment(namespace, null, null, null, null, null, null, null, null, null)

            println "Waiting for " + deploymentName
            for (ExtensionsV1beta1Deployment v1beta1Deployment : dList.getItems()) {
                if (v1beta1Deployment.getMetadata().getName() == deploymentName) {
                    // Using the 'skipReplicaWait' bool to avoid timeout waiting for ready replicas if we know
                    // the deployment will not have replicas available
                    if (v1beta1Deployment.getStatus().getReadyReplicas() ==
                            v1beta1Deployment.getSpec().getReplicas() ||
                            skipReplicaWait) {
                        // If skipReplicaWait is set, we still want to sleep for a few seconds to allow the deployment
                        // to work its way through the system.
                        if (skipReplicaWait) {
                            sleep(sleepDuration)
                        }
                        println deploymentName + ": deployment created."
                        return v1beta1Deployment.getMetadata().getUid()
                    }
                }
            }
            sleep(sleepDuration)
            waitTime += sleepDuration
        }
        println "Timed out waiting for " + deploymentName
    }

    String getDeploymentId(String deploymentName, String namespace = this.namespace) {
        ExtensionsV1beta1DeploymentList dList
        dList = beta1.listNamespacedDeployment(namespace, null, null, null, null, null, null, null, null, null)
        for (ExtensionsV1beta1Deployment v1beta1Deployment : dList.getItems()) {
            if (v1beta1Deployment.getMetadata().getName() == deploymentName) {
                def val = v1beta1Deployment.getMetadata().uid
                if (v1beta1Deployment.getStatus().getReadyReplicas() > 0) {
                    println val + ": deployment id found."
                    return val
                }
            }
        }
    }

    def batchCreateDeployments(List<Deployment> deployments) {
        for (Deployment deployment: deployments) {
            createDeploymentNoWait(deployment)
        }
        for (Deployment deployment: deployments) {
            waitForDeploymentAndPopulateInfo(deployment)
        }
    }

    def createDeploymentNoWait(Deployment deployment) {
        deployment.getNamespace() != null ?: deployment.setNamespace(this.namespace)

        List<V1ContainerPort> containerPorts = deployment.getPorts().collect {
            k, v -> new V1ContainerPort().containerPort(k).protocol(v)
        }

        List<V1VolumeMount> deploymount = new LinkedList<>()
        for (int i = 0; i < deployment.getVolMounts().size(); ++i) {
            V1VolumeMount volmount = new V1VolumeMount()
                    .name(deployment.getVolMounts().get(i))
                    .mountPath(deployment.getMountpath())
                    .readOnly(true)
            deploymount.add(volmount)
        }

        List<V1Volume> deployVolumes = new LinkedList<>()
        for (int i = 0; i < deployment.getVolNames().size(); ++i) {
            V1Volume deployVol = new V1Volume()
                    .name(deployment.getVolNames().get(i))
                    .secret(new V1SecretVolumeSource()
                    .secretName(deployment.getSecretNames().get(i)))
            deployVolumes.add(deployVol)
        }

        V1PodSpec v1PodSpec = new V1PodSpec()
                .containers(
                [
                        new V1Container()
                                .name(deployment.getName())
                                .image(deployment.getImage())
                                .command(deployment.getCommand())
                                .args(deployment.getArgs())
                                .ports(containerPorts)
                                .volumeMounts(deploymount),
                ]
        )
                .volumes(deployVolumes)

        ExtensionsV1beta1Deployment k8sDeployment = new ExtensionsV1beta1Deployment()
                .metadata(
                new V1ObjectMeta()
                            .name(deployment.getName())
                            .namespace(deployment.getNamespace())
                            .labels(deployment.getLabels()))
                    .spec(new ExtensionsV1beta1DeploymentSpec()
                    .replicas(1)
                    .minReadySeconds(15)
                    .template(new V1PodTemplateSpec()
                    .spec(v1PodSpec)
                    .metadata(new V1ObjectMeta()
                        .name(deployment.getName())
                        .namespace(this.namespace)
                        .labels(deployment.getLabels()))
        )
        )
        try {
            beta1.createNamespacedDeployment(deployment.getNamespace(), k8sDeployment, null)
            println("Told the orchestrator to create " + deployment.getName())
        } catch (Exception e) {
            println("Error creating kube deployment" + e.toString())
        }
    }

    def waitForDeploymentAndPopulateInfo(Deployment deployment) {
        try {
            deployment.deploymentUid = waitForDeploymentCreation(
                    deployment.getName(),
                    deployment.getNamespace(),
                    deployment.skipReplicaWait
            )

            // Filtering pod query by using the "name=<name>" because it should always be present in the deployment
            // object - IF this is ever missing, it may cause problems fetching pod details
            V1PodList deployedPods = this.api.listNamespacedPod(
                    deployment.namespace,
                    null,
                    null,
                    null,
                    null,
                    "name=" + deployment.name,
                    null,
                    null,
                    null,
                    null
            )
            for (V1Pod pod : deployedPods.getItems()) {
                deployment.addPod(
                        pod.getMetadata().getName(),
                        pod.getMetadata().getUid(),
                        pod.getStatus().getContainerStatuses() != null ?
                                pod.getStatus().getContainerStatuses().stream().map {
                                    container -> container.getContainerID()
                                }.collect(Collectors.toList()) :
                                [],
                        pod.getStatus().getPodIP()
                )
            }

            // Create service if needed
            if (deployment.exposeAsService) {
                api.createNamespacedService(
                        this.namespace,
                        new V1Service()
                                .metadata(
                                        new V1ObjectMeta()
                                                .name(deployment.name)
                                                .namespace(this.namespace)
                                                .labels(deployment.labels)
                                )
                                .spec(
                                        new V1ServiceSpec()
                                                .ports(deployment.getPorts().collect {
                                            k, v -> new V1ServicePort()
                                                    .name(k as String)
                                                    .port(k as Integer)
                                                    .protocol(v) })
                                                .selector(deployment.labels)
                                ),
                        null
                )
                println "${deployment.name}: Service created"
            }
        } catch (Exception e) {
            println("Error while waiting for deployment/populating deployment info: " + e.toString())
        }
    }

    def createDeployment(Deployment deployment) {
        createDeploymentNoWait(deployment)
        waitForDeploymentAndPopulateInfo(deployment)
    }

    def deleteDeployment(String name, String namespace = this.namespace, Boolean deleteService = false) {
        if (deleteService) {
            this.deleteService(name, namespace)
        }
        this.beta1.deleteNamespacedDeployment(
                name,
                namespace, new V1DeleteOptions()
                .gracePeriodSeconds(0)
                .orphanDependents(false),
                null,
                0,
                false,
                null
        )
        println name + ": deployment removed."
    }

    def deleteSecret(String name, String namespace = this.namespace) {
        this.api.deleteNamespacedSecret(
                name,
                namespace, new V1DeleteOptions()
                .gracePeriodSeconds(0)
                .orphanDependents(false),
                null,
                0,
                false,
                null
        )
        sleep(sleepDuration)
        println name + ": Secret removed."
    }

    def deleteService(String name, String namespace = this.namespace) {
        this.api.deleteNamespacedService(
                name,
                namespace, new V1DeleteOptions()
                .gracePeriodSeconds(0)
                .orphanDependents(false),
                null,
                0,
                false,
                null
        )
    }

    String createSecret(String name) {
        Map<String, byte[]> data = new HashMap<String, byte[]>()
        data.put("username", "YWRtaW4=".getBytes())
        data.put("password", "MWYyZDFlMmU2N2Rm".getBytes())

        V1Secret createsecret = new V1Secret()
                .apiVersion("v1")
                .kind("Secret")
                .metadata(new V1ObjectMeta()
                .name(name))
                .type("Opaque")
                .data(data)
        V1Secret createdSecret = this.api.createNamespacedSecret("qa", createsecret, "true")
        return createdSecret.metadata.uid
    }

    def createClairifyDeployment() {
        //create clairify service
        Map<String, String> selector = new HashMap<String, String>()
        selector.put("app", "clairify")

        V1Service clairifyService = new V1Service()
                .apiVersion("v1")
                .metadata(new V1ObjectMeta()
                .name("clairify")
                .namespace("stackrox"))
                .spec(new V1ServiceSpec()
                .addPortsItem(new V1ServicePort()
                .name("clair-http")
                .port(6060)
                .targetPort(new IntOrString(6060)
        )
        )
                .addPortsItem(new V1ServicePort()
                .name("clairify-http")
                .port(8080)
                .targetPort(new IntOrString(8080)
        )
        )
                .type("ClusterIP")
                .selector(selector)
        )
        this.api.createNamespacedService("stackrox", clairifyService, null)

        //create clairify deployment
        Map<String, String> labels = new HashMap<>()
        labels.put("app", "clairify")
        Map<String, String> annotations = new HashMap<>()
        annotations.put("owner", "stackrox")
        annotations.put("email", "support@stackrox.com")

        List<String> commands = new LinkedList<>()
        commands.add("/init")
        commands.add("/clairify")

        V1Container clairContainer = new V1Container()
                .name("clairify")
                .image("stackrox/clairify:0.3.1")
                .command(commands)
                .imagePullPolicy("Always")
                .addPortsItem(new V1ContainerPort()
                .name("clair-http")
                .containerPort(6060)
        )
                .addPortsItem(new V1ContainerPort()
                .name("clairify-http")
                .containerPort(8080)
        )
                .securityContext(new V1SecurityContext()
                .capabilities(new V1Capabilities()
                .addDropItem("NET_RAW")
        )
        )

        ExtensionsV1beta1Deployment clairifyDeployment = new ExtensionsV1beta1Deployment()
                .metadata(new V1ObjectMeta()
                .name("clairify")
                .namespace("stackrox")
                .labels(labels).annotations(annotations)
        )
                .spec(new ExtensionsV1beta1DeploymentSpec()
                .replicas(1)
                .selector(new V1LabelSelector()
                .matchLabels(labels))
                .template(new V1PodTemplateSpec()
                .metadata(new V1ObjectMeta()
                .namespace("stackrox")
                .labels(labels))
                .spec(new V1PodSpec()
                .addContainersItem(clairContainer)
                .addImagePullSecretsItem(new V1LocalObjectReference()
                .name("stackrox")
        )
        )
        )
        )

        this.beta1.createNamespacedDeployment("stackrox", clairifyDeployment, null)
        waitForDeploymentCreation("clairify", "stackrox")
    }

    String getClairifyEndpoint() {
        return "clairify.stackrox:8080"
    }

    String applyNetworkPolicy(NetworkPolicy policy) {
        V1beta1NetworkPolicy networkPolicy = createNetworkPolicyObject(policy)

        println "${networkPolicy.metadata.name}: NetworkPolicy created:"
        println YamlGenerator.toYaml(networkPolicy)
        V1beta1NetworkPolicy createdPolicy = this.beta1.createNamespacedNetworkPolicy(
                networkPolicy.metadata.namespace ?
                        networkPolicy.metadata.namespace :
                        this.namespace,
                networkPolicy,
                null
        )
        policy.uid = createdPolicy.metadata.uid
        return createdPolicy.metadata.uid
    }

    boolean deleteNetworkPolicy(NetworkPolicy policy) {
        V1Status status = this.beta1.deleteNamespacedNetworkPolicy(
                policy.name,
                policy.namespace ?
                        policy.namespace :
                        this.namespace,
                new V1DeleteOptions()
                        .gracePeriodSeconds(0)
                        .orphanDependents(false),
                null,
                0,
                false,
                null
        )
        if (status.status == "Success") {
            println "${policy.name}: NetworkPolicy removed."
            return true
        }

        println "${policy.name}: Failed to remove NetworkPolicy."
        return false
    }

    String generateYaml(Object orchestratorObject) {
        if (orchestratorObject instanceof NetworkPolicy) {
            return YamlGenerator.toYaml(createNetworkPolicyObject(orchestratorObject))
        }

        return ""
    }

    def wasContainerKilled(String containerName, String namespace = this.namespace) {
        Long startTime = System.currentTimeMillis()
        V1PodList pods = new V1PodList()

        while (System.currentTimeMillis() - startTime < 60000) {
            try {
                pods = api.listNamespacedPod(
                        namespace,
                        null,
                        null,
                        "metadata.name=${containerName}",
                        true,
                        null,
                        Integer.MAX_VALUE,
                        null,
                        180,
                        false)
                if (pods.items.size() == 0) {
                    println "Could not query K8S for pod details, assuming pod was killed"
                    return true
                }
            } catch (Exception e) {
                println "wasContainerKilled: error fetching pod details - retrying"
            }
        }

        println "wasContainerKilled: did not determine container was killed before 60s timeout"
        println "container details were found:\n${containerName}: ${pods.getItems().get(0).toString()}"
        return false
    }

    def getDeploymentReplicaCount(Deployment deployment) {
        ExtensionsV1beta1DeploymentList deployments = this.beta1.listNamespacedDeployment(
                deployment.namespace,
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                null
        )
        for (ExtensionsV1beta1Deployment d : deployments.getItems()) {
            if (d.getMetadata().getName() == deployment.name) {
                println "${deployment.name}: Replicas=${d.getSpec().getReplicas()}"
                return d.getSpec().getReplicas()
            }
        }
        return null
    }

    def getDeploymentUnavailableReplicaCount(Deployment deployment) {
        ExtensionsV1beta1DeploymentList deployments = this.beta1.listNamespacedDeployment(
                deployment.namespace,
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                null
        )
        for (ExtensionsV1beta1Deployment d : deployments.getItems()) {
            if (d.getMetadata().getName() == deployment.name) {
                println "${deployment.name}: Unavailable Replicas=${d.getStatus().getUnavailableReplicas()}"
                return d.getStatus().getUnavailableReplicas()
            }
        }
        return null
    }

    def getDeploymentNodeSelectors(Deployment deployment) {
        ExtensionsV1beta1DeploymentList deployments = this.beta1.listNamespacedDeployment(
                deployment.namespace,
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                null
        )
        for (ExtensionsV1beta1Deployment d : deployments.getItems()) {
            if (d.getMetadata().getName() == deployment.name) {
                println "${deployment.name}: Host=${d.getSpec().getTemplate().getSpec().getNodeSelector()}"
                return d.getSpec().getTemplate().getSpec().getNodeSelector()
            }
        }
        return null
    }

    /*
        Private K8S Support functions
     */

    private V1beta1NetworkPolicy createNetworkPolicyObject(NetworkPolicy policy) {
        V1beta1NetworkPolicy networkPolicy = new V1beta1NetworkPolicy()
        networkPolicy.setApiVersion("extensions/v1beta1")
        networkPolicy.setKind("NetworkPolicy")
        networkPolicy.setMetadata(new V1ObjectMeta())
        networkPolicy.setSpec(new V1beta1NetworkPolicySpec())
        networkPolicy.getMetadata().setName(policy.name)

        if (policy.namespace) {
            networkPolicy.getMetadata().setNamespace(policy.namespace)
        }

        if (policy.metadataPodSelector != null) {
            networkPolicy.getSpec().setPodSelector(new V1LabelSelector().matchLabels(policy.metadataPodSelector))
        }

        if (policy.types != null) {
            for (NetworkPolicyTypes type : policy.types) {
                networkPolicy.getSpec().addPolicyTypesItem(type.toString())
            }
        }

        if (policy.ingressPodSelector != null) {
            networkPolicy.getSpec().addIngressItem(
                    new V1beta1NetworkPolicyIngressRule().addFromItem(
                            new V1beta1NetworkPolicyPeer().podSelector(
                                    new V1LabelSelector().matchLabels(policy.ingressPodSelector)
                            )
                    )
            )
        }

        if (policy.egressPodSelector != null) {
            networkPolicy.getSpec().addEgressItem(
                    new V1beta1NetworkPolicyEgressRule().addToItem(
                            new V1beta1NetworkPolicyPeer().podSelector(
                                    new V1LabelSelector().matchLabels(policy.egressPodSelector)
                            )
                    )
            )
        }

        if (policy.ingressNamespaceSelector != null) {
            networkPolicy.getSpec().addIngressItem(
                    new V1beta1NetworkPolicyIngressRule().addFromItem(
                            new V1beta1NetworkPolicyPeer().namespaceSelector(
                                    new V1LabelSelector().matchLabels(policy.ingressNamespaceSelector)
                            )
                    )
            )
        }

        if (policy.egressNamespaceSelector != null) {
            networkPolicy.getSpec().addEgressItem(
                    new V1beta1NetworkPolicyEgressRule().addToItem(
                            new V1beta1NetworkPolicyPeer().namespaceSelector(
                                    new V1LabelSelector().matchLabels(policy.egressNamespaceSelector)
                            )
                    )
            )
        }

        return networkPolicy
    }

    String getpods() {
        List<String>podIds = new ArrayList<>()
        V1PodList pods = this.api.listNamespacedPod("qa", "", "", "", false, "", 1, "", 5, false)
        List<V1Pod> podlist = pods.getItems()
        for ( V1Pod pod : podlist) {
            podIds.add(podlist.metadata.name)
        }
        return podIds.get(0)
    }
}

