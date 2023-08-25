package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.GroupServiceGrpc
import io.stackrox.proto.api.v1.GroupServiceOuterClass
import io.stackrox.proto.api.v1.GroupServiceOuterClass.GetGroupsRequest
import io.stackrox.proto.storage.GroupOuterClass.Group
import io.stackrox.proto.storage.GroupOuterClass.GroupProperties

@Slf4j
class GroupService extends BaseService {
    static getGroupService() {
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

    static createGroup(Group group) {
        try {
            return getGroupService().createGroup(group)
        } catch (Exception e) {
            log.error("Error creating new Group", e)
        }
    }

    static deleteGroup(GroupProperties props) {
        try {
            return getGroupService().deleteGroup(GroupServiceOuterClass.DeleteGroupRequest.newBuilder()
                    .setAuthProviderId(props.authProviderId)
                    .setId(props.id)
                    .setKey(props.key)
                    .setValue(props.value)
                    .build()
            )
        } catch (Exception e) {
            log.error("Error deleting group", e)
        }
    }

    static getGroup(GroupProperties props) {
        return getGroupService().getGroup(props)
    }

    static getGroups(GetGroupsRequest req) {
        return getGroupService().getGroups(req)
    }
}
