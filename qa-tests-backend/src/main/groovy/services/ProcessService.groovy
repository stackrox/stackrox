package services

import io.stackrox.proto.api.v1.Indicator.ProcessIndicator
import io.stackrox.proto.api.v1.ProcessServiceGrpc
import io.stackrox.proto.api.v1.ProcessServiceOuterClass

class ProcessService extends BaseService {
    static getClient() {
        return ProcessServiceGrpc.newBlockingStub(getChannel())
    }

    static List<String> getProcessPaths(String deploymentID) {
        def response = getClient().getProcessesByDeployment(ProcessServiceOuterClass.GetProcessesByDeploymentRequest
            .newBuilder()
            .setDeploymentId(deploymentID)
            .build())

        List<String> paths = []
        for ( ProcessIndicator process : response.getProcessesList() ) {
            paths.add(process.getSignal().getExecFilePath())
        }
        return paths
    }
}
