package services

import static io.stackrox.proto.api.v1.RbacServiceOuterClass.SubjectAndRoles
import static io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import static io.stackrox.proto.storage.Rbac.Subject

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.RbacServiceGrpc
import io.stackrox.proto.storage.Rbac

import objects.K8sRole
import objects.K8sRoleBinding
import util.Timer

@Slf4j
@CompileStatic
class RbacService extends BaseService {
    static RbacServiceGrpc.RbacServiceBlockingStub getRbacService() {
        return RbacServiceGrpc.newBlockingStub(getChannel())
    }

    static List<Rbac.K8sRole> getRoles(RawQuery query = RawQuery.newBuilder().build()) {
        return getRbacService().listRoles(query).rolesList
    }

    static Rbac.K8sRole getRole(String id) {
        return getRbacService().getRole(
                Common.ResourceByID.newBuilder().setId(id).build()
        ).role
    }

    static boolean waitForRole(K8sRole role) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            log.debug "Waiting for Role"
            def roles = getRoles()
            def r = roles.find {
                it.name == role.name &&
                        it.namespace == role.namespace
            }

            if (r) {
                return true
            }
        }
        log.warn "Time out for Waiting for ${role.name} Role"
        return false
    }

    static boolean waitForRoleRemoved(K8sRole role) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            log.debug "Waiting for Role removed"
            def roles = getRoles()
            def r = roles.find {
                it.name == role.name &&
                        it.namespace == role.namespace
            }
            if (!r) {
                return true
            }
        }
        log.warn "Time out for Waiting for Role removal"
        return false
    }

    static List<Rbac.K8sRoleBinding> getRoleBindings(RawQuery query = RawQuery.newBuilder().build()) {
        log.debug("Get bindings list: ${query}")
        return getRbacService().listRoleBindings(query).bindingsList
    }

    static boolean waitForRoleBinding(K8sRoleBinding roleBinding) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            log.debug "Waiting for Role Binding"
            def roleBindings = getRoleBindings()
            def r = roleBindings.find {
                it.name == roleBinding.name &&
                        it.namespace == roleBinding.namespace
            }

            if (r) {
                return true
            }
        }
        log.warn "Time out for Waiting for Role Binding"
        return false
    }

    static boolean waitForRoleBindingRemoved(K8sRoleBinding roleBinding) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            log.debug "Waiting for Role Binding removed"
            def roleBindings = getRoleBindings()
            def r = roleBindings.find {
                it.name == roleBinding.name &&
                        it.namespace == roleBinding.namespace
            }
            if (!r) {
                return true
            }
        }
        log.warn "Time out for Waiting for Role Binding removal"
        return false
    }

    static List<SubjectAndRoles> getSubjects(
            RawQuery query = RawQuery.newBuilder().build()) {
        return getRbacService().listSubjects(query).subjectAndRolesList
    }

    static Subject getSubject(String id) {
        return getRbacService().getSubject(
                Common.ResourceByID.newBuilder().setId(id).build()
        ).subject
    }
}
