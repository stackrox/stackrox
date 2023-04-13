import io.grpc.Status
import io.grpc.StatusRuntimeException

import io.stackrox.proto.api.v1.GroupServiceOuterClass.GetGroupsRequest
import io.stackrox.proto.storage.AuthProviderOuterClass
import io.stackrox.proto.storage.GroupOuterClass.Group
import io.stackrox.proto.storage.GroupOuterClass.GroupProperties

import services.AuthProviderService
import services.GroupService

import spock.lang.Tag

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
                    .setRoleName("Analyst")
                    .build()): PROVIDERS[0].getName(),
            (Group.newBuilder()
                    .setRoleName("Admin")
                    .setProps(GroupProperties.newBuilder()
                            .setKey("foo")
                            .setValue("bar")
                            .build())
                    .build()): PROVIDERS[0].getName(),
            (Group.newBuilder()
                    .setRoleName("Scope Manager")
                    .setProps(GroupProperties.newBuilder()
                            .setKey("foo")
                            .setValue("bar")
                            .build())
                    .build()): PROVIDERS[1].getName(),
    ]

    private static final Map<String, Group> GROUPS_WITH_IDS = [:]

    def setupSpec() {
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
    }

    def "Test that GetGroup and GetGroups work correctly with query args (#authProviderId, #key, #value)"() {
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
        authProviderName | key | id | value |
                expectGroup | expectGroups
        "groups-test-provider-1" | null  | GROUPS_WITH_IDS["QAGroupTest-Group1"].props.getId() | null  |
                "Group1"    | ["Group1", "Group2"]
        null                     | "foo" | "some-id"                                           | "bar" |
                null        | ["Group2", "Group3"]
        "groups-test-provider-1" | "foo" | GROUPS_WITH_IDS["QAGroupTest-Group2"].props.getId() | "bar" |
                "Group2"    | ["Group2"]
        "groups-test-provider-2" | null  | "some-id"                                           | null  |
                null        | ["Group3"]
        "groups-test-provider-2" | "foo" | GROUPS_WITH_IDS["QAGroupTest-Group3"].props.getId() | "bar" |
                "Group3"    | ["Group3"]
    }
}
