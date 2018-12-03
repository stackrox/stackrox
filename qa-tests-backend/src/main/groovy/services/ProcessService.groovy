package services

import stackrox.generated.Indicator.ProcessIndicator
import stackrox.generated.ProcessServiceGrpc
import stackrox.generated.ProcessServiceOuterClass

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
