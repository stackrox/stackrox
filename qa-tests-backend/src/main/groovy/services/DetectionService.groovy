package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.DetectionServiceGrpc
import io.stackrox.proto.api.v1.DetectionServiceOuterClass.DeployDetectionResponse
import io.stackrox.proto.api.v1.DetectionServiceOuterClass.DeployYAMLDetectionRequest

@CompileStatic
class DetectionService extends BaseService {
    static DetectionServiceGrpc.DetectionServiceBlockingStub getDetectionClient() {
        return DetectionServiceGrpc.newBlockingStub(getChannel())
    }

    static DeployDetectionResponse getDetectDeploytimeFromYAML(
            DeployYAMLDetectionRequest request = DeployYAMLDetectionRequest.newBuilder().build()) {
        return getDetectionClient().detectDeployTimeFromYAML(request)
    }

}
