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

    DockerEE() {
        DockerClientConfig config = DefaultDockerClientConfig.createDefaultConfigBuilder()
                .withApiVersion("1.23")
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

    def deleteDeployment(String deploymentName) {
        docker.removeServiceCmd(deploymentName).exec()
        println "Service removed."
    }

}
