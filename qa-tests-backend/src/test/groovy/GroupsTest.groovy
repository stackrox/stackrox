import io.grpc.Status
import io.grpc.StatusRuntimeException

import io.stackrox.proto.api.v1.GroupServiceOuterClass
import io.stackrox.proto.api.v1.GroupServiceOuterClass.GetGroupsRequest
import io.stackrox.proto.storage.AuthProviderOuterClass
import io.stackrox.proto.storage.GroupOuterClass.Group
import io.stackrox.proto.storage.GroupOuterClass.GroupProperties
import io.stackrox.proto.storage.RoleOuterClass

import services.AuthProviderService
import services.GroupService
import services.RoleService

import spock.lang.Tag
import spock.lang.Unroll

@Tag("BAT")

class GroupsTest extends BaseSpecification {

    private static final PROVIDERS = [
            AuthProviderOuterClass.AuthProvider.newBuilder()
                    .setName("groups-test-provider-1")
                    .setType("iap")
                    .putConfig("audience", "test-audience")
                    .build(),
            AuthProviderOuterClass.AuthProvider.newBuilder()
                    .setName("groups-test-provider-2")
                    .setType("iap")
                    .putConfig("audience", "test-audience")
                    .build(),
    ]

    private static final Map<String, String> PROVIDER_IDS_BY_NAME = [:]

    private static final Map<Group, String> GROUPS_TO_AUTH_PROVIDER = [
            (Group.newBuilder()
                    .setRoleName("QAGroupTest-Group1")
                    .build()): PROVIDERS[0].getName(),
            (Group.newBuilder()
                    .setRoleName("QAGroupTest-Group2")
                    .setProps(GroupProperties.newBuilder()
                            .setKey("foo")
                            .setValue("bar")
                            .build())
                    .build()): PROVIDERS[0].getName(),
            (Group.newBuilder()
                    .setRoleName("QAGroupTest-Group3")
                    .setProps(GroupProperties.newBuilder()
                            .setKey("foo")
                            .setValue("bar")
                            .build())
                    .build()): PROVIDERS[1].getName(),
    ]

    private static final ROLES = [
            RoleOuterClass.Role.newBuilder()
                    .setName("QAGroupTest-Group1")
                    .setAccessScopeId("ffffffff-ffff-fff4-f5ff-ffffffffffff")
                    .setPermissionSetId("ffffffff-ffff-fff4-f5ff-ffffffffffff")
                    .build(),
            RoleOuterClass.Role.newBuilder()
                    .setName("QAGroupTest-Group2")
                    .setAccessScopeId("ffffffff-ffff-fff4-f5ff-ffffffffffff")
                    .setPermissionSetId("ffffffff-ffff-fff4-f5ff-ffffffffffff")
                    .build(),
            RoleOuterClass.Role.newBuilder()
                    .setName("QAGroupTest-Group3")
                    .setAccessScopeId("ffffffff-ffff-fff4-f5ff-ffffffffffff")
                    .setPermissionSetId("ffffffff-ffff-fff4-f5ff-ffffffffffff")
                    .build(),
    ]

    private static final Map<String, Group> GROUPS_WITH_IDS = [:]

    def setupSpec() {
        for (def role : ROLES) {
            RoleService.createRole(role)
        }
        for (def provider : PROVIDERS) {
            def authProviderId = AuthProviderService.createAuthProvider(provider.getName(), provider.getType(),
                    provider.getConfigMap())
            PROVIDER_IDS_BY_NAME[provider.getName()] = authProviderId
        }
        GROUPS_TO_AUTH_PROVIDER.each { group, authProviderName ->
            def props = group.toBuilder()
                .getPropsBuilder()
                .setKey(group.getProps().getKey())
                .setValue(group.getProps().getValue())
                .setAuthProviderId(PROVIDER_IDS_BY_NAME[authProviderName])
                .build()
            GroupService.createGroup(group
                    .toBuilder()
                    .setProps(props)
                    .build())
            def groupWithId = GroupService.getGroups(GetGroupsRequest.newBuilder()
                    .setAuthProviderId(props.getAuthProviderId())
                    .setValue(props.getValue())
                    .setKey(props.getKey()).build()
            ).getGroups(0)
            GROUPS_WITH_IDS[groupWithId.roleName] = groupWithId
        }
    }

    def cleanupSpec() {
        GROUPS_WITH_IDS.values().flatten().each { group ->
            try {
                GroupService.deleteGroup(group.props)
            } catch (Exception ex) {
                log.warn("Failed to delete group", ex)
            }
        }
        PROVIDER_IDS_BY_NAME.values().flatten().each { authProviderId ->
            try {
                AuthProviderService.deleteAuthProvider(authProviderId)
            } catch (Exception ex) {
                log.warn("Failed to delete auth provider", ex)
            }
        }
        for (def role : ROLES) {
            RoleService.deleteRoleWithoutPermissionSet(role.getName())
        }
    }

    def "Test that creating group with invalid role name returns an error"() {
        when:
        def props = GroupProperties.newBuilder()
                .setAuthProviderId(PROVIDER_IDS_BY_NAME["groups-test-provider-1"])
                .setKey("this is so that group")
                .setValue("will be non-default and we get invalid arguments error")
                .build()
        GroupService.getGroupService().createGroup(Group.newBuilder()
                .setProps(props)
                .setRoleName("non-existent")
                .build())
        then:
        def error = thrown(StatusRuntimeException)
        assert error.getStatus().getCode() == Status.Code.INVALID_ARGUMENT
    }

    def "Test that creating group with invalid auth provider id returns an error"() {
        when:
        def props = GroupProperties.newBuilder()
                .setAuthProviderId("non-existent-provider-id")
                .build()
        GroupService.getGroupService().createGroup(Group.newBuilder()
                .setProps(props)
                .setRoleName("Admin")
                .build())
        then:
        def error = thrown(StatusRuntimeException)
        assert error.getStatus().getCode() == Status.Code.INVALID_ARGUMENT
    }

    def "Test that updating group with invalid role name returns an error"() {
        when:
        def group = GROUPS_WITH_IDS["QAGroupTest-Group2"]
        def updatedGroup = group.toBuilder()
                .setRoleName("non-existent")
                .build()
        GroupService.getGroupService().updateGroup(
                GroupServiceOuterClass.UpdateGroupRequest.newBuilder()
                        .setGroup(updatedGroup)
                        .build()
        )
        then:
        def error = thrown(StatusRuntimeException)
        assert error.getStatus().getCode() == Status.Code.INVALID_ARGUMENT
    }

    def "Test that updating group with invalid auth provider id returns an error"() {
        when:
        def group = GROUPS_WITH_IDS["QAGroupTest-Group2"]
        def updatedGroup = group.toBuilder()
            .setProps(group.getProps().toBuilder()
                    .setAuthProviderId("non-existent")
                    .build())
            .build()
        GroupService.getGroupService().updateGroup(
                GroupServiceOuterClass.UpdateGroupRequest.newBuilder()
                        .setGroup(updatedGroup)
                        .build()
        )
        then:
        def error = thrown(StatusRuntimeException)
        assert error.getStatus().getCode() == Status.Code.INVALID_ARGUMENT
    }

    @Unroll
    @SuppressWarnings('LineLength')
    def "Test that GetGroup and GetGroups work correctly with query args (#authProviderName, #key, #value)"() {
        when:
        "A query is made for GetGroup and GetGroups with the given params"
        def propsBuilder = GroupProperties.newBuilder()
        def reqBuilder = GetGroupsRequest.newBuilder()
        if (authProviderName != null) {
            propsBuilder.setAuthProviderId(PROVIDER_IDS_BY_NAME[authProviderName])
            reqBuilder.setAuthProviderId(PROVIDER_IDS_BY_NAME[authProviderName])
        }
        if (key != null) {
            propsBuilder.setKey(key)
            reqBuilder.setKey(key)
        }
        if (value != null) {
            propsBuilder.setValue(value)
            reqBuilder.setValue(value)
        }
        if (id != null) {
            propsBuilder.setId(id)
        }

        String matchedGroup = null
        try {
            def grp = GroupService.getGroup(propsBuilder.build())
            if (grp.roleName.startsWith("QAGroupTest-")) {
                matchedGroup = grp.roleName["QAGroupTest-".length()..-1]
            }
        } catch (StatusRuntimeException ex) {
            if (ex.status.code != Status.Code.NOT_FOUND &&
                    (authProviderName == null && ex.status.code != Status.Code.INVALID_ARGUMENT)) {
                throw ex
            }
        }
        def matchedGroups = GroupService.getGroups(reqBuilder.build()).groupsList*.roleName.collectMany {
            return it.startsWith("QAGroupTest-") ? [it["QAGroupTest-".length()..-1]] : []
        }.sort()

        then:
        "Results should match the expected data"
        assert expectGroup == matchedGroup
        assert expectGroups == matchedGroups

        where:
        "Data inputs are"
        authProviderName | key | id | value | expectGroup | expectGroups
        "groups-test-provider-1" | null  | GROUPS_WITH_IDS["QAGroupTest-Group1"].props.getId() | null  | "Group1"    | ["Group1", "Group2"]
        null                     | "foo" | "some-id"                                           | "bar" | null        | ["Group2", "Group3"]
        "groups-test-provider-1" | "foo" | GROUPS_WITH_IDS["QAGroupTest-Group2"].props.getId() | "bar" | "Group2"    | ["Group2"]
        "groups-test-provider-2" | null  | "some-id"                                           | null  | null        | ["Group3"]
        "groups-test-provider-2" | "foo" | GROUPS_WITH_IDS["QAGroupTest-Group3"].props.getId() | "bar" | "Group3"    | ["Group3"]
    }
}
