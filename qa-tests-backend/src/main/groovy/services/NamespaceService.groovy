package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.EmptyOuterClass.Empty
import io.stackrox.proto.api.v1.NamespaceServiceGrpc
import io.stackrox.proto.api.v1.NamespaceServiceOuterClass.Namespace

class NamespaceService extends BaseService {

    static getNamespaceClient() {
        return NamespaceServiceGrpc.newBlockingStub(getChannel())
    }

    static List<Namespace> getNamespaces() {
        return getNamespaceClient().getNamespaces(Empty.newBuilder().build()).namespacesList
    }
    static Namespace getNamespace(String id) {
        try {
            return getNamespaceClient().getNamespace(Common.ResourceByID.newBuilder().setId(id).build())
        } catch (Exception e) {
            println "Could not find namespace ${id}: ${e.message}"
        }
        return null
    }

    static waitForNamespace(String id, int timeoutSeconds = 10) {
        int intervalSeconds = 1
        def startTime = System.currentTimeMillis()
        for (int waitTime = 0; waitTime < timeoutSeconds / intervalSeconds; waitTime++) {
            if (getNamespace(id) != null ) {
                println "SR found namespace ${id} within ${(System.currentTimeMillis() - startTime) / 1000}s"
                return true
            }
            println "Retrying in ${intervalSeconds}..."
            sleep(intervalSeconds * 1000)
        }
        println "SR did not detect the namespace ${id}"
        return false
    }

}
