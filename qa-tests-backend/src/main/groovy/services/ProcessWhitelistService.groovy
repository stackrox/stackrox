package services

import io.stackrox.proto.api.v1.ProcessWhitelistServiceGrpc
import io.stackrox.proto.api.v1.ProcessWhitelistServiceOuterClass
import io.stackrox.proto.storage.ProcessWhitelistOuterClass
import io.stackrox.proto.storage.ProcessWhitelistOuterClass.ProcessWhitelistKey
import objects.Deployment
import util.Timer

class ProcessWhitelistService extends BaseService {
    static getProcessWhitelistService() {
        return ProcessWhitelistServiceGrpc.newBlockingStub(getChannel())
    }
    static  ProcessWhitelistOuterClass.ProcessWhitelist getProcessWhitelist(
            String clusterId, Deployment deployment, String containerName = null,
            int iterations = 20, int interval = 6) {
        String namespace = deployment.getNamespace()
        String deploymentId = deployment.getDeploymentUid()
        String cName = containerName ?: deployment.getName()
        ProcessWhitelistServiceOuterClass.GetProcessWhitelistRequest request = ProcessWhitelistServiceOuterClass.
                GetProcessWhitelistRequest.newBuilder().
                setKey(ProcessWhitelistOuterClass.ProcessWhitelistKey.newBuilder()
                        .setClusterId(clusterId)
                        .setNamespace(namespace)
                        .setDeploymentId(deploymentId)
                        .setContainerName(cName).build())
                .build()
        Timer t = new Timer(iterations, interval)
        while (t.IsValid()) {
            if (getWhitelistProcesses(request)) {
                println "SR found whitelisted process for the key - " +
                        "${clusterId}, ${namespace}, ${deploymentId}, ${containerName} " +
                            " within ${t.SecondsSince()}s"
                return getProcessWhitelistService().getProcessWhitelist(request)
                }
            println "SR has not found whitelisted  process for the key - " +
                    "${clusterId}, ${namespace}, ${deploymentId}, ${containerName} yet"
        }
        println "SR has not found whitelisted  process for the key in - " +
                "${clusterId}, ${namespace}, ${deploymentId}, ${containerName} " +
                "${iterations * interval} seconds"
        return null
    }

    static List<ProcessWhitelistOuterClass.ProcessWhitelist> lockProcessWhitelists(
            String clusterId, Deployment deployment, String containerName, boolean  lock) {
        try {
            String cName = containerName ?: deployment.getName()
            ProcessWhitelistServiceOuterClass.LockProcessWhitelistsRequest lockRequest =
                     ProcessWhitelistServiceOuterClass
                    .LockProcessWhitelistsRequest.newBuilder()
                    .addKeys(ProcessWhitelistOuterClass.ProcessWhitelistKey
                    .newBuilder().setClusterId(clusterId)
                            .setNamespace(deployment.getNamespace())
                            .setDeploymentId(deployment.getDeploymentUid())
                            .setContainerName(cName).build())
                             .setLocked(lock).build()
            return getProcessWhitelistService().lockProcessWhitelists(lockRequest).whitelistsList
        } catch (Exception e) {
            println "Error locking process whitelists : ${e}"
        }
    }

    static List<ProcessWhitelistOuterClass.ProcessWhitelist> updateProcessWhitelists(
            ProcessWhitelistKey[] keys,
            String [] toBeAddedProcesses,
            String[] toBeRemovedProcesses) {
        try {
            ProcessWhitelistServiceOuterClass.UpdateProcessWhitelistsRequest.Builder requestBuilder =
                ProcessWhitelistServiceOuterClass.UpdateProcessWhitelistsRequest.newBuilder()
            for ( ProcessWhitelistKey key : keys) {
                requestBuilder.addKeys(key)
            }
            ProcessWhitelistOuterClass.WhitelistItemOrBuilder itemBuilder =
                    ProcessWhitelistOuterClass.WhitelistItem.newBuilder()
            for ( String processToBeAdded : toBeAddedProcesses) {
                ProcessWhitelistOuterClass.WhitelistItem   item  =
                        itemBuilder.setProcessName(processToBeAdded).build()
                requestBuilder.addAddElements(item)
            }

            for ( String processToBeRemoved : toBeRemovedProcesses) {
                ProcessWhitelistOuterClass.WhitelistItem   item  =
                        itemBuilder.setProcessName(processToBeRemoved).build()
                requestBuilder.addRemoveElements(item)
            }
            List<ProcessWhitelistOuterClass.ProcessWhitelist> updatedLst = getProcessWhitelistService()
                .updateProcessWhitelists(requestBuilder.build()).whitelistsList
            return updatedLst
    } catch (Exception e) {
            println "Error updating process whitelists: ${e}"
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

    static boolean waitForDeploymentWhitelistsCreated(String clusterId, Deployment deployment, String containerName) {
        Timer t = new Timer(20, 6)
        try {
            while (t.IsValid()) {
                ProcessWhitelistOuterClass.ProcessWhitelist whitelist =
                        getProcessWhitelist(clusterId, deployment, containerName)
                if (whitelist != null) {
                    return true
                }
            }
            println("Did not find whitelists for deployment ${deployment.getDeploymentUid()}")
        }
        catch (Exception e) {
            println "Error waiting for deployment whitelists to be created"
        }
        return false
    }

    static boolean waitForDeploymentWhitelistsDeleted(String clusterId, Deployment deployment, String containerName) {
        Timer t = new Timer(5, 2)
        try {
            while (t.IsValid()) {
                ProcessWhitelistOuterClass.ProcessWhitelist whitelist =
                        getProcessWhitelist(clusterId, deployment, containerName)
                if (whitelist == null) {
                    return true
                }
            }
            println("Whitelists still exist for deployment ${deployment.getDeploymentUid()}")
        }
        catch (Exception e) {
            println "Error waiting for deployment whitelists to be deleted"
        }
        return false
    }
}
