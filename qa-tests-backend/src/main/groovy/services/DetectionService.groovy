package services

import io.stackrox.proto.api.v1.DetectionServiceGrpc
import io.stackrox.proto.api.v1.DetectionServiceOuterClass.DeployDetectionResponse
import io.stackrox.proto.api.v1.DetectionServiceOuterClass.DeployYAMLDetectionRequest

class DetectionService extends BaseService {
    static getDetectionClient() {
        return DetectionServiceGrpc.newBlockingStub(getChannel())
    }

    static DeployDetectionResponse getDetectDeploytimeFromYAML(
            DeployYAMLDetectionRequest request = DeployYAMLDetectionRequest.newBuilder().build()) {
        return getDetectionClient().detectDeployTimeFromYAML(request)
    }

}
