package services

import io.stackrox.proto.api.v1.GroupServiceGrpc
import io.stackrox.proto.api.v1.GroupServiceOuterClass.Group
import io.stackrox.proto.api.v1.GroupServiceOuterClass.GroupProperties

class GroupService extends BaseService {
    static getGroupService() {
        return GroupServiceGrpc.newBlockingStub(getChannel())
    }

    static createGroup(Group group) {
        try {
            return getGroupService().createGroup(group)
        } catch (Exception e) {
            println "Error creating new Group: ${e}"
        }
    }

    static deleteGroup(GroupProperties props) {
        try {
            return getGroupService().deleteGroup(props)
        } catch (Exception e) {
            println "Error deleting group: ${e}"
        }
    }

    static getGroup(GroupProperties props) {
        return getGroupService().getGroup(props)
    }
}
