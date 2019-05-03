package services

import io.stackrox.proto.api.v1.ProcessWhitelistServiceGrpc
import io.stackrox.proto.api.v1.ProcessWhitelistServiceOuterClass
import io.stackrox.proto.storage.ProcessWhitelistOuterClass

class ProcessWhitelistService extends BaseService {
    static getProcessWhitelistService() {
        return ProcessWhitelistServiceGrpc.newBlockingStub(getChannel())
    }
    static List<ProcessWhitelistOuterClass.ProcessWhitelist> getProcessWhiteLists() {
        return getProcessWhitelistService().getProcessWhitelists().getWhitelistsList()
    }
    static  ProcessWhitelistOuterClass.ProcessWhitelist getProcessWhitelist(String deploymentId, String containerName) {
        ProcessWhitelistServiceOuterClass.GetProcessWhitelistRequest request = ProcessWhitelistServiceOuterClass.
                GetProcessWhitelistRequest.newBuilder().
                setKey(ProcessWhitelistOuterClass.ProcessWhitelistKey.newBuilder()
                        .setDeploymentId(deploymentId).setContainerName(containerName).build())
                .build()
        return getProcessWhitelistService().getProcessWhitelist(request)
    }
    static List<ProcessWhitelistOuterClass.ProcessWhitelist> updateProcessWhitelists(
            ProcessWhitelistServiceOuterClass.UpdateProcessWhitelistsRequest request) {
        try {
            return getProcessWhitelistService().updateProcessWhitelists(request).whitelistsList
        } catch (Exception e) {
            println "Error updating process whitelists: ${e}"
        }
    }
    static List<ProcessWhitelistOuterClass.ProcessWhitelist> lockProcessWhitelists(
            String deploymentId, String containerName) {
        try {
            ProcessWhitelistServiceOuterClass.LockProcessWhitelistsRequest lockRequest =
                     ProcessWhitelistServiceOuterClass
                    .LockProcessWhitelistsRequest.newBuilder()
                    .addKeys(ProcessWhitelistOuterClass.ProcessWhitelistKey
                    .newBuilder().setDeploymentId(deploymentId)
                    .setContainerName(containerName).build())
                    .setLocked(true).build()
            return getProcessWhitelistService().lockProcessWhitelists(lockRequest).whitelistsList
        } catch (Exception e) {
            println "Error locking process whitelists : ${e}"
        }
    }
}
