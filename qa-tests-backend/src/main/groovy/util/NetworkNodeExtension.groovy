package util

import stackrox.generated.NetworkGraphOuterClass.NetworkNode
import stackrox.generated.NetworkFlowOuterClass.NetworkEntityInfo

class NetworkNodeExtension {
    static String getDeploymentId(NetworkNode self) {
        if (self.entity == null || self.entity.type == NetworkEntityInfo.Type.UNKNOWN_TYPE) {
            return null
        }
        return self.entity.id
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
