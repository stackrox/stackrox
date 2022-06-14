import common.Constants
import groups.BAT
import io.stackrox.proto.api.v1.NamespaceServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.NodeOuterClass.Node
import objects.Namespace
import org.javers.core.Javers
import org.javers.core.JaversBuilder
import org.javers.core.diff.ListCompareAlgorithm
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.NamespaceService
import services.NodeService
import services.SummaryService
import spock.lang.IgnoreIf
import util.Env

class SummaryTest extends BaseSpecification {

    @Category([BAT])
    @IgnoreIf({ Env.CI_JOBNAME.contains("postgres") })
    def "Verify TopNav counts for Nodes, Deployments, and Secrets"() {
        // https://stack-rox.atlassian.net/browse/ROX-6844
        Assume.assumeFalse(ClusterService.isOpenShift4())

        expect:
        "Counts API should match orchestrator details"

        withRetry(10, 6) {
            def stackroxSummaryCounts = SummaryService.getCounts()
            List<String> orchestratorResourceNames = orchestrator.getDeploymentCount() +
                    orchestrator.getDaemonSetCount() +
                    orchestrator.getStaticPodCount() +
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
            }

            assert stackroxSummaryCounts.numDeployments == orchestratorResourceNames.size()
            assert stackroxSummaryCounts.numSecrets == orchestrator.getSecretCount()
            assert stackroxSummaryCounts.numNodes == orchestrator.getNodeCount()
        }
    }

    @Category([BAT])
    @IgnoreIf({ Env.CI_JOBNAME.contains("postgres") })
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
                        orchestratorTruncated[name] = orchestratorTruncated[name].substring(0,
                                Constants.STACKROX_NODE_ANNOTATION_TRUNCATION_LENGTH - 1) + "..."
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

    @Category([BAT])
    def "Verify namespace details"() {
        // https://stack-rox.atlassian.net/browse/ROX-6844
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
