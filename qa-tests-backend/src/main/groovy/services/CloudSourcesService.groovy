package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j
import io.grpc.Status
import io.grpc.StatusRuntimeException

import io.stackrox.proto.api.v1.CloudSourceService
import io.stackrox.proto.api.v1.CloudSourcesServiceGrpc

@CompileStatic
@Slf4j
class CloudSourcesService extends BaseService {
    static CloudSourcesServiceGrpc.CloudSourcesServiceBlockingStub getCloudSourcesClient() {
        return CloudSourcesServiceGrpc.newBlockingStub(getChannel())
    }

    static String createCloudSource(CloudSourceService.CloudSource cloudSource) {
        CloudSourceService.CloudSource createdCloudSource
        try {
            createdCloudSource = getCloudSourcesClient().createCloudSource(CloudSourceService.
                    CreateCloudSourceRequest.newBuilder().
                    setCloudSource(cloudSource as CloudSourceService.CloudSource).build()).
                    getCloudSource()
        } catch (Exception e) {
            log.error("Unable to create cloud source ${cloudSource.getName()}", e)
        }
        if (!createdCloudSource || !createdCloudSource.getId()) {
            return ""
        }

        CloudSourceService.CloudSource foundCloudSource
        try {
            foundCloudSource = getCloudSourcesClient().
                    getCloudSource(CloudSourceService.
                            GetCloudSourceRequest.newBuilder().
                            setId(createdCloudSource.getId()).build()).
                    getCloudSource()
        } catch (Exception e) {
            log.error("Unable to find the created cloud source ${cloudSource.getName()}", e)
        }

        if (foundCloudSource) {
            return foundCloudSource.getId()
        }
        return ""
    }

    static Boolean deleteCloudSource(String cloudSourceId) {
        try {
            getCloudSourcesClient().deleteCloudSource(CloudSourceService.DeleteCloudSourceRequest.
                    newBuilder().setId(cloudSourceId).build())
        } catch (Exception e) {
            log.error("Failed to delete cloud source with id ${cloudSourceId}", e)
        }

        try {
            getCloudSourcesClient().
                    getCloudSource(CloudSourceService.
                            GetCloudSourceRequest.newBuilder().
                            setId(cloudSourceId).build())
        } catch (StatusRuntimeException e) {
            if (e.status.code == Status.Code.NOT_FOUND) {
                log.info "Cloud source deleted: ${cloudSourceId}"
                return true
            }
            log.error("Error retrieving deleted cloud source", e)
        }
        return false
    }
}
