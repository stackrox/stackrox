import static util.Helpers.waitForTrue
import static util.Helpers.withRetry

import io.fabric8.kubernetes.api.model.Pod
import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.storage.NodeOuterClass.Node
import io.stackrox.proto.storage.NodeOuterClass.NodeScan

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

        when:
        "Sensor and scanner-v4-matcher are ready to process node indexes"
        log.info("Waiting for Sensor and scanner-v4-matcher deployments to be ready")
        waitForTrue(20, 6) {
            orchestrator.deploymentReady(Constants.STACKROX_NAMESPACE, "sensor") &&
                    orchestrator.deploymentReady(Constants.STACKROX_NAMESPACE, "scanner-v4-matcher")
        }

        and:
        "collectors on nodes missing an index get one new initial scan"
        // A failed initial index creation can otherwise wait for the four-hour interval.
        // Recover one node at a time, leaving collectors with valid scans running.
        List<String> nodesToRecover = nodesMissingIndex()
        log.info("Nodes requiring collector recreation for indexing: ${nodesToRecover}")
        nodesToRecover.each { nodeName ->
            // An outstanding report may have arrived while another node recovered.
            if (nodesMissingIndex().contains(nodeName)) {
                restartCollectorForNode(nodeName)
                withRetry(12, 30) {
                    Node node = NodeService.getNodes().find { it.getName() == nodeName }
                    assert node != null, "Node disappeared during index recovery: ${nodeName}"
                    assert hasIndex(node), "Missing node index after recovery: ${nodeName}"
                }
                log.info("Node index recovered after collector recreation: ${nodeName}")
            }
        }

        then:
        "each OpenShift node scan has RPM packages, not only kubelet/kernel/runtime"
        withRetry(12, 30) {
            List<Node> nodes = NodeService.getNodes()
            assert nodes.size() > 0, "Expected to find at least one node"
            nodes.each { node ->
                assert node.hasScan(), "Expected to find a nodeScan on ${node.getName()}"
                assert node.getScan().getScannerVersion() == NodeScan.Scanner.SCANNER_V4,
                        "Expected a Scanner V4 scan on ${node.getName()}"
                int n = node.getScan().getComponentsCount()
                log.info("Node ${node.getName()} scan contains ${n} components")
                assert n > 4, "Expected to find more than 4 components on OpenShift node"
            }
        }
    }

    private List<String> nodesMissingIndex() {
        List<Node> nodes = NodeService.getNodes()
        assert nodes.size() > 0, "Expected to find at least one node"
        return nodes.findAll { !hasIndex(it) }*.getName()
    }

    private static boolean hasIndex(Node node) {
        return node.hasScan() && node.getScan().getScannerVersion() == NodeScan.Scanner.SCANNER_V4 &&
                node.getScan().getComponentsCount() > 4
    }

    private void restartCollectorForNode(String nodeName) {
        String namespace = Constants.STACKROX_NAMESPACE
        Pod pod = orchestrator.getPodsByLabel(namespace, [app: "collector"]).find {
            it.spec.nodeName == nodeName && !it.metadata.deletionTimestamp
        }
        assert pod != null, "Expected a collector pod on ${nodeName}"
        // Do not erase an unexpected restart from the post-test restart checks.
        assert pod.status.containerStatuses.every { it.restartCount == 0 },
                "Collector ${pod.metadata.name} already has container restarts; preserve it for investigation"

        saveComplianceLog(pod.metadata.name)
        log.warn("Recreating collector ${pod.metadata.name} (${pod.metadata.uid}) on ${nodeName} for node indexing")
        orchestrator.deletePodAndWait(namespace, pod.metadata.name, 60, 5)
        waitForTrue(60, 5) {
            orchestrator.getPodsByLabel(namespace, [app: "collector"]).any {
                it.spec.nodeName == nodeName && it.metadata.uid != pod.metadata.uid && orchestrator.podReady(it)
            }
        }
    }

    private void saveComplianceLog(String podName) {
        try {
            File logDir = new File(Env.QA_TEST_DEBUG_LOGS, "node-index-recovery")
            logDir.mkdirs()
            File logFile = new File(logDir, "${podName}-compliance.log")
            logFile.setText(orchestrator.getContainerlogs(Constants.STACKROX_NAMESPACE, podName, "compliance"),
                    "UTF-8")
            log.info("Saved compliance log before collector recreation: ${logFile.absolutePath}")
        } catch (Exception e) {
            log.warn("Could not save compliance log before recreating ${podName}", e)
        }
    }
}
