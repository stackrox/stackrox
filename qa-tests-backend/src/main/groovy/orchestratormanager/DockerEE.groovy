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

import objects.Deployment

import java.util.stream.Collectors

class DockerEE extends OrchestratorCommon implements OrchestratorMain {
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
    def deleteDeployment(String deploymentName, String namespace = "") {
        docker.removeServiceCmd(deploymentName).exec()
        println "Service removed."
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
}
