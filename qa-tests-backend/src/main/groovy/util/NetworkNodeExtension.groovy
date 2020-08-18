package util

import io.stackrox.proto.api.v1.NetworkGraphOuterClass.NetworkNode
import io.stackrox.proto.storage.NetworkFlowOuterClass.L4Protocol
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntityInfo

class NetworkNodeExtension {
    static String getDeploymentId(NetworkNode self) {
        if (self.entity == null || self.entity.type == NetworkEntityInfo.Type.UNKNOWN_TYPE) {
            return null
        }
        return self.entity.id
    }

    static List<NetworkEntityInfo.Deployment.ListenPort> listenPorts(NetworkNode self, L4Protocol filterProto = null) {
        def ports = (self?.entity?.deployment?.listenPortsList ?: [])
        if (!filterProto) {
            return ports
        }
        return ports.findAll {
            it.l4Protocol == filterProto
        }
    }

    static String getDeploymentName(NetworkNode self) {
        if (self.entity == null || self.entity.type == NetworkEntityInfo.Type.UNKNOWN_TYPE) {
            return null
        }
        return self.entity.deployment.name
    }

    static String getNamespace(NetworkNode self) {
        if (self.entity == null || self.entity.type == NetworkEntityInfo.Type.UNKNOWN_TYPE) {
            return null
        }
        return self.entity.deployment.namespace
    }
}
