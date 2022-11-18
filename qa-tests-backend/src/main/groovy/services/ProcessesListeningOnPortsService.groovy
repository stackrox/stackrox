package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.ProcessListeningOnPortServiceGrpc
import io.stackrox.proto.api.v1.ProcessListeningOnPortServiceOuterClass.GetProcessesListeningOnPortsResponse
import io.stackrox.proto.api.v1.ProcessListeningOnPortServiceOuterClass.GetProcessesListeningOnPortsWithDeploymentResponse
import io.stackrox.proto.api.v1.ProcessListeningOnPortServiceOuterClass.GetProcessesListeningOnPortsByNamespaceRequest
import io.stackrox.proto.api.v1.ProcessListeningOnPortServiceOuterClass.GetProcessesListeningOnPortsByNamespaceAndDeploymentRequest

@Slf4j
class ProcessesListeningOnPortsService extends BaseService {
    static getProcessesListeningOnPortsService() {
        return ProcessListeningOnPortServiceGrpc.newBlockingStub(getChannel())
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
