import groups.BAT
import io.stackrox.proto.api.v1.NamespaceServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import objects.Namespace
import org.junit.experimental.categories.Category
import services.ClusterService
import services.NamespaceService
import services.NodeService
import services.SummaryService
import io.stackrox.proto.storage.NodeOuterClass.Node

class SummaryTest extends BaseSpecification {

    @Category([BAT])
    def "Verify TopNav counts for Nodes, Deployments, and Secrets"() {
        expect:
        "Counts API should match orchestrator details"

        def start = System.currentTimeMillis()
        // Groovy doesn't have do-while loops, so simulating one here.
        def first = true
        def counts
        def deployments
        while (first ||
            (counts.numDeployments != deployments.size() && (System.currentTimeMillis() - start) < (60 * 1000))) {
            first = false

            counts = SummaryService.getCounts()
            deployments = orchestrator.getDeploymentCount() +
                orchestrator.getDaemonSetCount() +
                orchestrator.getStaticPodCount()
        }

        def deploymentNames = Services.getDeployments()*.name
        println "SR Deployments: ${deploymentNames.sort()}"
        println "Actual Deployments: ${deployments.sort()}"
        assert counts.numDeployments == deployments.size()
        assert counts.numSecrets == orchestrator.getSecretCount()
        assert counts.numNodes == orchestrator.getNodeCount()
    }

    @Category([BAT])
    def "Verify node details"() {
        given:
        "fetch the list of nodes"
        List<Node> stackroxNodes = NodeService.getNodes()
        List<objects.Node> orchNodes = orchestrator.getNodeDetails()

        expect:
        "verify Node Details"
        assert stackroxNodes.size() == orchNodes.size()
        for (Node node : stackroxNodes) {
            objects.Node actualNode = orchNodes.find { it.uid == node.id }
            assert node.clusterId == ClusterService.getClusterId()
            assert node.name == actualNode.name
            assert node.labelsMap == actualNode.labels
            assert node.annotationsMap == actualNode.annotations
            assert node.internalIpAddressesList == actualNode.internalIps
            assert node.externalIpAddressesList == actualNode.externalIps
            assert node.containerRuntimeVersion == actualNode.containerRuntimeVersion
            assert node.kernelVersion == actualNode.kernelVersion
            assert node.osImage == actualNode.osImage
        }
    }

    @Category([BAT])
    def "Verify namespace details"() {
        given:
        "fetch the list of namespace"

        List<Namespace> orchNamespaces = orchestrator.getNamespaceDetails()
        Namespace orchQANamespace = orchNamespaces.find { it.name == "qa" }.collect().first()
        NamespaceService.waitForNamespace(orchQANamespace.uid)

        List<NamespaceServiceOuterClass.Namespace> stackroxNamespaces = NamespaceService.getNamespaces()

        expect:
        "verify Node Details"
        assert stackroxNamespaces.size() == orchNamespaces.size()
        for (NamespaceServiceOuterClass.Namespace ns : stackroxNamespaces) {
            def start = System.currentTimeMillis()
            Namespace actualNamespace = orchNamespaces.find { it.uid == ns.metadata.id }
            while (ns.numDeployments != actualNamespace.deploymentCount.size() &&
                    (System.currentTimeMillis() - start) < (60 * 1000)) {
                ns = NamespaceService.getNamespace(ns.metadata.id)
            }
            def deploymentNames = Services.getDeployments(
                    SearchServiceOuterClass.RawQuery.newBuilder().setQuery("Namespace:${ ns.metadata.name }").build()
            )*.name
            println "SR deployments in ${ns.metadata.name}: ${deploymentNames.sort()}"
            println "Actual deployments in ${ns.metadata.name}: ${actualNamespace.deploymentCount.sort()}"
            assert ns.metadata.clusterId == ClusterService.getClusterId()
            assert ns.metadata.name == actualNamespace.name
            assert ns.metadata.labelsMap == actualNamespace.labels
            assert ns.numDeployments == actualNamespace.deploymentCount.size()
            assert ns.numSecrets == actualNamespace.secretsCount
            assert ns.numNetworkPolicies == actualNamespace.networkPolicyCount
        }
    }
}
