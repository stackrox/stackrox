package services

import groovy.util.logging.Slf4j
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

    static deleteRole(String name) {
        try {
            def role = getRole(name)
            getRoleService().deleteRole(Common.ResourceByID.newBuilder().setId(name).build())
            deletePermissionSet(role.permissionSetId)
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

