package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.ProcessesListeningOnPortsServiceGrpc
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsResponse
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsWithDeploymentResponse
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsByNamespaceRequest
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsByNamespaceAndDeploymentRequest

@Slf4j
class ProcessesListeningOnPortsService extends BaseService {
    static getProcessesListeningOnPortsService() {
        return ProcessesListeningOnPortsServiceGrpc.newBlockingStub(getChannel())
    }

    static GetProcessesListeningOnPortsWithDeploymentResponse getProcessesListeningOnPortsWithDeploymentResponse(
        String namespace) {

        GetProcessesListeningOnPortsByNamespaceRequest request =
                GetProcessesListeningOnPortsByNamespaceRequest.newBuilder()
                        .setNamespace(namespace)
                        .build()

        def processesListeningOnPorts = getProcessesListeningOnPortsService()
                        .getProcessesListeningOnPortsByNamespace(request)

        return processesListeningOnPorts
    }

    static GetProcessesListeningOnPortsResponse getProcessesListeningOnPortsResponse(
        String namespace, String deploymentId) {

        GetProcessesListeningOnPortsByNamespaceAndDeploymentRequest request =
                GetProcessesListeningOnPortsByNamespaceAndDeploymentRequest.newBuilder()
                        .setNamespace(namespace)
                        .setDeploymentId(deploymentId)
                        .build()

        def processesListeningOnPorts = getProcessesListeningOnPortsService()
                        .getProcessesListeningOnPortsByNamespaceAndDeployment(request)

        return processesListeningOnPorts
    }
}
