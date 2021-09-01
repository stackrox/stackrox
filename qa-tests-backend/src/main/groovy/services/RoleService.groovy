package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.RoleServiceGrpc
import io.stackrox.proto.api.v1.RoleServiceOuterClass
import io.stackrox.proto.storage.RoleOuterClass
import io.stackrox.proto.storage.RoleOuterClass.Role

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
            println "Failed to fetch resources: ${e}"
        }
    }

    static Role createRole(Role role) {
        Role r = role
        if (role.permissionSetId == "" &&
                FeatureFlagService.isFeatureFlagEnabled('ROX_SCOPED_ACCESS_CONTROL_V2')) {
            def permissionSet = createPermissionSet(
                    "Test Automation Permission Set ${UUID.randomUUID()} for ${role.name}", role.resourceToAccess)
            r = Role.newBuilder(role)
                    .clearResourceToAccess()
                    .setPermissionSetId(permissionSet.id).build()
        }
        getRoleService().createRole(RoleServiceOuterClass.CreateRoleRequest
                .newBuilder()
                .setName(r.name)
                .setRole(r)
                .build()
        )
        r
    }

    static deleteRole(String name) {
        try {
            if (FeatureFlagService.isFeatureFlagEnabled('ROX_SCOPED_ACCESS_CONTROL_V2')) {
                def role = getRole(name)
                getRoleService().deleteRole(Common.ResourceByID.newBuilder().setId(name).build())
                deletePermissionSet(role.permissionSetId)
            } else {
                getRoleService().deleteRole(Common.ResourceByID.newBuilder().setId(name).build())
            }
        } catch (Exception e) {
            println "Error deleting role ${name}: ${e}"
        }
    }

    static createPermissionSet(String name, Map<String, RoleOuterClass.Access> resourceAccess) {
        getRoleService().postPermissionSet(RoleOuterClass.PermissionSet.newBuilder()
                .setName(name)
                .putAllResourceToAccess(resourceAccess).build())
    }

    static deletePermissionSet(String id) {
        try {
            getRoleService().deletePermissionSet(Common.ResourceByID.newBuilder().setId(id).build())
        } catch (Exception e) {
            println "Error deleting permission set ${id}: ${e}"
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

