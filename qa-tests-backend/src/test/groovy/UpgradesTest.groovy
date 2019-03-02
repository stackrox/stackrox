import groups.Upgrade
import com.google.protobuf.Timestamp
import io.stackrox.proto.storage.ClusterOuterClass
import org.junit.experimental.categories.Category
import services.ClusterService

@Category([Upgrade])
class UpgradesTest extends BaseSpecification {

    def "Verify cluster exists and that field values are retained"() {
        expect:
        def clusters = ClusterService.getClusters()
        clusters.size() == 1
        def expectedCluster = ClusterOuterClass.Cluster.newBuilder()
            .setId("260e11a3-cbea-464c-95f0-588fa7695b49")
            .setName("remote")
            .setType(ClusterOuterClass.ClusterType.KUBERNETES_CLUSTER)
            .setLastContact(Timestamp.newBuilder().setSeconds(1551412107).setNanos(857477786).build())
            .setMainImage("stackrox/main:2.4.16.4")
            .setRuntimeSupport(true)
            .setMonitoringEndpoint("monitoring.stackrox:443")
            .setCentralApiEndpoint("central.stackrox:443")
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
            .build()

        clusters.get(0).equals(expectedCluster)
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
