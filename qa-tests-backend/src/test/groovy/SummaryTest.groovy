import static util.Helpers.withRetry

import org.javers.core.Javers
import org.javers.core.JaversBuilder
import org.javers.core.diff.ListCompareAlgorithm

import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.NodeOuterClass.Node

import services.ClusterService
import services.NamespaceService
import services.NodeService
import services.SummaryService
import util.Helpers

import org.junit.Assume
import spock.lang.Ignore
import spock.lang.IgnoreIf
import spock.lang.Tag

@Tag("PZ")
class SummaryTest extends BaseSpecification {

    @Tag("BAT")
    @Tag("COMPATIBILITY")
    // Temporarily enable this test to gather debug data and fix the flaky behavior
    // @Ignore("ROX-24528: This API is deprecated in 4.5. Remove this test once the API is removed")
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

            // For Openshift, there is the following discrepancy here:
            // Stackrox has: 'kube-rbac-proxy-crio'
            // Openshift has: 'kube-rbac-proxy-crio-<clustername>-gtnb7-master-2.c.acs-team-temp-dev.internal'
            // (for each node)

            if (stackroxSummaryCounts.numDeployments != orchestratorResourceNames.size()) {
                log.info "The summary count for deployments in ACS does not equal the orchestrator count."
                log.info "ACS count: ${stackroxSummaryCounts.numDeployments}, " +
                        "orchestrator count ${orchestratorResourceNames.size()}"
                log.info "This diff may help with debug, however deployment names may be different between APIs"
                log.info "In this diff, 'removed' means 'missing in orchestrator but given in ACS', " +
                    "whereas 'added' - the other way round"
                List<String> stackroxDeploymentNames = Services.getDeployments()*.name
                Javers javers = JaversBuilder.javers()
                        .withListCompareAlgorithm(ListCompareAlgorithm.AS_SET)
                        .build()
                log.info javers.compare(stackroxDeploymentNames.sort(), orchestratorResourceNames.sort()).prettyPrint()

                log.info "Use the full set of deployments to compare manually if diff isn't helpful"
                log.info "ACS deployments: " + stackroxDeploymentNames.sort().join(",")
                log.info "Orchestrator deployments: " + orchestratorResourceNames.sort().join(",")
            }

            assert Math.abs(stackroxSummaryCounts.numDeployments - orchestratorResourceNames.size()) <= 2
            List<String> stackroxSecretNames = Services.getSecrets()*.name
            log.info "ACS secrets: " + stackroxSecretNames.join(",")
            assert Math.abs(stackroxSummaryCounts.numSecrets - orchestrator.getSecretCount()) <= 2
            assert Math.abs(stackroxSummaryCounts.numNodes - orchestrator.getNodeCount()) <= 2
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
        "ACS and Orchestrator have the same namespaces"
        List<String> orchestratorNamespaces = new ArrayList<>()
        Map<String, String> stackroxNamespaces
        withRetry(3, 5) {
            stackroxNamespaces = new HashMap<>()
            orchestratorNamespaces = orchestrator.getNamespaces()

            NamespaceService.getNamespaces().collect {
                stackroxNamespaces.put(it.metadata.name, it.metadata.id)
            }
            if (stackroxNamespaces.keySet().size() != orchestratorNamespaces.size()) {
                log.info "There is a difference in the namespace count"
                log.info "Stackrox has ${stackroxNamespaces.keySet().size()}, " +
                        "the orchestrator has ${orchestratorNamespaces.size()}"
                log.info "In this diff, 'removed' means 'missing in orchestrator but given in ACS', " +
                        "whereas 'added' - the other way round"
                Javers javers = JaversBuilder.javers()
                        .withListCompareAlgorithm(ListCompareAlgorithm.AS_SET)
                        .build()
                log.info javers.compare(stackroxNamespaces.keySet().sort(), orchestratorNamespaces.sort())
                        .prettyPrint()
            }
            assert stackroxNamespaces.keySet().sort() == orchestratorNamespaces.sort()
            assert !orchestratorNamespaces.isEmpty()
        }
        expect:
        "Namespace details should match the resources in the orchestrator"
        for (String ns : orchestratorNamespaces) {
            withRetry(5, 5) {
                // Retrieve the namespace details from ACS and the orchestrator
                def orchestratorNamespaceDetails = orchestrator.getNamespaceDetailsByName(ns)
                def stackroxNamespaceDetails = NamespaceService.getNamespace(stackroxNamespaces.get(ns))

                log.info "Comparing namespace ${ns}"
                if (stackroxNamespaceDetails.numDeployments != orchestratorNamespaceDetails.deploymentCount.size()) {
                    log.info "There is a difference in the deployment count for namespace ${ns}"
                    log.info "Stackrox has ${stackroxNamespaceDetails.numDeployments}, " +
                            "the orchestrator has ${orchestratorNamespaceDetails.deploymentCount.size()}"
                    log.info "This diff may help with debug, however deployment names may be different between APIs"
                    log.info "In this diff, 'removed' means 'missing in orchestrator but given in ACS', " +
                            "whereas 'added' - the other way round"
                    List<String> stackroxDeploymentNames = Services.getDeployments(
                            SearchServiceOuterClass.RawQuery.newBuilder().setQuery(
                                    "Namespace:${ stackroxNamespaceDetails.metadata.name }").build()
                    )*.name
                    Javers javers = JaversBuilder.javers()
                            .withListCompareAlgorithm(ListCompareAlgorithm.AS_SET)
                            .build()
                    log.info javers.compare(stackroxDeploymentNames, orchestratorNamespaceDetails.deploymentCount)
                            .prettyPrint()
                }
                assert stackroxNamespaceDetails.numDeployments == orchestratorNamespaceDetails.deploymentCount.size()
                assert stackroxNamespaceDetails.metadata.clusterId == ClusterService.getClusterId()
                assert stackroxNamespaceDetails.metadata.name == orchestratorNamespaceDetails.name
                assert stackroxNamespaceDetails.metadata.labelsMap == orchestratorNamespaceDetails.labels
                assert stackroxNamespaceDetails.numSecrets == orchestratorNamespaceDetails.secretsCount
                assert stackroxNamespaceDetails.numNetworkPolicies == orchestratorNamespaceDetails.networkPolicyCount
            }
        }
    }
}
