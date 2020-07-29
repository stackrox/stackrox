import groups.BAT
import org.junit.experimental.categories.Category
import services.ClusterService
import spock.lang.Stepwise
import util.Cert

@Category(BAT)
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
}
