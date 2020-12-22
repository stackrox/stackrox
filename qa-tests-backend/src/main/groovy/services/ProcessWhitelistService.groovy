package services

import io.stackrox.proto.api.v1.ProcessBaselineServiceGrpc
import io.stackrox.proto.storage.ProcessBaselineOuterClass
import io.stackrox.proto.storage.ProcessBaselineOuterClass.ProcessBaselineKey
import io.stackrox.proto.api.v1.ProcessBaselineServiceOuterClass
import io.stackrox.proto.api.v1.ProcessBaselineServiceOuterClass.DeleteProcessBaselinesRequest
import objects.Deployment
import util.Timer

class ProcessWhitelistService extends BaseService {
    static getProcessWhitelistService() {
        return ProcessBaselineServiceGrpc.newBlockingStub(getChannel())
    }

    static  ProcessBaselineOuterClass.ProcessBaseline getProcessWhitelist(
            String clusterId, Deployment deployment, String containerName = null,
            int retries = 20, int interval = 6) {
        String namespace = deployment.getNamespace()
        String deploymentId = deployment.getDeploymentUid()
        String cName = containerName ?: deployment.getName()
        ProcessBaselineServiceOuterClass.GetProcessBaselineRequest request = ProcessBaselineServiceOuterClass.
                GetProcessBaselineRequest.newBuilder().
                setKey(ProcessBaselineKey.newBuilder()
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

    static List<ProcessBaselineOuterClass.ProcessBaseline> lockProcessWhitelists(
            String clusterId, Deployment deployment, String containerName, boolean lock) {
        try {
            String cName = containerName ?: deployment.getName()
            ProcessBaselineKey keyToLock = ProcessBaselineKey
                    .newBuilder()
                        .setClusterId(clusterId)
                        .setNamespace(deployment.getNamespace())
                        .setDeploymentId(deployment.getDeploymentUid())
                        .setContainerName(cName)
                    .build()

            ProcessBaselineServiceOuterClass.LockProcessBaselinesRequest lockRequest =
                     ProcessBaselineServiceOuterClass.LockProcessBaselinesRequest
                             .newBuilder()
                               .addKeys(keyToLock)
                               .setLocked(lock)
                             .build()

            def fromUpdate = getProcessWhitelistService().lockProcessBaselines(lockRequest).baselinesList

            ProcessBaselineServiceOuterClass.GetProcessBaselineRequest getRequest =
                    ProcessBaselineServiceOuterClass.GetProcessBaselineRequest
                            .newBuilder()
                                .setKey(keyToLock)
                            .build()

            ProcessBaselineOuterClass.ProcessBaseline wl =
                    getProcessWhitelistService().getProcessBaseline(getRequest)

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
        DeleteProcessBaselinesRequest req = DeleteProcessBaselinesRequest.newBuilder()
            .setQuery(query).setConfirm(true).build()
        return getProcessWhitelistService().deleteProcessBaselines(req)
    }

    static List<ProcessBaselineOuterClass.ProcessBaseline> updateProcessWhitelists(
            ProcessBaselineKey[] keys,
            String [] toBeAddedProcesses,
            String[] toBeRemovedProcesses) {
        try {
            ProcessBaselineServiceOuterClass.UpdateProcessBaselinesRequest.Builder requestBuilder =
                ProcessBaselineServiceOuterClass.UpdateProcessBaselinesRequest.newBuilder()
            for ( ProcessBaselineKey key : keys) {
                requestBuilder.addKeys(key)
            }
            ProcessBaselineOuterClass.BaselineItemOrBuilder itemBuilder =
                    ProcessBaselineOuterClass.BaselineItem.newBuilder()
            for ( String processToBeAdded : toBeAddedProcesses) {
                ProcessBaselineOuterClass.BaselineItem   item  =
                        itemBuilder.setProcessName(processToBeAdded).build()
                requestBuilder.addAddElements(item)
            }

            for ( String processToBeRemoved : toBeRemovedProcesses) {
                ProcessBaselineOuterClass.BaselineItem   item  =
                        itemBuilder.setProcessName(processToBeRemoved).build()
                requestBuilder.addRemoveElements(item)
            }
            List<ProcessBaselineOuterClass.ProcessBaseline> updatedLst = getProcessWhitelistService()
                .updateProcessBaselines(requestBuilder.build()).baselinesList
            return updatedLst
        } catch (Exception e) {
            println "Error updating process baselines: ${e}"
        }
    }

    static ProcessBaselineOuterClass.ProcessBaseline getWhitelistProcesses(
        ProcessBaselineServiceOuterClass.GetProcessBaselineRequest request) {
        try {
            return getProcessWhitelistService().getProcessBaseline(request)
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
                ProcessBaselineOuterClass.ProcessBaseline whitelist =
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
                ProcessBaselineOuterClass.ProcessBaseline whitelist =
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
