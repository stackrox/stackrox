package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.ProcessesListeningOnPortsServiceGrpc
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsResponse
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsRequest

@Slf4j
class ProcessesListeningOnPortsService extends BaseService {
    static getProcessesListeningOnPortsService() {
        return ProcessesListeningOnPortsServiceGrpc.newBlockingStub(getChannel())
    }

    static GetProcessesListeningOnPortsResponse getProcessesListeningOnPortsResponse(
        String deploymentId) {

        GetProcessesListeningOnPortsRequest request =
                GetProcessesListeningOnPortsRequest.newBuilder()
                        .setDeploymentId(deploymentId)
                        .build()

        def processesListeningOnPorts = getProcessesListeningOnPortsService()
                        .getProcessesListeningOnPorts(request)

        return processesListeningOnPorts
    }
}
