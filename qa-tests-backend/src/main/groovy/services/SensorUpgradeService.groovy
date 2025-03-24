package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.SensorUpgradeServiceGrpc

@CompileStatic
class SensorUpgradeService extends BaseService {
    static SensorUpgradeServiceGrpc.SensorUpgradeServiceBlockingStub getSensorUpgradeServiceClient() {
        return SensorUpgradeServiceGrpc.newBlockingStub(getChannel())
    }

    static triggerCertRotation(String clusterId) {
        return getSensorUpgradeServiceClient().triggerSensorCertRotation(
            Common.ResourceByID.newBuilder().setId(clusterId).build()
        )
    }

}
