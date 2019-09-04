package services

import io.stackrox.proto.api.v1.ProcessWhitelistServiceGrpc
import io.stackrox.proto.api.v1.ProcessWhitelistServiceOuterClass
import io.stackrox.proto.storage.ProcessWhitelistOuterClass.ProcessWhitelist
import io.stackrox.proto.storage.ProcessWhitelistOuterClass.WhitelistItemOrBuilder
import io.stackrox.proto.storage.ProcessWhitelistOuterClass.WhitelistItem
import io.stackrox.proto.storage.ProcessWhitelistOuterClass.ProcessWhitelistKey
import objects.Deployment
import util.Timer

class ProcessWhitelistService extends BaseService {
    static getProcessWhitelistService() {
        return ProcessWhitelistServiceGrpc.newBlockingStub(getChannel())
    }
    static ProcessWhitelist getProcessWhitelist(
            String clusterId, Deployment deployment, String containerName = null) {
        String namespace = deployment.getNamespace()
        String deploymentId = deployment.getDeploymentUid()
        String cName = containerName ?: deployment.getName()
        ProcessWhitelistServiceOuterClass.GetProcessWhitelistRequest request = ProcessWhitelistServiceOuterClass.
                GetProcessWhitelistRequest.newBuilder().
                setKey(ProcessWhitelistKey.newBuilder()
                        .setClusterId(clusterId)
                        .setNamespace(namespace)
                        .setDeploymentId(deploymentId)
                        .setContainerName(cName).build())
                .build()
        return getWhitelistProcesses(request)
    }

    static List<ProcessWhitelist> lockProcessWhitelists(
            String clusterId, Deployment deployment, String containerName, boolean  lock) {
        try {
            String cName = containerName ?: deployment.getName()
            ProcessWhitelistServiceOuterClass.LockProcessWhitelistsRequest lockRequest =
                     ProcessWhitelistServiceOuterClass
                    .LockProcessWhitelistsRequest.newBuilder()
                    .addKeys(ProcessWhitelistKey
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

    static List<ProcessWhitelist> updateProcessWhitelists(
            ProcessWhitelistKey[] keys,
            String [] toBeAddedProcesses,
            String[] toBeRemovedProcesses) {
        try {
            ProcessWhitelistServiceOuterClass.UpdateProcessWhitelistsRequest.Builder requestBuilder =
                ProcessWhitelistServiceOuterClass.UpdateProcessWhitelistsRequest.newBuilder()
            for ( ProcessWhitelistKey key : keys) {
                requestBuilder.addKeys(key)
            }
            WhitelistItemOrBuilder itemBuilder =
                    WhitelistItem.newBuilder()
            for ( String processToBeAdded : toBeAddedProcesses) {
                WhitelistItem item  =
                        itemBuilder.setProcessName(processToBeAdded).build()
                requestBuilder.addAddElements(item)
            }

            for ( String processToBeRemoved : toBeRemovedProcesses) {
                WhitelistItem   item  =
                        itemBuilder.setProcessName(processToBeRemoved).build()
                requestBuilder.addRemoveElements(item)
            }
            List<ProcessWhitelist> updatedLst = getProcessWhitelistService()
                .updateProcessWhitelists(requestBuilder.build()).whitelistsList
            return updatedLst
    } catch (Exception e) {
            println "Error updating process whitelists: ${e}"
    }
    }

    static ProcessWhitelist getWhitelistProcesses(
        ProcessWhitelistServiceOuterClass.GetProcessWhitelistRequest request) {
        try {
            return getProcessWhitelistService().getProcessWhitelist(request)
        }
        catch (Exception e) {
            println "Error getting  process whitelists: ${e}"
        }
        return null
    }

    static ProcessWhitelist waitForDeploymentWhitelistsCreated(String clusterId, Deployment deployment,
                                                               String containerName) {
        Timer t = new Timer(20, 6)
        try {
            while (t.IsValid()) {
                ProcessWhitelist whitelist =
                        getProcessWhitelist(clusterId, deployment, containerName)
                if (whitelist != null) {
                    return whitelist
                }
                println("Waiting for whitelist to be created")
            }
            println("Did not find whitelists for deployment ${deployment.getDeploymentUid()}")
        }
        catch (Exception e) {
            println "Error waiting for deployment whitelists to be created ${e}"
        }
        return null
    }

    static boolean waitForDeploymentWhitelistsDeleted(String clusterId, Deployment deployment, String containerName) {
        Timer t = new Timer(5, 2)
        try {
            while (t.IsValid()) {
                ProcessWhitelist whitelist =
                        getProcessWhitelist(clusterId, deployment, containerName)
                if (whitelist == null) {
                    return true
                }
                println("Waiting for whitelist to be deleted")
            }
            println("Whitelists still exist for deployment ${deployment.getDeploymentUid()}")
        }
        catch (Exception e) {
            println "Error waiting for deployment whitelists to be deleted ${e}"
        }
        return false
    }
}
