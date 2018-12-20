package services

import io.stackrox.proto.api.v1.ProcessServiceGrpc
import io.stackrox.proto.api.v1.ProcessServiceOuterClass
import io.stackrox.proto.storage.ProcessIndicatorOuterClass.ProcessIndicator

class ProcessService extends BaseService {
    static getClient() {
        return ProcessServiceGrpc.newBlockingStub(getChannel())
    }

    static Set<String> getUniqueProcessPaths(String deploymentID) {
        def response = getClient().getProcessesByDeployment(ProcessServiceOuterClass.GetProcessesByDeploymentRequest
            .newBuilder()
            .setDeploymentId(deploymentID)
            .build())

        Set<String> paths = []
        for ( ProcessIndicator process : response.getProcessesList() ) {
            paths.add(process.getSignal().getExecFilePath())
        }
        return paths
    }

    // Returns a map of process path -> list of container id's for each container
    // the path was executed (this list may have duplicates)
    static Map<String, List<String> > getProcessContainerMap(String deploymentID) {
        def response = getClient().getProcessesByDeployment(ProcessServiceOuterClass.GetProcessesByDeploymentRequest
            .newBuilder()
            .setDeploymentId(deploymentID)
            .build())

        Map<String, List<String> > pathContainerMap = new HashMap<>()
        for ( ProcessIndicator process : response.getProcessesList() ) {
            String path = process.getSignal().getExecFilePath()
            String containerId = process.getSignal().getContainerId()
            List<String> containerList = pathContainerMap.get(path)
            if (containerList == null) {
                containerList = new ArrayList<>()
                pathContainerMap.put(path, containerList)
            }
            containerList.add(containerId)
        }
        return pathContainerMap
    }
}
