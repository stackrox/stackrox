import org.junit.Assume
import services.CredentialExpiryService
import util.Cert

import spock.lang.Tag
import spock.lang.IgnoreIf
import util.Env
import util.Helpers

@Tag("BAT")
@Tag("COMPATIBILITY")
@Tag("PZ")
// skip if executed in a test environment with just secured-cluster deployed in the test cluster
// i.e. central is deployed elsewhere
@IgnoreIf({ Env.ONLY_SECURED_CLUSTER == "true" })
class CertExpiryTest extends BaseSpecification {
    def "Test Central cert expiry"() {
        when:
        "Fetch the current central-tls secret, and the central cert expiry as returned by Central"
        def centralTLSSecret = orchestrator.getSecret("central-tls", "stackrox")
        assert centralTLSSecret
        def centralCertExpiryFromCentral = new Date(CredentialExpiryService.getCentralCertExpiry().getSeconds() * 1000)
        assert centralCertExpiryFromCentral

        then:
        "Make sure they match"
        assert Cert.loadBase64EncodedCert(centralTLSSecret.data["cert.pem"]).notAfter == centralCertExpiryFromCentral
    }

    def "Test Scanner V4 cert expiry"() {
        when:
        "Fetch Scanner V4 TLS secrets and the expiry Central reports for SCANNER_V4"
        def indexerTLSSecret = orchestrator.getSecret("scanner-v4-indexer-tls", "stackrox")
        def matcherTLSSecret = orchestrator.getSecret("scanner-v4-matcher-tls", "stackrox")
        Assume.assumeTrue("Scanner V4 TLS secrets are present",
                indexerTLSSecret != null && matcherTLSSecret != null)
        // Retry since scanner integration registration happens asynchronously.
        def scannerCertExpiryFromCentral = Helpers.evaluateWithRetry(5, 5) {
            return new Date(CredentialExpiryService.getScannerV4CertExpiry().getSeconds() * 1000)
        }
        assert scannerCertExpiryFromCentral

        then:
        "Make sure Central's expiry is the earlier of indexer and matcher leaf certs"
        def indexerExpiry = Cert.loadBase64EncodedCert(indexerTLSSecret.data["cert.pem"]).notAfter
        def matcherExpiry = Cert.loadBase64EncodedCert(matcherTLSSecret.data["cert.pem"]).notAfter
        assert [indexerExpiry, matcherExpiry].min() == scannerCertExpiryFromCentral
    }

    def "Test Scanner cert expiry"() {
        when:
        "Fetch Scanner V2 TLS secret on mixed-version compatibility installs that still deploy V2"
        Assume.assumeTrue("Scanner V4 is absent so the V2 expiry path applies",
                orchestrator.getSecret("scanner-v4-indexer-tls", "stackrox") == null)
        def scannerTLSSecret = orchestrator.getSecret("scanner-tls", "stackrox")
        Assume.assumeTrue("Scanner V2 TLS secret is present", scannerTLSSecret != null)
        def scannerCertExpiryFromCentral = Helpers.evaluateWithRetry(5, 5) {
            return new Date(CredentialExpiryService.getScannerCertExpiry().getSeconds() * 1000)
        }
        assert scannerCertExpiryFromCentral

        then:
        "Make sure they match"
        assert Cert.loadBase64EncodedCert(scannerTLSSecret.data["cert.pem"]).notAfter == scannerCertExpiryFromCentral
    }

}
