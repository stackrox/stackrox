import static util.Helpers.waitForTrue
import static util.Helpers.withRetry

import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.storage.NodeOuterClass.Node

import org.junit.Assume

import common.Constants
import services.BaseService
import services.NodeService
import util.Env

import spock.lang.IgnoreIf
import spock.lang.Requires
import spock.lang.Tag

// Central's scanner-v4-matcher is required to enrich index reports into node scans.
@IgnoreIf({ Env.ONLY_SECURED_CLUSTER == "true" })
// V4 node index reports are produced on OpenShift (RHCOS/RHEL RPM DBs).
@Requires({ Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT })
@Tag("PZ")
class NodeIndexTest extends BaseSpecification {
    def setupSpec() {
        BaseService.useBasicAuth()
    }

    @Tag("BAT")
    def "Verify node index scans"() {
        given:
        "Scanner V4 is enabled and the cluster has nodes"
        Assume.assumeTrue("Scanner V4 node indexing is required", scannerV4Enabled)
        List<Node> nodes = NodeService.getNodes()
        assert nodes.size() > 0

        when:
        "scanner-v4-matcher is ready so index reports can be enriched"
        log.info("Waiting for scanner-v4-matcher deployment to be ready")
        waitForTrue(20, 6) {
            orchestrator.deploymentReady(Constants.STACKROX_NAMESPACE, "scanner-v4-matcher")
        }
        // Matcher-not-ready drops the first index report as unretryable. CI deploy
        // sets ROX_NODE_SCANNING_MAX_INITIAL_WAIT=1s and ROX_NODE_SCANNING_INTERVAL=30s
        // so a later scan lands after matcher is up without restarting collector.

        then:
        "each OpenShift node scan has RPM packages, not only kubelet/kernel/runtime"
        withRetry(12, 30) {
            nodes = NodeService.getNodes()
            assert nodes.size() > 0, "Expected to find at least one node"
            nodes.each { node ->
                assert node.getScan(), "Expected to find a nodeScan on the node"
                int n = node.getScan().getComponentsList().size()
                log.info("Node ${node.getName()} scan contains ${n} components")
                assert n > 4, "Expected to find more than 4 components on OpenShift node"
            }
        }
    }
}
