import groups.BAT
import org.junit.experimental.categories.Category
import services.ClusterService
import spock.lang.Stepwise

import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate

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
        def sensorCert = (X509Certificate) CertificateFactory.getInstance("X.509").generateCertificate(
            new ByteArrayInputStream(sensorTLSSecret.data["sensor-cert.pem"].decodeBase64())
        )
        def expiryFromCert = sensorCert.notAfter
        assert expiryFromCert
        assert expiryFromCert == expiryFromCluster
    }
}
