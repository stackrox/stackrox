import com.google.protobuf.UnknownFieldSet
import groups.Upgrade
import com.google.protobuf.Timestamp
import io.stackrox.proto.storage.ClusterOuterClass
import io.stackrox.proto.storage.ProcessIndicatorOuterClass
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ConfigService
import services.ProcessService

class UpgradesTest extends BaseSpecification {
    static final private String CLUSTERID = "260e11a3-cbea-464c-95f0-588fa7695b49"

    @Category(Upgrade)
    def "Verify cluster exists and that field values are retained"() {
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
                .build()

        def cluster = ClusterOuterClass.Cluster.newBuilder(clusters.get(0))
                .setUnknownFields(UnknownFieldSet.defaultInstance)
                .build()
        cluster == expectedCluster
    }

    @Category(Upgrade)
    def "Verify process indicators have cluster IDs and namespaces added"() {
        expect:
        "Migrated ProcessIndicatos to have a cluster ID and a namespace"
        def processIndicators = ProcessService.getProcessIndicatorsByDeployment("33b3eb66-3bd4-11e9-b563-42010a8a0101")
        processIndicators.size() > 0
        for (ProcessIndicatorOuterClass.ProcessIndicator indicator : processIndicators) {
            assert(indicator.getClusterId() == CLUSTERID)
            assert(indicator.getNamespace() != "")
        }
    }

    @Category(Upgrade)
    def "Verify private config contains the correct retention duration for alerts and images"() {
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

    // TODO
    // deployment (incl risk)
    // image (with CVEs)
    // violation (with/without resolved, and with process)
    // nodes
    // network flow edges
    // process indicator
    // secret
    // compliance
    // clairify integration
    // slack integration
    // policies
}
