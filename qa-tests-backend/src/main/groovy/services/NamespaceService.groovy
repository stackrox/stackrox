package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.NamespaceServiceGrpc
import io.stackrox.proto.api.v1.NamespaceServiceOuterClass.GetNamespaceRequest
import io.stackrox.proto.api.v1.NamespaceServiceOuterClass.Namespace

import util.Timer

@Slf4j
@CompileStatic
class NamespaceService extends BaseService {

    static NamespaceServiceGrpc.NamespaceServiceBlockingStub getNamespaceClient() {
        return NamespaceServiceGrpc.newBlockingStub(getChannel())
    }

    static List<Namespace> getNamespaces() {
        return getNamespaceClient().getNamespaces(GetNamespaceRequest.newBuilder().build()).namespacesList
    }
    static Namespace getNamespace(String id) {
        try {
            return getNamespaceClient().getNamespace(Common.ResourceByID.newBuilder().setId(id).build())
        } catch (Exception e) {
            log.error("Could not find namespace ${id}", e)
        }
        return null
    }
    static waitForNamespace(String id, int timeoutSeconds = 10) {
        int intervalSeconds = 1
        int retries = (timeoutSeconds / intervalSeconds) as int
        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            if (getNamespace(id) != null ) {
                log.debug "SR found namespace ${id} within ${t.SecondsSince()}s"
                return true
            }
            log.debug "Retrying in ${intervalSeconds}..."
        }
        log.warn "SR did not detect the namespace ${id}"
        return false
    }

}
