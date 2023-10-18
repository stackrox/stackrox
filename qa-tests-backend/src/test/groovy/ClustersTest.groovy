import io.stackrox.proto.storage.ClusterOuterClass

import services.ClusterService
import util.Cert

import spock.lang.Stepwise
import spock.lang.Tag

@Tag("BAT")
@Tag("PZ")
@Stepwise
class ClustersTest extends BaseSpecification {

    def "Test cluster status has cert expiry"() {
        when:
        "Get the cluster, and the sensor-tls cert"
        def cluster = ClusterService.getCluster()
        assert cluster
        def sensorTLSSecret = orchestrator.getSecret("sensor-tls", "stackrox")

        then:
        "Verify the cluster has sensor cert expiry information, and that is matches what's in the secret"
        def expiryFromCluster = new Date(
            cluster.getStatus().getCertExpiryStatus().getSensorCertExpiry().getSeconds() * 1000
        )
        assert expiryFromCluster
        def sensorCert = Cert.loadBase64EncodedCert(sensorTLSSecret.data["sensor-cert.pem"])
        def expiryFromCert = sensorCert.notAfter
        assert expiryFromCert
        assert expiryFromCert == expiryFromCluster
    }

    def "Test cluster health status is healthy"() {
        when:
        "Get the cluster, and the cluster health status"
        def cluster = ClusterService.getCluster()
        assert cluster
        def overallClusterHealthStatus = cluster.healthStatus.overallHealthStatus

        then:
        "Verify the cluster's overall health status is healthy"
        assert overallClusterHealthStatus == ClusterOuterClass.ClusterHealthStatus.HealthStatusLabel.HEALTHY
    }
}
