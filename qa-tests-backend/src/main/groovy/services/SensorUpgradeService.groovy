package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.SensorUpgradeServiceGrpc

class SensorUpgradeService extends BaseService {
    static getSensorUpgradeServiceClient() {
        return SensorUpgradeServiceGrpc.newBlockingStub(getChannel())
    }

    static triggerCertRotation(String clusterId) {
        return getSensorUpgradeServiceClient().triggerSensorCertRotation(
            Common.ResourceByID.newBuilder().setId(clusterId).build()
        )
    }

}
