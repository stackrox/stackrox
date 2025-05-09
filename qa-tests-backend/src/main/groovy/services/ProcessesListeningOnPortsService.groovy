package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v1.PaginationOuterClass
import io.stackrox.proto.api.v1.ListeningEndpointsServiceGrpc
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsRequest
import io.stackrox.proto.api.v1.ProcessListeningOnPortService.GetProcessesListeningOnPortsResponse

import objects.Pagination

@Slf4j
@CompileStatic
class ProcessesListeningOnPortsService extends BaseService {
    static ListeningEndpointsServiceGrpc.ListeningEndpointsServiceBlockingStub getProcessesListeningOnPortsService() {
        return ListeningEndpointsServiceGrpc.newBlockingStub(getChannel())
    }

    static GetProcessesListeningOnPortsResponse getProcessesListeningOnPortsResponse(
        String deploymentId, Pagination pagination = null) {

        GetProcessesListeningOnPortsRequest.Builder request =
                GetProcessesListeningOnPortsRequest.newBuilder()
                        .setDeploymentId(deploymentId)

        if (pagination != null) {
           log.info "pagination.offset= ${pagination.offset}"
           log.info "pagination.limit= ${pagination.limit}"
           PaginationOuterClass.Pagination.Builder pbuilder =
               PaginationOuterClass.Pagination.newBuilder()
                   .setOffset(pagination.offset)
                   .setLimit(pagination.limit)
               request.setPagination(pbuilder.build())
        }

        log.info "request= ${request}"

        def processesListeningOnPorts = getProcessesListeningOnPortsService()
                        .getListeningEndpoints(request.build())

        return processesListeningOnPorts
    }
}
