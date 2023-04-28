package services

import groovy.util.logging.Slf4j
import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.RoleServiceGrpc
import io.stackrox.proto.api.v1.RoleServiceOuterClass
import io.stackrox.proto.storage.RoleOuterClass
import io.stackrox.proto.storage.RoleOuterClass.Role

@Slf4j
class RoleService extends BaseService {
    static getRoleService() {
        return RoleServiceGrpc.newBlockingStub(getChannel())
    }

    static getRoles() {
        return getRoleService().getRoles(EMPTY)
    }

    static getRole(String roleId) {
        return getRoleService().getRole(Common.ResourceByID.newBuilder().setId(roleId).build())
    }

    static Boolean checkRoleExists(String roleId) {
        try {
            getRoleService().getRole(Common.ResourceByID.newBuilder().setId(roleId).build())
        } catch (StatusRuntimeException e) {
            if (e.status.code == Status.Code.NOT_FOUND) {
                return false
            }
            throw e
        }
        return true
    }

    static RoleServiceOuterClass.GetResourcesResponse getResources() {
        try {
            return getRoleService().getResources(EMPTY)
        } catch (Exception e) {
            log.warn("Failed to fetch resources", e)
        }
    }

    static Role createRoleWithScopeAndPermissionSet(String name, String accessScopeId,
        Map<String, RoleOuterClass.Access> resourceToAccess) {

        def permissionSet = createPermissionSet(
                "Test Automation Permission Set ${UUID.randomUUID()} for ${name}", resourceToAccess)
        Role role = Role.newBuilder()
            .setName(name)
            .setAccessScopeId(accessScopeId)
            .setPermissionSetId(permissionSet.id)
            .build()
        getRoleService().createRole(RoleServiceOuterClass.CreateRoleRequest
                .newBuilder()
                .setName(role.name)
                .setRole(role)
                .build()
        )
        role
    }

    static Role createRole(Role role) {
        getRoleService().createRole(RoleServiceOuterClass.CreateRoleRequest
                .newBuilder()
                .setName(role.name)
                .setRole(role)
                .build()
        )
        log.info "Created role: ${role.name}"
        role
    }

    static deleteRole(String name) {
        try {
            def role = getRole(name)
            getRoleService().deleteRole(Common.ResourceByID.newBuilder().setId(name).build())
            deletePermissionSet(role.permissionSetId)
            log.info "Deleted role: ${name} and permission set"
        } catch (Exception e) {
            log.warn("Error deleting role ${name} or permission set", e)
        }
    }

    static deleteRoleWithoutPermissionSet(String name, Boolean alsoDeletePermissionSet = true) {
        try {
            getRoleService().deleteRole(Common.ResourceByID.newBuilder().setId(name).build())
            log.info "Deleted role: ${name}"
        } catch (Exception e) {
            log.warn("Error deleting role ${name}", e)
        }
    }

    static createPermissionSet(String name, Map<String, RoleOuterClass.Access> resources) {
        getRoleService().postPermissionSet(RoleOuterClass.PermissionSet.newBuilder()
                .setName(name)
                .putAllResourceToAccess(resources).build())
    }

    static deletePermissionSet(String id) {
        try {
            getRoleService().deletePermissionSet(Common.ResourceByID.newBuilder().setId(id).build())
        } catch (Exception e) {
            log.warn("Error deleting permission set ${id}", e)
        }
    }

    static RoleOuterClass.SimpleAccessScope createAccessScope(RoleOuterClass.SimpleAccessScope accessScope) {
        return getRoleService().postSimpleAccessScope(accessScope)
    }

    static deleteAccessScope(String id) {
        getRoleService().deleteSimpleAccessScope(Common.ResourceByID.newBuilder().setId(id).build())
    }

    static myPermissions() {
        getRoleService().getMyPermissions(EMPTY)
    }
}

