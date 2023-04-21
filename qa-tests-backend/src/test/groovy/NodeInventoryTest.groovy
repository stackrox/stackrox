import io.stackrox.proto.storage.NodeOuterClass.Node

import services.BaseService
import services.ClusterService
import services.NodeService

import spock.lang.Shared
import spock.lang.Tag

class NodeInventoryTest extends BaseSpecification {
    @Shared
    private String clusterId

    def setupSpec() {
        BaseService.useBasicAuth()

        // Get cluster ID
        clusterId = ClusterService.getClusterId()
        assert clusterId
    }

    @Tag("BAT")
    def "Verify node inventories and their scans"() {
        given:
        "given a list of nodes"
        List<Node> nodes = NodeService.getNodes()

        expect:
        "confirm the number of components in the inventory and their scan"
        assert nodes.size() > 0, "Expected to find at least one node"
        nodes.each { node ->
            assert node.getScan(), "Expected to find a nodeScan on the node"
            log.info("Node ${node.getName()} scan contains ${node.getScan().getComponentsList().size()} components")

            if (!ClusterService.isOpenShift4()) {
                // No RHCOS node scanning on this cluster
                assert node.getScan().getComponentsList().size() == 4,
                    "Expected to find exactly 4 components on non-RHCOS node"
                return
            }
            assert node.getScan().getComponentsList().size() > 4,
                "Expected to find more than 4 components on RHCOS node"

            // assume that there must be at least one vulnerability within all the components
            assert node.getScan().getComponentsList().sum { it.getVulnerabilitiesList().size() }
                > 0, "Expected to find at least one vulnerability among the components"
        }
    }
}
