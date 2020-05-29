import com.google.protobuf.Timestamp
import com.google.protobuf.UnknownFieldSet
import groups.Upgrade
import io.stackrox.proto.api.v1.SummaryServiceOuterClass
import io.stackrox.proto.storage.ClusterOuterClass
import io.stackrox.proto.storage.ProcessIndicatorOuterClass
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ConfigService
import services.GraphQLService
import services.ProcessService
import services.SummaryService
import spock.lang.Unroll
import util.Env

class UpgradesTest extends BaseSpecification {
    private final static String CLUSTERID = Env.mustGet("UPGRADE_CLUSTER_ID")

    private static final COMPLIANCE_QUERY = """query getAggregatedResults(
        \$groupBy: [ComplianceAggregation_Scope!],
        \$unit: ComplianceAggregation_Scope!,
        \$where: String) {
            results: aggregatedResults(groupBy: \$groupBy, unit: \$unit, where: \$where) {
                results {
                    aggregationKeys {
                          id
                    }
                    unit
                }
            }
        }"""

    @Category(Upgrade)
    def "Verify cluster exists and that field values are retained"() {
        given:
        "Only run on specific upgrade from 2.4.16"
        Assume.assumeTrue(CLUSTERID=="260e11a3-cbea-464c-95f0-588fa7695b49")

        expect:
        def clusters = ClusterService.getClusters()
        clusters.size() == 1
        def expectedCluster = ClusterOuterClass.Cluster.newBuilder()
                .setId(CLUSTERID)
                .setName("remote")
                .setType(ClusterOuterClass.ClusterType.KUBERNETES_CLUSTER)
                .setPriority(1)
                .setMainImage("stackrox/main:2.4.16.4")
                .setCentralApiEndpoint("central.stackrox:443")
                .setCollectionMethod(ClusterOuterClass.CollectionMethod.KERNEL_MODULE)
                .setRuntimeSupport(true)
                .setTolerationsConfig(ClusterOuterClass.TolerationsConfig.newBuilder()
                        .setDisabled(true)
                        .build())
                .setStatus(ClusterOuterClass.ClusterStatus.newBuilder()
                        .setLastContact(Timestamp.newBuilder().setSeconds(1551412107).setNanos(857477786).build())
                        .setProviderMetadata(ClusterOuterClass.ProviderMetadata.newBuilder()
                                .setGoogle(ClusterOuterClass.GoogleProviderMetadata.newBuilder()
                                        .setProject("ultra-current-825")
                                        .setClusterName("setup-devde6c6")
                                        .build())
                                .setRegion("us-west1")
                                .setZone("us-west1-c")
                                .build())
                        .setOrchestratorMetadata(ClusterOuterClass.OrchestratorMetadata.newBuilder()
                                .setVersion("v1.11.7-gke.4")
                                .setBuildDate(Timestamp.newBuilder().setSeconds(1549394549).build())
                                .build())
                        .build())
                .setDynamicConfig(ClusterOuterClass.DynamicClusterConfig.newBuilder()
                        .setAdmissionControllerConfig(ClusterOuterClass.AdmissionControllerConfig.newBuilder()
                                .setTimeoutSeconds(3)))
                .build()

        def cluster = ClusterOuterClass.Cluster.newBuilder(clusters.get(0))
                .setUnknownFields(UnknownFieldSet.defaultInstance)
                .build()
        cluster == expectedCluster
    }

    @Category(Upgrade)
    def "Verify process indicators have cluster IDs and namespaces added"() {
        given:
        "Only run on specific upgrade from 2.4.16"
        Assume.assumeTrue(CLUSTERID=="260e11a3-cbea-464c-95f0-588fa7695b49")

        expect:
        "Migrated ProcessIndicators to have a cluster ID and a namespace"
        def processIndicators = ProcessService.getProcessIndicatorsByDeployment("33b3eb66-3bd4-11e9-b563-42010a8a0101")
        processIndicators.size() > 0
        for (ProcessIndicatorOuterClass.ProcessIndicator indicator : processIndicators) {
            assert(indicator.getClusterId() == CLUSTERID)
            assert(indicator.getNamespace() != "")
        }
    }

    @Category(Upgrade)
    def "Verify private config contains the correct retention duration for alerts and images"() {
        given:
        "Only run on specific upgrade from 2.4.16"
        Assume.assumeTrue(CLUSTERID=="260e11a3-cbea-464c-95f0-588fa7695b49")

        expect:
        "Alert retention duration is nil, image rentention duration is 7 days"
        def config = ConfigService.getConfig()
        config != null
        config.getPrivateConfig().getAlertConfig() != null
        config.getPrivateConfig().getAlertConfig().getAllRuntimeRetentionDurationDays() == 0
        config.getPrivateConfig().getAlertConfig().getResolvedDeployRetentionDurationDays() == 0
        config.getPrivateConfig().getAlertConfig().getDeletedRuntimeRetentionDurationDays() == 0
        config.getPrivateConfig().getImageRetentionDurationDays() == 7
    }

    @Category(Upgrade)
    def "Verify that summary API returns non-zero values on upgrade"() {
        expect:
        "Summary API returns non-zero values on upgrade"
        SummaryServiceOuterClass.SummaryCountsResponse resp = SummaryService.getCounts()
        assert resp.numAlerts != 0
        assert resp.numDeployments != 0
        assert resp.numSecrets != 0
        assert resp.numClusters != 0
        assert resp.numImages != 0
        assert resp.numNodes != 0
    }

    @Unroll
    @Category(Upgrade)
    def "verify that we find the correct number of #resourceType for query"() {
        when:
        "Fetch the #resourceType from GraphQL"
        def gqlService = new GraphQLService()
        def resultRet = gqlService.Call(getQuery(resourceType), [ query: searchQuery ])
        assert resultRet.getCode() == 200
        println "return code " + resultRet.getCode()

        then:
        "Check that we got the correct number of #resourceType from GraphQL "
        assert resultRet.getValue() != null
        def items = resultRet.getValue()[resourceType]
        assert items.size() >= minResults

        where:
        "Data Inputs Are:"
        resourceType      | searchQuery               | minResults
        "policies"        | "Policy:Latest Tag"       | 1
        "nodes"           | "Cluster ID:${CLUSTERID}" | 2
        "violations"      | ""                        | 1
        "secrets"         | "Cluster ID:${CLUSTERID}" | 1
        "deployments"     | "Cluster ID:${CLUSTERID}" | 1
        "images"          | "Cluster ID:${CLUSTERID}" | 1
        "components"      | "Cluster ID:${CLUSTERID}" | 1
        "vulnerabilities" | "Cluster ID:${CLUSTERID}" | 1
    }

    static getQuery(resourceType) {
        return """query get${resourceType}(\$query: String!) {
                ${resourceType} : ${resourceType}(query: \$query) {
                     id
                }
            }"""
    }

    @Unroll
    @Category(Upgrade)
    def "verify that we find the correct number of compliance results"() {
        when:
        "Fetch the compliance results by #unit from GraphQL"
        def gqlService = new GraphQLService()
        def resultRet = gqlService.Call(COMPLIANCE_QUERY, [ groupBy: groupBy, unit: unit ])
        assert resultRet.getCode() == 200
        println "return code " + resultRet.getCode()

        then:
        "Check that we got the correct number of #unit from GraphQL "
        assert resultRet.getValue() != null
        def resultList = resultRet.getValue()["results"]
        assert resultList.size() >= numResults

        where:
        "Data Inputs Are:"
        groupBy                   | unit      | numResults
        ["STANDARD", "CLUSTER"]   | "CHECK"   | 1
        ["STANDARD", "NAMESPACE"] | "CHECK"   | 1
        ["STANDARD", "CLUSTER"]   | "CONTROL" | 1
        ["STANDARD", "NAMESPACE"] | "CONTROL" | 1
    }

    // TODO
    // network flow edges
    // compliance
    // clairify integration
    // slack integration
}
