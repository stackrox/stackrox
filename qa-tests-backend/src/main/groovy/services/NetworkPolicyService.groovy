package services

import static util.Helpers.evaluateWithRetry

import groovy.util.logging.Slf4j
import io.grpc.StatusRuntimeException

import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.NetworkGraphScope
import io.stackrox.proto.api.v1.NetworkPolicyServiceGrpc
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.ApplyNetworkPolicyYamlRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GenerateNetworkPoliciesRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GenerateNetworkPoliciesRequest.DeleteExistingPoliciesMode
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GetNetworkGraphRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GetNetworkPoliciesRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GetUndoModificationRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.SendNetworkPolicyYamlRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.SimulateNetworkGraphRequest
import io.stackrox.proto.storage.NetworkPolicyOuterClass.NetworkPolicy
import io.stackrox.proto.storage.NetworkPolicyOuterClass.NetworkPolicyModification
import io.stackrox.proto.storage.NetworkPolicyOuterClass.NetworkPolicyReference

import util.Timer

@Slf4j
class NetworkPolicyService extends BaseService {

    static getNetworkPolicyClient() {
        return NetworkPolicyServiceGrpc.newBlockingStub(getChannel())
    }

    static getNetworkPolicyGraph(String query = null, String scopeQuery = null) {
        try {
            GetNetworkGraphRequest.Builder request =
                    GetNetworkGraphRequest.newBuilder()
                            .setClusterId(ClusterService.getClusterId())
            if (query != null) {
                request.setQuery(query)
            }
            if (scopeQuery != null) {
                request.setScope(NetworkGraphScope.newBuilder().setQuery(scopeQuery))
            }
            return getNetworkPolicyClient().getNetworkGraph(request.build())
        } catch (Exception e) {
            log.error("Exception fetching network policy graph", e)
        }
    }

    static List<NetworkPolicy> getNetworkPolicies() {
        return getNetworkPolicyClient().getNetworkPolicies(
                GetNetworkPoliciesRequest.newBuilder()
                        .setClusterId(ClusterService.getClusterId()).build()
        ).networkPoliciesList
    }

    static submitNetworkGraphSimulation(
            String yaml,
            String query = null,
            String scopeQuery = null,
            List<NetworkPolicyReference> toDelete = null) {
        log.debug("Generating simulation using YAML: %s", yaml)
        try {
            NetworkPolicyModification.Builder mod = NetworkPolicyModification.newBuilder()
                    .setApplyYaml(yaml)
            if (toDelete != null) {
                mod.addAllToDelete(toDelete)
            }
            SimulateNetworkGraphRequest.Builder request =
                    SimulateNetworkGraphRequest.newBuilder()
                            .setClusterId(ClusterService.getClusterId())
                            .setModification(mod)
                            .setIncludeNodeDiff(true)
            if (query != null) {
                request.setQuery(query)
            }
            if (scopeQuery != null) {
                request.setScope(NetworkGraphScope.newBuilder().setQuery(scopeQuery))
            }
            return getNetworkPolicyClient().simulateNetworkGraph(request.build())
        } catch (Exception e) {
            log.error("Error generating simulation using YAML", e)
        }
    }

    static sendSimulationNotification(
            List<String> notifierIds,
            String yaml,
            String clusterId = ClusterService.getClusterId()) {
        try {
            SendNetworkPolicyYamlRequest.Builder request =
                    SendNetworkPolicyYamlRequest.newBuilder()
            if (notifierIds != null) {
                for (String notifierId : notifierIds) {
                    request.addNotifierIds(notifierId)
                }
            }
            clusterId == null ?: request.setClusterId(clusterId)
            yaml == null ?: request.setModification(NetworkPolicyModification.newBuilder().setApplyYaml(yaml))
            return getNetworkPolicyClient().sendNetworkPolicyYAML(request.build())
        } catch (Exception e) {
            log.error(e.toString(), e)
            assert e instanceof StatusRuntimeException
        }
    }

    static waitForNetworkPolicy(String id, int timeoutSeconds = 30) {
        int intervalSeconds = 1
        int retries = timeoutSeconds / intervalSeconds
        try {
            evaluateWithRetry(retries, intervalSeconds) {
                getNetworkPolicyClient().getNetworkPolicy(ResourceByID.newBuilder().setId(id).build())
            }
            return true
        } catch (Exception e) {
            log.error("SR did not detect the network policy", e)
        }
        return false
    }

    static waitForNetworkPolicyRemoval(String id, int timeoutSeconds = 60) {
        Timer t = new Timer((timeoutSeconds / 2) as int, 2)
        while (t.IsValid()) {
            try {
                getNetworkPolicyClient().getNetworkPolicy(ResourceByID.newBuilder().setId(id).build())
            } catch (Exception e) {
                log.error("SR did not detect the network policy", e)
                return true
            }
        }

        log.warn("SR still detects the network policy")
        return false
    }

    static generateNetworkPolicies(
            DeleteExistingPoliciesMode deleteMode = DeleteExistingPoliciesMode.NONE,
            String query = "", String clusterId = ClusterService.getClusterId()) {
        return getNetworkPolicyClient().generateNetworkPolicies(
                GenerateNetworkPoliciesRequest.newBuilder()
                        .setClusterId(clusterId)
                        .setDeleteExisting(deleteMode)
                        .setQuery(query ?: "")
                        .build()).modification
    }

    static applyGeneratedNetworkPolicy(
            NetworkPolicyModification mod,
            String clusterId = ClusterService.getClusterId()) {
        getNetworkPolicyClient().applyNetworkPolicy(ApplyNetworkPolicyYamlRequest.newBuilder()
                .setClusterId(clusterId)
                .setModification(mod)
                .build())
    }

    static undoGeneratedNetworkPolicy(String clusterId = ClusterService.getClusterId()) {
        try {
            return getNetworkPolicyClient().getUndoModification(GetUndoModificationRequest.newBuilder()
                    .setClusterId(clusterId)
                    .build()).undoRecord
        } catch (Exception e) {
            log.error("Network Policy undo failed!", e)
        }
    }
}
