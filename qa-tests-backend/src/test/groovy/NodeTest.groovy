import io.stackrox.proto.storage.NodeOuterClass

import services.ClusterService
import services.NodeService
import util.Helpers

import spock.lang.Tag

@Tag("PZ")
class NodeTest extends BaseSpecification {

    private static Map<String, String> filterVolatileLabels(Map<String, String> labels) {
        return labels.findAll { k, v -> !k.startsWith("image-prefetcher.stackrox.io/") }
    }

    @Tag("BAT")
    def "Verify node details"() {
        given:
        "fetch the list of nodes"
        List<NodeOuterClass.Node> stackroxNodes = NodeService.getNodes()
        List<objects.Node> orchestratorNodes = orchestrator.getNodeDetails()

        expect:
        "verify Node Details"
        assert stackroxNodes.size() == orchestratorNodes.size()
        for (NodeOuterClass.Node stackroxNode : stackroxNodes) {
            objects.Node orchestratorNode = orchestratorNodes.find { it.uid == stackroxNode.id }
            assert stackroxNode.clusterId == ClusterService.getClusterId()
            assert stackroxNode.name == orchestratorNode.name
            def stableStackroxLabels = filterVolatileLabels(stackroxNode.labelsMap)
            def stableOrchestratorLabels = filterVolatileLabels(orchestratorNode.labels)
            if (stableStackroxLabels != stableOrchestratorLabels) {
                log.info "There is a node label difference"
                // Javers helps provide an useful error in the test log
                def diff = JAVERS.compare(stableStackroxLabels, stableOrchestratorLabels)
                assert diff.changes.size() == 0
                assert diff.changes.size() != 0 // should not get here
            }
            assert stableStackroxLabels == stableOrchestratorLabels
            // compareAnnotations() - asserts on difference
            Helpers.compareAnnotations(orchestratorNode.annotations, stackroxNode.getAnnotationsMap())
            assert stackroxNode.internalIpAddressesList == orchestratorNode.internalIps
            assert stackroxNode.externalIpAddressesList == orchestratorNode.externalIps
            assert stackroxNode.containerRuntimeVersion == orchestratorNode.containerRuntimeVersion
            assert stackroxNode.kernelVersion == orchestratorNode.kernelVersion
            assert stackroxNode.osImage == orchestratorNode.osImage
            assert stackroxNode.kubeletVersion == orchestratorNode.kubeletVersion
            assert stackroxNode.kubeProxyVersion == orchestratorNode.kubeProxyVersion
        }
    }
}
