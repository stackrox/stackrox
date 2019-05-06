package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.RbacServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import objects.K8sRole
import util.Timer

class RbacService extends BaseService {
    static getRbacService() {
        return RbacServiceGrpc.newBlockingStub(getChannel())
    }

    static getRoles(SearchServiceOuterClass.RawQuery query = SearchServiceOuterClass.RawQuery.newBuilder().build()) {
        return getRbacService().listRoles(query).rolesList
    }

    static getRole(String id) {
        try {
            return getRbacService().getRole(
                    Common.ResourceByID.newBuilder().setId(id).build()
            ).role
        } catch (Exception e) {
            println "Error fetching role: ${e.toString()}"
        }
    }

    static boolean waitForRole(K8sRole role) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            println "Waiting for ${role.clusterRole ? "Cluster " : ""}Role"
            def roles = getRoles()
            def r = roles.find {
                it.name == role.name &&
                        it.namespace == role.namespace
            }

            if (r) {
                return true
            }
        }
        println "Time out for Waiting for ${role.clusterRole ? "Cluster " : ""}Role"
        return false
    }

    static boolean waitForRoleRemoved(K8sRole role) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            println "Waiting for ${role.clusterRole ? "Cluster " : ""}Role removed"
            def roles = getRoles()
            def r = roles.find {
                it.name == role.name &&
                        it.namespace == role.namespace
            }
            if (!r) {
                return true
            }
        }
        println "Time out for Waiting for ${role.clusterRole ? "Cluster " : ""}Role removal"
        return false
    }
}
