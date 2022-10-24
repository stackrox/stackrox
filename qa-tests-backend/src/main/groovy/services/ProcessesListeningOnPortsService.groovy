package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.ProcessesListeningOnPortsServiceGrpc
import io.stackrox.proto.api.v1.ProcessesListeningOnPortsServiceOuterClass.GetProcessesListeningOnPortsResponse
import io.stackrox.proto.api.v1.ProcessesListeningOnPortsServiceOuterClass.GetProcessesListeningOnPortsWithDeploymentResponse
import io.stackrox.proto.api.v1.ProcessesListeningOnPortsServiceOuterClass.GetProcessesListeningOnPortsByNamespaceRequest
import io.stackrox.proto.api.v1.ProcessesListeningOnPortsServiceOuterClass.GetProcessesListeningOnPortsByNamespaceAndDeploymentRequest

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
