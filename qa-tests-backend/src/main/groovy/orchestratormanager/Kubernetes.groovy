package orchestratormanager

import io.kubernetes.client.ApiClient
import io.kubernetes.client.ApiException
import io.kubernetes.client.Configuration
import io.kubernetes.client.apis.CoreV1Api
import io.kubernetes.client.apis.ExtensionsV1beta1Api
import io.kubernetes.client.models.V1ObjectMeta
import io.kubernetes.client.models.V1Namespace
import io.kubernetes.client.models.ExtensionsV1beta1Deployment
import io.kubernetes.client.models.ExtensionsV1beta1DeploymentSpec
import io.kubernetes.client.models.V1ContainerPort
import io.kubernetes.client.models.V1PodTemplateSpec
import io.kubernetes.client.models.V1PodSpec
import io.kubernetes.client.models.V1Container
import io.kubernetes.client.models.V1DeleteOptions

import io.kubernetes.client.util.Config
import objects.Deployment

import java.util.stream.Collectors

class Kubernetes extends OrchestratorCommon implements OrchestratorMain {
    private final String namespace

    private CoreV1Api api
    private ExtensionsV1beta1Api beta1

    Kubernetes(String ns) {
        this.namespace = ns
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

    def setup() {
        ApiClient client = Config.defaultClient()
        Configuration.setDefaultApiClient(client)

        this.api = new CoreV1Api()
        this.beta1 = new ExtensionsV1beta1Api()

        ensureNamespaceExists()
    }

    @Override
    def cleanup() {
    }

    def portToContainerPort = {  p -> new V1ContainerPort().containerPort(p) }

    String createDeployment(Deployment deployment) {
        List<V1ContainerPort> containerPorts = deployment.getPorts().stream()
                .map(portToContainerPort)
                .collect(Collectors.<V1ContainerPort> toList())

        ExtensionsV1beta1Deployment k8sDeployment = new ExtensionsV1beta1Deployment()
        .metadata(
                new V1ObjectMeta()
                        .name(deployment.getName())
                        .namespace(this.namespace)
                        .labels(deployment.getLabels()))
        .spec(new ExtensionsV1beta1DeploymentSpec()
                .replicas(1)
                .template(new V1PodTemplateSpec()
                    .spec(new V1PodSpec()
                        .containers(
                            [
                                    new V1Container()
                                            .name(deployment.getName())
                                            .image(deployment.getImage())
                                            .ports(containerPorts),
                            ]
                        )
                    )
                    .metadata(new V1ObjectMeta()
                        .name(deployment.getName())
                        .labels(deployment.getLabels())
                    )
                )
        )

        beta1.createNamespacedDeployment(this.namespace, k8sDeployment, null)
    }

    def deleteDeployment(String name) {
        this.beta1.deleteNamespacedDeployment(
                name,
                this.namespace, new V1DeleteOptions()
                    .gracePeriodSeconds(0)
                    .orphanDependents(false),
                null,
                0,
                false,
                null)
        println "Deployment removed."
    }

}
