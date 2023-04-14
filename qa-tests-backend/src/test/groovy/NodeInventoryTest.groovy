import io.stackrox.proto.storage.Compliance.ComplianceRunResults
import io.stackrox.proto.storage.NodeOuterClass.Node

import services.BaseService
import services.ClusterService
import services.NodeService

import spock.lang.Shared
import spock.lang.Tag

class NodeInventoryTest extends BaseSpecification {
    @Shared
    private static final Map<String, ComplianceRunResults> BASE_RESULTS = [:]
    @Shared
    private String clusterId
    @Shared
    private Map<String, String> standardsByName = [:]

    def setupSpec() {
        BaseService.useBasicAuth()

        // Get cluster ID
        clusterId = ClusterService.getClusterId()
        assert clusterId
    }

    @Override
    def cleanupSpec() {}

    @Tag("BAT")
    def "Verify nodes and node inventories"() {
        given:
        "given a list of nodes and inventories"
        List<Node> nodes = NodeService.getNodes()

        expect:
        "confirm the number of nodes and the inventories"
        assert nodes.size() > 0
        for (def node : nodes) {
            log.info("we got node: {}", node.getName())
            log.info("we got node scan: {}", node.getScan())
            for (def scan : node.getScan()) {
                log.info("scan components: {}", scan.getComponentsList())
                for (def comp : scan.getComponentsList()) {
                    log.info("component vulnerablities: {}", comp.getVulnerabilities())
                }
            }
        }
    }
}
