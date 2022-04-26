package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.ProcessBaselineServiceGrpc
import io.stackrox.proto.storage.ProcessBaselineOuterClass
import io.stackrox.proto.storage.ProcessBaselineOuterClass.ProcessBaselineKey
import io.stackrox.proto.api.v1.ProcessBaselineServiceOuterClass
import io.stackrox.proto.api.v1.ProcessBaselineServiceOuterClass.DeleteProcessBaselinesRequest
import objects.Deployment
import util.Timer

@Slf4j
class ProcessBaselineService extends BaseService {
    static getProcessBaselineService() {
        return ProcessBaselineServiceGrpc.newBlockingStub(getChannel())
    }

    static  ProcessBaselineOuterClass.ProcessBaseline getProcessBaseline(
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
            def baseline = getBaselineProcesses(request)
            if (baseline) {
                log.info "SR found process in baseline for the key - " +
                        "${clusterId}, ${namespace}, ${deploymentId}, ${containerName} " +
                            " within ${t.SecondsSince()}s"
                return baseline
                }
            log.debug "SR has not found process in baseline for the key - " +
                    "${clusterId}, ${namespace}, ${deploymentId}, ${containerName} yet"
        }
        log.warn "SR has not found process in baseline for the key in - " +
                "${clusterId}, ${namespace}, ${deploymentId}, ${containerName} " +
                "${t.SecondsSince()} seconds"
        return null
    }

    static List<ProcessBaselineOuterClass.ProcessBaseline> lockProcessBaselines(
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

            def fromUpdate = getProcessBaselineService().lockProcessBaselines(lockRequest).baselinesList

            ProcessBaselineServiceOuterClass.GetProcessBaselineRequest getRequest =
                    ProcessBaselineServiceOuterClass.GetProcessBaselineRequest
                            .newBuilder()
                                .setKey(keyToLock)
                            .build()

            ProcessBaselineOuterClass.ProcessBaseline wl =
                    getProcessBaselineService().getProcessBaseline(getRequest)

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
            log.warn("Error locking process baselines ", e)
        }
    }

    static deleteProcessBaselines(String query) {
        DeleteProcessBaselinesRequest req = DeleteProcessBaselinesRequest.newBuilder()
            .setQuery(query).setConfirm(true).build()
        return getProcessBaselineService().deleteProcessBaselines(req)
    }

    static List<ProcessBaselineOuterClass.ProcessBaseline> updateProcessBaselines(
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
            List<ProcessBaselineOuterClass.ProcessBaseline> updatedLst = getProcessBaselineService()
                .updateProcessBaselines(requestBuilder.build()).baselinesList
            return updatedLst
        } catch (Exception e) {
            log.warn("Error updating process baselines", e)
        }
    }

    static ProcessBaselineOuterClass.ProcessBaseline getBaselineProcesses(
        ProcessBaselineServiceOuterClass.GetProcessBaselineRequest request) {
        try {
            return getProcessBaselineService().getProcessBaseline(request)
        }
        catch (Exception e) {
            log.warn("Error getting  process baselines", e)
        }
        return null
    }

    static boolean waitForDeploymentBaselinesCreated(String clusterId, Deployment deployment, String containerName) {
        Timer t = new Timer(20, 6)
        try {
            while (t.IsValid()) {
                ProcessBaselineOuterClass.ProcessBaseline baseline =
                        getProcessBaseline(clusterId, deployment, containerName)
                if (baseline != null) {
                    return true
                }
            }
            log.debug "Did not find baselines for deployment ${deployment.getDeploymentUid()}"
        }
        catch (Exception e) {
            log.warn("Error waiting for deployment baselines to be created", e)
        }
        return false
    }

    static boolean waitForDeploymentBaselinesDeleted(String clusterId, Deployment deployment, String containerName) {
        Timer t = new Timer(5, 2)
        try {
            while (t.IsValid()) {
                ProcessBaselineOuterClass.ProcessBaseline baseline =
                        getProcessBaseline(clusterId, deployment, containerName, 1)
                if (baseline == null) {
                    return true
                }
            }
            log.debug "Baselines still exist for deployment ${deployment.getDeploymentUid()}"
        }
        catch (Exception e) {
            log.warn("Error waiting for deployment baselines to be deleted", e)
        }
        return false
    }
}
