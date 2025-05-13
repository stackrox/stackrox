package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

import io.stackrox.annotations.Retry
import io.stackrox.proto.api.v1.GroupServiceGrpc
import io.stackrox.proto.api.v1.GroupServiceOuterClass
import io.stackrox.proto.api.v1.GroupServiceOuterClass.GetGroupsRequest
import io.stackrox.proto.storage.GroupOuterClass.Group
import io.stackrox.proto.storage.GroupOuterClass.GroupProperties

@Slf4j
@CompileStatic
class GroupService extends BaseService {
    static GroupServiceGrpc.GroupServiceBlockingStub getGroupService() {
        return GroupServiceGrpc.newBlockingStub(getChannel())
    }

    static addDefaultMapping(String authProviderId, String defaultRoleName) {
        def group = Group.newBuilder()
            .setProps(GroupProperties.newBuilder()
                .setAuthProviderId(authProviderId))
            .setRoleName(defaultRoleName)
            .build()
        createGroup(group)
    }

    static removeAllMappingsForProvider(String authProviderId) {
        getGroups(
                GetGroupsRequest.newBuilder()
                        .setAuthProviderId(authProviderId)
                        .build()
        ).groupsList.forEach {
            deleteGroup(it.props)
        }
    }

    @Retry
    static createGroup(Group group) {
        return getGroupService().createGroup(group)
    }

    @Retry
    static deleteGroup(GroupProperties props) {
        return getGroupService().deleteGroup(GroupServiceOuterClass.DeleteGroupRequest.newBuilder()
                .setAuthProviderId(props.authProviderId)
                .setId(props.id)
                .setKey(props.key)
                .setValue(props.value)
                .build()
        )
    }

    @Retry
    static Group getGroup(GroupProperties props) {
        return getGroupService().getGroup(props)
    }

    @Retry
    static GroupServiceOuterClass.GetGroupsResponse getGroups(GetGroupsRequest req) {
        return getGroupService().getGroups(req)
    }
}
