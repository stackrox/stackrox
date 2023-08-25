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

    static Map<String, Set<Tuple2<Integer, Integer>>> getProcessUserAndGroupIds(String deploymentID) {
        def response = getClient().getProcessesByDeployment(ProcessServiceOuterClass.GetProcessesByDeploymentRequest
                .newBuilder()
                .setDeploymentId(deploymentID)
                .build())
        Map<String,Set<Tuple2<Integer,Integer>>> pathToIds = [:]
        for ( ProcessIndicator process : response.getProcessesList() ) {
            String path = process.getSignal().getExecFilePath()
            Integer uid = process.getSignal().getUid()
            Integer gid = process.getSignal().getGid()
            pathToIds.putIfAbsent(path, [] as Set<Tuple2<Integer,Integer>>)
            pathToIds[path].add([uid, gid] as Tuple2<Integer, Integer>)
        }
        return pathToIds
    }

    static List<Tuple2<String, String>> getProcessesWithArgs(String deploymentID) {
        def response = getClient().getProcessesByDeployment(ProcessServiceOuterClass.GetProcessesByDeploymentRequest
                .newBuilder()
                .setDeploymentId(deploymentID)
                .build())

        List<Tuple2<String, String>> processes = []
        for (ProcessIndicator process : response.getProcessesList()) {
            String path = process.getSignal().getExecFilePath()
            String args = process.getSignal().getArgs()
            processes.add([path, args] as Tuple2<String, String>)
        }
        return processes
    }

    // Returns a map of process path -> list of container id's for each container
    // the path was executed (this list may have duplicates)
    static Map<String, List<String> > getProcessContainerMap(String deploymentID, Set<String> processes = null) {
        def response = getClient().getProcessesByDeployment(ProcessServiceOuterClass.GetProcessesByDeploymentRequest
            .newBuilder()
            .setDeploymentId(deploymentID)
            .build())

        Map<String, List<String> > pathContainerMap = new HashMap<>()
        for ( ProcessIndicator process : response.getProcessesList() ) {
            String path = process.getSignal().getExecFilePath()
            if (processes != null && !processes.contains(path)) {
                continue
            }
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

    static List<ProcessIndicator> getProcessIndicatorsByDeployment(String deploymentID) {
        def response = getClient().getProcessesByDeployment(ProcessServiceOuterClass.GetProcessesByDeploymentRequest
                .newBuilder()
                .setDeploymentId(deploymentID)
                .build())

        return response.getProcessesList()
    }

    static getGroupedProcessByDeploymentAndContainer(String deploymentId) {
        def response = getClient().getGroupedProcessByDeploymentAndContainer(
            ProcessServiceOuterClass.GetProcessesByDeploymentRequest.newBuilder()
                .setDeploymentId(deploymentId)
                .build()
        )

        return response.groupsList
    }
}
