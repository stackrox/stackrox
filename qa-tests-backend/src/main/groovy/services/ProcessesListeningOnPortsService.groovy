package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v1.ListeningEndpointsServiceGrpc
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsRequest
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsResponse

@Slf4j
@CompileStatic
class ProcessesListeningOnPortsService extends BaseService {
    static ListeningEndpointsServiceGrpc.ListeningEndpointsServiceBlockingStub getProcessesListeningOnPortsService() {
        return ListeningEndpointsServiceGrpc.newBlockingStub(getChannel())
    }

    static GetProcessesListeningOnPortsResponse getProcessesListeningOnPortsResponse(
        String deploymentId) {

        GetProcessesListeningOnPortsRequest request =
                GetProcessesListeningOnPortsRequest.newBuilder()
                        .setDeploymentId(deploymentId)
                        .build()

        def processesListeningOnPorts = getProcessesListeningOnPortsService()
                        .getListeningEndpoints(request)

        return processesListeningOnPorts
    }
}
