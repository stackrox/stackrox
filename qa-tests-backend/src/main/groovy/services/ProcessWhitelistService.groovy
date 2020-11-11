package services

import io.stackrox.proto.api.v1.ProcessWhitelistServiceGrpc
import io.stackrox.proto.storage.ProcessWhitelistOuterClass
import io.stackrox.proto.storage.ProcessWhitelistOuterClass.ProcessWhitelistKey
import io.stackrox.proto.api.v1.ProcessWhitelistServiceOuterClass
import io.stackrox.proto.api.v1.ProcessWhitelistServiceOuterClass.DeleteProcessWhitelistsRequest
import objects.Deployment
import util.Timer

class ProcessWhitelistService extends BaseService {
    static getProcessWhitelistService() {
        return ProcessWhitelistServiceGrpc.newBlockingStub(getChannel())
    }

    static  ProcessWhitelistOuterClass.ProcessWhitelist getProcessWhitelist(
            String clusterId, Deployment deployment, String containerName = null,
            int retries = 20, int interval = 6) {
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
        Timer t = new Timer(retries, interval)
        while (t.IsValid()) {
            def whitelist = getWhitelistProcesses(request)
            if (whitelist) {
                println "SR found baselined process for the key - " +
                        "${clusterId}, ${namespace}, ${deploymentId}, ${containerName} " +
                            " within ${t.SecondsSince()}s"
                return whitelist
                }
            println "SR has not found baselined  process for the key - " +
                    "${clusterId}, ${namespace}, ${deploymentId}, ${containerName} yet"
        }
        println "SR has not found baselined  process for the key in - " +
                "${clusterId}, ${namespace}, ${deploymentId}, ${containerName} " +
                "${t.SecondsSince()} seconds"
        return null
    }

    static List<ProcessWhitelistOuterClass.ProcessWhitelist> lockProcessWhitelists(
            String clusterId, Deployment deployment, String containerName, boolean lock) {
        try {
            String cName = containerName ?: deployment.getName()
            ProcessWhitelistKey keyToLock = ProcessWhitelistKey
                    .newBuilder()
                        .setClusterId(clusterId)
                        .setNamespace(deployment.getNamespace())
                        .setDeploymentId(deployment.getDeploymentUid())
                        .setContainerName(cName)
                    .build()

            ProcessWhitelistServiceOuterClass.LockProcessWhitelistsRequest lockRequest =
                     ProcessWhitelistServiceOuterClass.LockProcessWhitelistsRequest
                             .newBuilder()
                               .addKeys(keyToLock)
                               .setLocked(lock)
                             .build()

            def fromUpdate = getProcessWhitelistService().lockProcessWhitelists(lockRequest).whitelistsList

            ProcessWhitelistServiceOuterClass.GetProcessWhitelistRequest getRequest =
                    ProcessWhitelistServiceOuterClass.GetProcessWhitelistRequest
                            .newBuilder()
                                .setKey(keyToLock)
                            .build()

            ProcessWhitelistOuterClass.ProcessWhitelist wl =
                    getProcessWhitelistService().getProcessWhitelist(getRequest)

            if (wl.hasUserLockedTimestamp()) {
                if (!lock) {
                    throw new RuntimeException("Asked to unlock but the lock is still set")
                }
            }
            else {
                if (lock) {
                    throw new RuntimeException("Asked to lock but the lock is not set")
                }
            }

            return fromUpdate
        } catch (Exception e) {
            println "Error locking process baselines : ${e}"
        }
    }

    static deleteProcessWhitelists(String query) {
        DeleteProcessWhitelistsRequest req = DeleteProcessWhitelistsRequest.newBuilder()
            .setQuery(query).setConfirm(true).build()
        return getProcessWhitelistService().deleteProcessWhitelists(req)
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
            println "Error updating process baselines: ${e}"
        }
    }

    static ProcessWhitelistOuterClass.ProcessWhitelist getWhitelistProcesses(
        ProcessWhitelistServiceOuterClass.GetProcessWhitelistRequest request) {
        try {
            return getProcessWhitelistService().getProcessWhitelist(request)
        }
        catch (Exception e) {
            println "Error getting  process baselines: ${e}"
        }
        return null
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
            println("Did not find baselines for deployment ${deployment.getDeploymentUid()}")
        }
        catch (Exception e) {
            println "Error waiting for deployment baselines to be created ${e}"
        }
        return false
    }

    static boolean waitForDeploymentWhitelistsDeleted(String clusterId, Deployment deployment, String containerName) {
        Timer t = new Timer(5, 2)
        try {
            while (t.IsValid()) {
                ProcessWhitelistOuterClass.ProcessWhitelist whitelist =
                        getProcessWhitelist(clusterId, deployment, containerName, 1)
                if (whitelist == null) {
                    return true
                }
            }
            println("Whitelists still exist for deployment ${deployment.getDeploymentUid()}")
        }
        catch (Exception e) {
            println "Error waiting for deployment baselines to be deleted ${e}"
        }
        return false
    }
}
