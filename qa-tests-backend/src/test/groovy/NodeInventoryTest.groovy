import static util.Helpers.waitForTrue
import static util.Helpers.withRetry

import com.google.protobuf.Timestamp

import io.stackrox.proto.storage.NodeOuterClass.Node

import common.Constants
import services.BaseService
import services.ClusterService
import services.NodeService
import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Ignore
import spock.lang.IgnoreIf
import util.Env

// skip if executed in a test environment with just secured-cluster deployed in the test cluster
// i.e. central is deployed elsewhere
@IgnoreIf({ Env.ONLY_SECURED_CLUSTER == "true" })
@Ignore("ROX-24871") // After merging PR #11865, the test now fails more often and needs attention
@Tag("PZ")
class NodeInventoryTest extends BaseSpecification {
    @Shared
    private String clusterId

    def setupSpec() {
        BaseService.useBasicAuth()

        // Get cluster ID
        assert ClusterService.getClusters().size() > 0, "There must be at least one secured cluster"
        clusterId = ClusterService.getClusters().get(0).getId()
        assert clusterId
    }

    @Tag("BAT")
    def "Verify node inventories and their scans"() {
        given:
        "given a non-empty list of nodes"
        List<Node> nodes = NodeService.getNodes()
        assert nodes.size() > 0
        def previousScanTime = [:]

        when:
        boolean nodeInventoryContainerAvailable =
            orchestrator.containsDaemonSetContainer(Constants.STACKROX_NAMESPACE, "collector", "node-inventory")
        if (nodeInventoryContainerAvailable) {
            // Sometimes one pod in daemon set will miss the env variable despite updating.
            // Let's try this operation twice before giving up
            waitForTrue(2, 20) {
                log.info("Setting collector.node-inventory ROX_NODE_SCANNING_MAX_INITIAL_WAIT to 1s")
                orchestrator.updateDaemonSetEnv(Constants.STACKROX_NAMESPACE, "collector", "node-inventory",
                    "ROX_NODE_SCANNING_MAX_INITIAL_WAIT", "1s")
                try {
                    log.info("Wait for collector DS to be restarted with new values")
                    waitForTrue(20, 10) {
                        orchestrator.daemonSetEnvVarUpdated(Constants.STACKROX_NAMESPACE, "collector",
                            "node-inventory", "ROX_NODE_SCANNING_MAX_INITIAL_WAIT", "1s")
                    }

                    log.info("Wait for collector DS to be ready")
                    waitForTrue(20, 10) {
                        orchestrator.daemonSetReady(Constants.STACKROX_NAMESPACE, "collector")
                    }
                }
                catch (Exception ignored) {
                    log.info("Unable to bring collector ds to the desired state")
                    return false
                }
                return true
            }
            // Finally, before starting the test, make note of the current scan time, which should be updated
            nodes.each { node ->
                previousScanTime[node.getId()] = node.hasScan() ?
                        node.getScan().getScanTime() : Timestamp.getDefaultInstance()
                log.info("Previous scan time of node ${node.getId()}: ${previousScanTime[node.getId()]}")
            }
        }
        log.info("Waiting for scanner deployment to be ready")
        waitForTrue(20, 6) {
            orchestrator.deploymentReady(Constants.STACKROX_NAMESPACE, "scanner")
        }

        then:
        "confirm the number of components in the inventory and their scan"
        // ensure that the nodes got scanned at least once - retry up to 6 minutes
        withRetry(12, 30) {
            nodes = NodeService.getNodes()
            assert nodes.size() > 0, "Expected to find at least one node"
            nodes.each { node ->
                assert node.getScan().getComponentsList().size() >= 4, "Expected to find at least 4 node components"
            }
        }
        nodes.each { node ->
            assert node.getScan(), "Expected to find a nodeScan on the node"
            log.info("Node ${node.getName()} scan contains ${node.getScan().getComponentsList().size()} components")

            if (!nodeInventoryContainerAvailable) {
                // No RHCOS node scanning on this cluster
                assert node.getScan().getComponentsList().size() == 4,
                    "Expected to find exactly 4 components on non-RHCOS node"
                return
            }
            assert node.getScan().getComponentsList().size() > 4,
                "Expected to find more than 4 components on RHCOS node"

            assert previousScanTime[node.getId()] != node.getScan().getScanTime(),
                "Expected the scan time of the node to have changed"
        }
    }
}
