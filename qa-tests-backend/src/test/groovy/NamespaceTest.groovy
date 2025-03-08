import static util.Helpers.withRetry

import org.javers.core.Javers
import org.javers.core.JaversBuilder
import org.javers.core.diff.ListCompareAlgorithm

import io.stackrox.proto.api.v1.SearchServiceOuterClass

import services.ClusterService
import services.NamespaceService

import org.junit.Assume
import spock.lang.IgnoreIf
import spock.lang.Tag

@Tag("PZ")
class NamespaceTest extends BaseSpecification {

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
