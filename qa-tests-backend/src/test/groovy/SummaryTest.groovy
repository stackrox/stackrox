import static util.Helpers.withRetry

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
import services.SummaryService

import org.junit.Assume
import spock.lang.IgnoreIf
import spock.lang.Tag

@Tag("PZ")
class SummaryTest extends BaseSpecification {

    @Tag("BAT")
    @Tag("COMPATIBILITY")
    @IgnoreIf({ System.getenv("OPENSHIFT_CI_CLUSTER_CLAIM") == "openshift-4" })
    def "Verify TopNav counts for Nodes, Deployments, and Secrets"() {
        // https://issues.redhat.com/browse/ROX-6844
        Assume.assumeFalse(ClusterService.isOpenShift4())

        expect:
        "Counts API should match orchestrator details"

        withRetry(10, 6) {
            def stackroxSummaryCounts = SummaryService.getCounts()
            List<String> orchestratorResourceNames = orchestrator.getDeploymentCount() +
                    orchestrator.getDaemonSetCount() +
                    // Static pods get renamed as "static-<name>-pods" in sensor, so match it for easy debugging
                    orchestrator.getStaticPodCount().collect {  "static-" + it + "-pods"  } +
                    orchestrator.getStatefulSetCount() +
                    orchestrator.getJobCount()

            if (stackroxSummaryCounts.numDeployments != orchestratorResourceNames.size()) {
                log.info "The summary count for deployments does not equate to the orchestrator count."
                log.info "Stackrox count: ${stackroxSummaryCounts.numDeployments}, " +
                        "orchestrator count ${orchestratorResourceNames.size()}"
                log.info "This diff may help with debug, however deployment names may be different between APIs"
                List<String> stackroxDeploymentNames = Services.getDeployments()*.name
                Javers javers = JaversBuilder.javers()
                        .withListCompareAlgorithm(ListCompareAlgorithm.AS_SET)
                        .build()
                log.info javers.compare(stackroxDeploymentNames, orchestratorResourceNames).prettyPrint()

                log.info "Use the full set of deployments to compare manually if diff isn't helpful"
                log.info "Stackrox deployments: " + stackroxDeploymentNames.join(",")
                log.info "Orchestrator deployments: " + orchestratorResourceNames.join(",")
            }

            assert stackroxSummaryCounts.numDeployments == orchestratorResourceNames.size()
            assert stackroxSummaryCounts.numSecrets == orchestrator.getSecretCount()
            assert stackroxSummaryCounts.numNodes == orchestrator.getNodeCount()
        }
    }

    @Tag("BAT")
    def "Verify node details"() {
        given:
        "fetch the list of nodes"
        List<Node> stackroxNodes = NodeService.getNodes()
        List<objects.Node> orchestratorNodes = orchestrator.getNodeDetails()

        expect:
        "verify Node Details"
        assert stackroxNodes.size() == orchestratorNodes.size()
        Boolean diff = false
        Javers javers = JaversBuilder.javers().build()
        for (Node stackroxNode : stackroxNodes) {
            objects.Node orchestratorNode = orchestratorNodes.find { it.uid == stackroxNode.id }
            assert stackroxNode.clusterId == ClusterService.getClusterId()
            assert stackroxNode.name == orchestratorNode.name
            if (stackroxNode.labelsMap != orchestratorNode.labels) {
                log.info "There is a node label difference - StackRox -v- Orchestrator:"
                log.info javers.compare(stackroxNode.labelsMap, orchestratorNode.labels).prettyPrint()
                diff = true
            }
            assert stackroxNode.labelsMap == orchestratorNode.labels
            if (stackroxNode.annotationsMap != orchestratorNode.annotations) {
                Map<String, String> orchestratorTruncated = orchestratorNode.annotations.clone()
                orchestratorTruncated.keySet().each { name ->
                    if (orchestratorTruncated[name].length() > Constants.STACKROX_NODE_ANNOTATION_TRUNCATION_LENGTH) {
                        // Assert that the stackrox node has an entry for that annotation
                        assert stackroxNode.annotationsMap[name].length() > 0

                        // Remove the annotation because the logic for truncation tries to maintain words and
                        // is more complicated than we'd like to test
                        stackroxNode.annotationsMap.remove(name)
                        orchestratorTruncated.remove(name)
                    }
                }
                if (stackroxNode.annotationsMap != orchestratorTruncated) {
                    log.info "There is a node annotation difference - StackRox -v- Orchestrator:"
                    log.info javers.compare(stackroxNode.annotationsMap, orchestratorTruncated).prettyPrint()
                    diff = true
                }
            }
            assert stackroxNode.internalIpAddressesList == orchestratorNode.internalIps
            assert stackroxNode.externalIpAddressesList == orchestratorNode.externalIps
            assert stackroxNode.containerRuntimeVersion == orchestratorNode.containerRuntimeVersion
            assert stackroxNode.kernelVersion == orchestratorNode.kernelVersion
            assert stackroxNode.osImage == orchestratorNode.osImage
            assert stackroxNode.kubeletVersion == orchestratorNode.kubeletVersion
            assert stackroxNode.kubeProxyVersion == orchestratorNode.kubeProxyVersion
        }
        assert !diff, "See diff(s) above"
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
