import org.javers.core.Javers
import org.javers.core.JaversBuilder
import org.javers.core.diff.ListCompareAlgorithm

import io.stackrox.proto.api.v1.NamespaceServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.NodeOuterClass.Node

import common.Constants
import objects.Namespace
import services.ClusterService
import services.NamespaceService
import services.NodeService
import util.Helpers

import org.junit.Assume
import spock.lang.IgnoreIf
import spock.lang.Tag

@Tag("PZ")
class SummaryTest extends BaseSpecification {

    @Tag("BAT")
    def "Verify node details"() {
        given:
        "fetch the list of nodes"
        List<Node> stackroxNodes = NodeService.getNodes()
        List<objects.Node> orchestratorNodes = orchestrator.getNodeDetails()

        expect:
        "verify Node Details"
        assert stackroxNodes.size() == orchestratorNodes.size()
        for (Node stackroxNode : stackroxNodes) {
            objects.Node orchestratorNode = orchestratorNodes.find { it.uid == stackroxNode.id }
            assert stackroxNode.clusterId == ClusterService.getClusterId()
            assert stackroxNode.name == orchestratorNode.name
            if (stackroxNode.labelsMap != orchestratorNode.labels) {
                log.info "There is a node label difference"
                // Javers helps provide an useful error in the test log
                Javers javers = JaversBuilder.javers().build()
                def diff = javers.compare(stackroxNode.labelsMap, orchestratorNode.labels)
                assert diff.changes.size() == 0
                assert diff.changes.size() != 0 // should not get here
            }
            assert stackroxNode.labelsMap == orchestratorNode.labels
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

    @Tag("BAT")
    @IgnoreIf({ System.getenv("OPENSHIFT_CI_CLUSTER_CLAIM") == "openshift-4" })
    def "Verify namespace details"() {
        // https://issues.redhat.com/browse/ROX-6844
        Assume.assumeFalse(ClusterService.isOpenShift4())

        given:
        "fetch the list of namespace"

        List<Namespace> orchestratorNamespaces = orchestrator.getNamespaceDetails()
        Namespace qaNamespace = orchestratorNamespaces.find {
            it.name == Constants.ORCHESTRATOR_NAMESPACE
        }
        NamespaceService.waitForNamespace(qaNamespace.uid)

        List<NamespaceServiceOuterClass.Namespace> stackroxNamespaces = NamespaceService.getNamespaces()

        expect:
        "verify Namespace Details"
        assert stackroxNamespaces.size() == orchestratorNamespaces.size()
        Boolean diff = false
        for (NamespaceServiceOuterClass.Namespace stackroxNamespace : stackroxNamespaces) {
            Namespace orchestratorNamespace = orchestratorNamespaces.find {
                it.uid == stackroxNamespace.metadata.id
            }
            def start = System.currentTimeMillis()
            while (stackroxNamespace.numDeployments != orchestratorNamespace.deploymentCount.size() &&
                (System.currentTimeMillis() - start) < (30 * 1000)) {
                stackroxNamespace = NamespaceService.getNamespace(stackroxNamespace.metadata.id)
                log.info "There is a difference in the deployment count for namespace "+
                        stackroxNamespace.metadata.name
                log.info "StackRox has ${stackroxNamespace.numDeployments}, "+
                        "the orchestrator has ${orchestratorNamespace.deploymentCount.size()}"
                log.info "will retry to find equivalence in 5 seconds"
                sleep(5000)
            }
            if (stackroxNamespace.numDeployments != orchestratorNamespace.deploymentCount.size()) {
                log.info "There is a difference in the deployment count for namespace "+
                        stackroxNamespace.metadata.name
                log.info "StackRox has ${stackroxNamespace.numDeployments}, "+
                        "the orchestrator has ${orchestratorNamespace.deploymentCount.size()}"
                log.info "This diff may help with debug, however deployment names may be different between APIs"
                List<String> stackroxDeploymentNames = Services.getDeployments(
                        SearchServiceOuterClass.RawQuery.newBuilder().setQuery(
                                "Namespace:${ stackroxNamespace.metadata.name }").build()
                )*.name
                Javers javers = JaversBuilder.javers()
                        .withListCompareAlgorithm(ListCompareAlgorithm.AS_SET)
                        .build()
                log.info javers.compare(stackroxDeploymentNames, orchestratorNamespace.deploymentCount).prettyPrint()
                diff = true
            }
            assert stackroxNamespace.metadata.clusterId == ClusterService.getClusterId()
            assert stackroxNamespace.metadata.name == orchestratorNamespace.name
            assert stackroxNamespace.metadata.labelsMap == orchestratorNamespace.labels
            assert stackroxNamespace.numSecrets == orchestratorNamespace.secretsCount
            assert stackroxNamespace.numNetworkPolicies == orchestratorNamespace.networkPolicyCount
        }
        assert !diff, "See diff(s) above"
    }
}
