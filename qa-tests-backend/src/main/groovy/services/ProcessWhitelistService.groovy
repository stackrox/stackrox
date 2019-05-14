package services

import io.stackrox.proto.api.v1.ProcessWhitelistServiceGrpc
import io.stackrox.proto.api.v1.ProcessWhitelistServiceOuterClass
import io.stackrox.proto.storage.ProcessWhitelistOuterClass
import util.Timer

class ProcessWhitelistService extends BaseService {
    static getProcessWhitelistService() {
        return ProcessWhitelistServiceGrpc.newBlockingStub(getChannel())
    }
    static  ProcessWhitelistOuterClass.ProcessWhitelist getProcessWhitelist(
            String deploymentId, String containerName, int iterations = 20, int interval = 6) {
        ProcessWhitelistServiceOuterClass.GetProcessWhitelistRequest request = ProcessWhitelistServiceOuterClass.
                GetProcessWhitelistRequest.newBuilder().
                setKey(ProcessWhitelistOuterClass.ProcessWhitelistKey.newBuilder()
                        .setDeploymentId(deploymentId).setContainerName(containerName).build())
                .build()
        Timer t = new Timer(iterations, interval)
        while (t.IsValid()) {
            if (getWhitelistProcesses(request)) {
                println "SR found whitelisted process for the key - ${deploymentId} and ${containerName} " +
                            " within ${t.SecondsSince()}s"
                return getProcessWhitelistService().getProcessWhitelist(request)
                }
            println "SR has not found whitelisted  process for the key - ${deploymentId} and ${containerName} yet"
        }
        println "SR has not found whitelisted  process for the key in - ${deploymentId} and ${containerName} " +
                "${iterations * interval} seconds"
        return null
    }

    static List<ProcessWhitelistOuterClass.ProcessWhitelist> lockProcessWhitelists(
            String deploymentId, String containerName, boolean  lock) {
        try {
            ProcessWhitelistServiceOuterClass.LockProcessWhitelistsRequest lockRequest =
                     ProcessWhitelistServiceOuterClass
                    .LockProcessWhitelistsRequest.newBuilder()
                    .addKeys(ProcessWhitelistOuterClass.ProcessWhitelistKey
                    .newBuilder().setDeploymentId(deploymentId)
                    .setContainerName(containerName).build())
                    .setLocked(lock).build()
            return getProcessWhitelistService().lockProcessWhitelists(lockRequest).whitelistsList
        } catch (Exception e) {
            println "Error locking process whitelists : ${e}"
        }
    }

    static boolean getWhitelistProcesses(ProcessWhitelistServiceOuterClass.GetProcessWhitelistRequest request) {
        try {
            getProcessWhitelistService().getProcessWhitelist(request)
        }
        catch (Exception e) {
            println "Error getting  process whitelists: ${e}"
            return false
        }
        return true
    }

    static List<ProcessWhitelistOuterClass.ProcessWhitelist> getProcessWhitelists() {
        try {
            return getProcessWhitelistService().getProcessWhitelists().whitelistsList
        }
        catch (Exception e) {
            println "Error getting all process whitelists"
        }
    }

    static boolean waitForDeploymentWhitelistsCreated(String deploymentId) {
        Timer t = new Timer(20, 6)
        try {
            while (t.IsValid()) {
                def whitelists = getProcessWhitelistService().getProcessWhitelists().whitelistsList
                for (ProcessWhitelistOuterClass.ProcessWhitelist whitelist : whitelists) {
                    if (whitelist.getKey().getDeploymentId() == deploymentId) {
                        return true
                    }
                }
                println("Did not find whitelists for deployment ${deploymentId}")
            }
        }
        catch (Exception e) {
            println "Error waiting for deployment whitelists to be created"
        }
        return false
    }

    static boolean waitForDeploymentWhitelistsDeleted(String deploymentId) {
        Timer t = new Timer(5, 2)
        try {
            while (t.IsValid()) {
                def whitelists = getProcessWhitelistService().getProcessWhitelists().whitelistsList
                for (ProcessWhitelistOuterClass.ProcessWhitelist whitelist : whitelists) {
                    if (whitelist.getKey().getDeploymentId() == deploymentId) {
                        println("Whitelists still exist for deployment ${deploymentId}")
                        continue
                    }
                }
                return true
            }
        }
        catch (Exception e) {
            println "Error waiting for deployment whitelists to be deleted"
        }
        return false
    }
}
