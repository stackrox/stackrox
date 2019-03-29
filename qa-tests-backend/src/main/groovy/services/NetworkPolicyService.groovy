package services

import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.NetworkPolicyServiceGrpc
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.ApplyNetworkPolicyYamlRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GenerateNetworkPoliciesRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GenerateNetworkPoliciesRequest.DeleteExistingPoliciesMode
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GetNetworkPoliciesRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GetNetworkGraphRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GetUndoModificationRequest
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.SimulateNetworkGraphRequest
import io.stackrox.proto.storage.NetworkPolicyOuterClass.NetworkPolicyModification
import io.stackrox.proto.storage.NetworkPolicyOuterClass.NetworkPolicyReference
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.SendNetworkPolicyYamlRequest

import io.stackrox.proto.storage.NetworkPolicyOuterClass.NetworkPolicy

class NetworkPolicyService extends BaseService {

    static getNetworkPolicyClient() {
        return NetworkPolicyServiceGrpc.newBlockingStub(getChannel())
    }

    static getNetworkPolicyGraph(String query = null) {
        try {
            GetNetworkGraphRequest.Builder request =
                    GetNetworkGraphRequest.newBuilder()
                            .setClusterId(ClusterService.getClusterId())
            if (query != null) {
                request.setQuery(query)
            }
            return getNetworkPolicyClient().getNetworkGraph(request.build())
        } catch (Exception e) {
            println "Exception fetching network policy graph: ${e.toString()}"
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
            List<NetworkPolicyReference> toDelete = null) {
        println "Generating simulation using YAML:"
        println yaml
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
            if (query != null) {
                request.setQuery(query)
            }
            return getNetworkPolicyClient().simulateNetworkGraph(request.build())
        } catch (Exception e) {
            println e.toString()
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
            println e.toString()
            assert e instanceof StatusRuntimeException
        }
    }

    static waitForNetworkPolicy(String id, int timeoutSeconds = 30) {
        int intervalSeconds = 1
        int waitTime
        for (waitTime = 0; waitTime < timeoutSeconds / intervalSeconds; waitTime++) {
            try {
                getNetworkPolicyClient().getNetworkPolicy(ResourceByID.newBuilder().setId(id).build())
                return true
            } catch (Exception e) {
                println "Exception checking for NetworkPolicy in SR, retrying...:"
                println e.toString()
                sleep(intervalSeconds * 1000)
            }
        }

        println "SR did not detect the network policy"
        return false
    }

    static generateNetworkPolicies(
            DeleteExistingPoliciesMode deleteMode = DeleteExistingPoliciesMode.NONE,
            String clusterId = ClusterService.getClusterId()) {
        try {
            return getNetworkPolicyClient().generateNetworkPolicies(
                    GenerateNetworkPoliciesRequest.newBuilder()
                            .setClusterId(clusterId)
                            .setDeleteExisting(deleteMode)
                            .build()).modification
        } catch (Exception e) {
            println "Network Policy generator failed!: ${e.toString()}"
            return e
        }
    }

    static applyGeneratedNetworkPolicy(
            NetworkPolicyModification mod,
            String clusterId = ClusterService.getClusterId()) {
        try {
            getNetworkPolicyClient().applyNetworkPolicy(ApplyNetworkPolicyYamlRequest.newBuilder()
                    .setClusterId(clusterId)
                    .setModification(mod)
                    .build())
        } catch (Exception e) {
            println "Network Policy apply failed!: ${e.toString()}"
            return  e
        }
    }

    static undoGeneratedNetworkPolicy(String clusterId = ClusterService.getClusterId()) {
        try {
            return getNetworkPolicyClient().getUndoModification(GetUndoModificationRequest.newBuilder()
                    .setClusterId(clusterId)
                    .build()).undoRecord
        } catch (Exception e) {
            println "Network Policy undo failed!: ${e.toString()}"
        }
    }
}
