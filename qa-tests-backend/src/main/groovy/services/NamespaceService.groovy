package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.NamespaceServiceGrpc
import io.stackrox.proto.api.v1.NamespaceServiceOuterClass.Namespace
import util.Timer

class NamespaceService extends BaseService {

    static getNamespaceClient() {
        return NamespaceServiceGrpc.newBlockingStub(getChannel())
    }

    static List<Namespace> getNamespaces() {
        return getNamespaceClient().getNamespaces(EMPTY).namespacesList
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
        int retries = timeoutSeconds / intervalSeconds
        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            if (getNamespace(id) != null ) {
                println "SR found namespace ${id} within ${t.SecondsSince()}s"
                return true
            }
            println "Retrying in ${intervalSeconds}..."
        }
        println "SR did not detect the namespace ${id}"
        return false
    }

}
