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
        // Matcher-not-ready drops the first index report as unretryable, so force a
        // prompt rescan after matcher is up. The default first-scan delay is 5m.
        // Sometimes one pod in the daemon set misses the env variable despite updating.
        waitForTrue(2, 20) {
            log.info("Setting collector.compliance ROX_NODE_SCANNING_MAX_INITIAL_WAIT to 1s")
            orchestrator.updateDaemonSetEnv(Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, "compliance",
                "ROX_NODE_SCANNING_MAX_INITIAL_WAIT", "1s")
            try {
                log.info("Wait for collector DS to be restarted with new values")
                waitForTrue(20, 10) {
                    orchestrator.daemonSetEnvVarUpdated(Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS,
                        "compliance", "ROX_NODE_SCANNING_MAX_INITIAL_WAIT", "1s")
                }

                log.info("Wait for collector DS to be ready")
                waitForTrue(20, 10) {
                    orchestrator.daemonSetReady(Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS)
                }
            }
            catch (Exception ignored) {
                log.info("Unable to bring collector ds to the desired state")
                return false
            }
            return true
        }

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
