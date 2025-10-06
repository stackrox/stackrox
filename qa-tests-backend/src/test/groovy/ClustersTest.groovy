import io.stackrox.proto.storage.ClusterOuterClass

import common.Constants
import services.ClusterService
import util.Cert
import util.Env

import spock.lang.IgnoreIf
import spock.lang.Stepwise
import spock.lang.Tag

@Tag("BAT")
@Tag("PZ")
@IgnoreIf({ Env.IS_BYODB })
@Stepwise
class ClustersTest extends BaseSpecification {

    def "Test cluster status has cert expiry"() {
        when:
        "Get the cluster, and the sensor TLS certificates (sensor-tls and tls-cert-sensor)"
        def cluster = ClusterService.getCluster()
        assert cluster
        def expiryFromCluster = new Date(
            cluster.getStatus().getCertExpiryStatus().getSensorCertExpiry().getSeconds() * 1000
        )
        assert expiryFromCluster

        def expiryFromCert = getCertExpiryFromSecret("sensor-tls", "sensor-cert.pem")
        assert expiryFromCert

        def expiryFromNewCert = null
        try {
            // tls-cert-sensor is a new secret that may exist in the cluster due to certificate rotation.
            expiryFromNewCert = getCertExpiryFromSecret("tls-cert-sensor", "cert.pem")
        } catch (Exception e) {
            log.debug(
                "tls-cert-sensor secret not found or could not be loaded " +
                "(this is expected if cert rotation has not occurred yet): ${e.message}"
            )
        }

        then:
        "Verify the cluster has sensor cert expiry information, and that is matches what's in the secret"
        assert expiryFromCluster == expiryFromCert || (expiryFromNewCert && expiryFromCluster == expiryFromNewCert)
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

    private getCertExpiryFromSecret(String secretName, String certKey) {
        def secret = orchestrator.getSecret(secretName, Constants.STACKROX_NAMESPACE)
        def cert = Cert.loadBase64EncodedCert(secret.data[certKey])
        return cert.notAfter
    }
}
