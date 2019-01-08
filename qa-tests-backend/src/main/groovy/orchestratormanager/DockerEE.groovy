package orchestratormanager

import com.github.dockerjava.api.DockerClient
import com.github.dockerjava.api.command.CreateServiceResponse
import com.github.dockerjava.api.model.PortConfig
import com.github.dockerjava.api.model.ServiceSpec
import com.github.dockerjava.api.model.EndpointSpec
import com.github.dockerjava.api.model.TaskSpec
import com.github.dockerjava.api.model.ContainerSpec
import com.github.dockerjava.api.model.PortConfigProtocol
import com.github.dockerjava.core.DefaultDockerClientConfig
import com.github.dockerjava.core.DockerClientBuilder
import com.github.dockerjava.core.DockerClientConfig
import objects.DaemonSet
import objects.Deployment
import objects.NetworkPolicy

import java.util.stream.Collectors

class DockerEE extends OrchestratorCommon {
    DockerClient docker

    final String dockerHost = System.getenv("DOCKER_HOST")
    final String dockerCertPath = System.getenv("DOCKER_CERT_PATH")
    final String apiVersion = System.getenv("DOCKER_API_VERSION")
    final String dockerUsername = "qa"
    final String dockerPassword = "W3g9xOPKyLTkBBMj"
    final String registryUrl = "https://apollo-dtr.rox.systems/"

    DockerEE() {
        DockerClientConfig config = DefaultDockerClientConfig.createDefaultConfigBuilder()
                .withDockerHost(dockerHost)
                .withDockerTlsVerify(true)
                .withDockerCertPath(dockerCertPath)
                .withApiVersion(apiVersion)
                .withRegistryUsername(dockerUsername)
                .withRegistryPassword(dockerPassword)
                .withRegistryUrl(registryUrl)
                .build()
        this.docker = DockerClientBuilder.getInstance(config).build()
    }

    def setup() { }
    def cleanup() { }

    def portToPortConfig = {
        p -> new PortConfig()
                .withPublishedPort(p)
                .withProtocol(PortConfigProtocol.TCP)
                .withTargetPort(p)
                .withPublishMode(PortConfig.PublishMode.ingress)
    }

    def batchCreateDeployments(List<Deployment> deployments) {
        for (Deployment deployment : deployments) {
            createDeployment(deployment)
        }
    }

    def createDeployment(Deployment deployment) {
        List<PortConfig> containerPorts = deployment.getPorts().stream()
                .map(portToPortConfig)
                .collect(Collectors.<PortConfig> toList())

        CreateServiceResponse resp = docker.createServiceCmd(
            new ServiceSpec()
                    .withEndpointSpec(
                        new EndpointSpec()
                            .withPorts(containerPorts)
                    )
                    .withName(deployment.getName())
                    .withLabels(deployment.getLabels())
                    .withTaskTemplate(
                        new TaskSpec()
                            .withContainerSpec(
                                new ContainerSpec()
                                    .withLabels(deployment.getLabels())
                                    .withImage(deployment.getImage())
                            )
                    )
        ).exec()
        println resp
    }

    @Override
    def deleteDeployment(Deployment deployment) {
        docker.removeServiceCmd(deployment.name).exec()
        println "Service removed."
    }

    @Override
    def createDaemonSet(DaemonSet daemonSet) {
    }

    @Override
    def deleteDaemonSet(DaemonSet daemonSet) {
    }

    @Override
    def getDaemonSetReplicaCount(DaemonSet daemonSet) {
    }

    @Override
    def getDaemonSetNodeSelectors(DaemonSet daemonSet) {
    }

    @Override
    def getDaemonSetUnavailableReplicaCount(DaemonSet daemonSet) {
    }

    @Override
    def deleteService(String serviceName, String namespace = "") {
    }

    @Override
    def createClairifyDeployment() {
    }

    String getClairifyEndpoint() {
        return "clairify.prevent_net:8080"
    }

    @Override
    def createSecret(String name) {
    }

    @Override
    def deleteSecret(String name, String namespace = "") {
    }

    @Override
    String applyNetworkPolicy(NetworkPolicy policy) {
    }

    @Override
    boolean deleteNetworkPolicy(NetworkPolicy policy) {
    }

    @Override
    String generateYaml(Object orchestratorObject) {
    }

    @Override
    String getDeploymentId(Deployment deployment) {
    }

    @Override
    String getpods() {
    }

    @Override
    def wasContainerKilled(String containerName, String namespace = "") {
    }

    @Override
    def getDeploymentReplicaCount(Deployment deployment) {
    }

    @Override
    def getDeploymentUnavailableReplicaCount(Deployment deployment) {
    }

    @Override
    def getDeploymentNodeSelectors(Deployment deployment) {
    }

    @Override
    def getNodeCount() {
    }

    @Override
    def getNameSpace() {
    }

    @Override
    def getDeploymentCount() {
    }

    @Override
    def getSecretCount() {
    }

    @Override
    def getDaemonSetCount() {
    }

    @Override
    def isKubeProxyPresent() {
    }

    @Override
    def supportsNetworkPolicies() {
    }

    @Override
    def isKubeDashboardRunning() {
    }

    @Override
    def getContainerlogs(Deployment deployment) {
    }
}
