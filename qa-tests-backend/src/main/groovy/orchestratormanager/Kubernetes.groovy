package orchestratormanager

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
import io.kubernetes.client.models.V1DeleteOptions
import io.kubernetes.client.models.V1SecurityContext
import io.kubernetes.client.models.V1Service
import io.kubernetes.client.models.V1Secret
import io.kubernetes.client.models.V1ServicePort
import io.kubernetes.client.models.V1ServiceSpec
import io.kubernetes.client.models.V1VolumeMount
import io.kubernetes.client.util.Config
import objects.Deployment

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

    def portToContainerPort = { p -> new V1ContainerPort().containerPort(p) }

    def waitForDeploymentCreation(String deploymentName, String namespace) {
        int waitTime = 0

        while (waitTime < maxWaitTime) {
            ExtensionsV1beta1DeploymentList dList
            dList = beta1.listNamespacedDeployment(namespace, null, null, null, null, null, null, null, null, null)

            for (ExtensionsV1beta1Deployment v1beta1Deployment : dList.getItems()) {
                if (v1beta1Deployment.getMetadata().getName() == deploymentName) {
                    println "Waiting for " + deploymentName
                    sleep(sleepDuration)
                    if (v1beta1Deployment.getStatus().getReplicas() > 0) {
                        println deploymentName + ": deployment created."
                        //continue to sleep 5s to make the test more stable
                        sleep(sleepDuration)
                        return
                    }
                }
                waitTime += sleepDuration
            }
        }
        println "Timed out waiting for " + deploymentName
    }

    String createDeployment(Deployment deployment) {
        List<V1ContainerPort> containerPorts = deployment.getPorts().stream()
                .map(portToContainerPort)
                .collect(Collectors.<V1ContainerPort> toList())

        List<V1VolumeMount>deploymount = new LinkedList<>()
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
               .ports(containerPorts)
               .volumeMounts(deploymount),
          ]
         )
        .volumes(deployVolumes)

        ExtensionsV1beta1Deployment k8sDeployment = new ExtensionsV1beta1Deployment()
                    .metadata(
                    new V1ObjectMeta()
                            .name(deployment.getName())
                            .namespace(this.namespace)
                            .labels(deployment.getLabels()))
                    .spec(new ExtensionsV1beta1DeploymentSpec()
                    .replicas(1)
                    .template(new V1PodTemplateSpec()
                    .spec(v1PodSpec)
                    .metadata(new V1ObjectMeta()
                        .name(deployment.getName())
                        .labels(deployment.getLabels())
            )
            )
            )

        try {
            beta1.createNamespacedDeployment(this.namespace, k8sDeployment, null)
            waitForDeploymentCreation(deployment.getName(), this.namespace)
        } catch (Exception e) {
            println("Creating deployment error: " + e.toString())
        }
    }

    def deleteDeployment(String name, String namespace = this.namespace) {
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
        sleep(sleepDuration)
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
    }
