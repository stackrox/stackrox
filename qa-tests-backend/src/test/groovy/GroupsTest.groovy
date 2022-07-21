import groups.BAT
import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.GroupServiceOuterClass.GetGroupsRequest
import io.stackrox.proto.storage.GroupOuterClass.Group
import io.stackrox.proto.storage.GroupOuterClass.GroupProperties
import org.junit.experimental.categories.Category
import services.GroupService
import spock.lang.Unroll

@Category(BAT)
class GroupsTest extends BaseSpecification {

    private static final PROVIDERS = [
            UUID.randomUUID().toString(),
            UUID.randomUUID().toString(),
    ]

    private static final GROUPS = [
            Group.newBuilder()
                    .setRoleName("QAGroupTest-Group1")
                    .setProps(GroupProperties.newBuilder()
                        .setAuthProviderId(PROVIDERS[0])
                        .build())
                    .build(),
            Group.newBuilder()
                    .setRoleName("QAGroupTest-Group2")
                    .setProps(GroupProperties.newBuilder()
                        .setAuthProviderId(PROVIDERS[0])
                        .setKey("foo")
                        .setValue("bar")
                        .build())
                    .build(),
            Group.newBuilder()
                    .setRoleName("QAGroupTest-Group3")
                    .setProps(GroupProperties.newBuilder()
                        .setAuthProviderId(PROVIDERS[1])
                        .setKey("foo")
                        .setValue("bar")
                        .build())
                    .build(),
    ]

<<<<<<< HEAD
    private static final GROUPIDS = ["": ""]
=======
    private static final GROUP_IDS = ["": ""]
>>>>>>> 84b314e271 (Add groovy test)

    def setupSpec() {
        for (def group : GROUPS) {
            GroupService.createGroup(group)
            def props = group.getProps()
            def groupWithId = GroupService.getGroups(GetGroupsRequest.newBuilder()
                    .setAuthProviderId(props.getAuthProviderId())
                    .setValue(props.getValue())
                    .setKey(props.getKey()).build()
            ).getGroups(0)
            GROUPIDS[groupWithId.roleName] = groupWithId.getProps().getId()
        }
    }

    def cleanupSpec() {
        for (def group : GROUPS) {
            try {
                GroupService.deleteGroup(group.props)
            } catch (Exception ex) {
                log.warn("Failed to delete group", ex)
            }
        }
    }

    @Unroll
    def "Test that GetGroup and GetGroups work correctly with query args (#authProviderId, #key, #value)"() {
        when:
        "A query is made for GetGroup and GetGroups with the given params"
        def propsBuilder = GroupProperties.newBuilder()
        def reqBuilder = GetGroupsRequest.newBuilder()
        if (authProviderId != null) {
            propsBuilder.setAuthProviderId(PROVIDERS[authProviderId])
            reqBuilder.setAuthProviderId(PROVIDERS[authProviderId])
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
            if (ex.status.code != Status.Code.NOT_FOUND) {
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
        authProviderId | key   | id                              | value | expectGroup | expectGroups
        0              | null  | GROUPIDS["QAGroupTest-Group1"] | null  | "Group1"    | ["Group1", "Group2"]
        null           | "foo" | "some-id"                       | "bar" | null        | ["Group2", "Group3"]
        0              | "foo" | GROUPIDS["QAGroupTest-Group2"] | "bar" | "Group2"    | ["Group2"]
        1              | null  | "some-id"                       | null  | null        | ["Group3"]
        1              | "foo" | GROUPIDS["QAGroupTest-Group3"] | "bar" | "Group3"    | ["Group3"]
    }
}
