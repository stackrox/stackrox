import static util.Helpers.shellCmdExitValue

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

    def "Test Scanner cert expiry"() {
        when:
        "Fetch the current scanner-tls secret, and the scanner cert expiry as returned by Central"
        assert shellCmdExitValue("./scripts/ci/is-scanner-v2-available.sh stackrox") == 0
        def scannerTLSSecret = orchestrator.getSecret("scanner-tls", "stackrox")
        assert scannerTLSSecret
        // Retry since scanner integration registration happens asynchronously.
        def scannerCertExpiryFromCentral = Helpers.evaluateWithRetry(5, 5) {
            return new Date(CredentialExpiryService.getScannerCertExpiry().getSeconds() * 1000)
        }
        assert scannerCertExpiryFromCentral

        then:
        "Make sure they match"
        assert Cert.loadBase64EncodedCert(scannerTLSSecret.data["cert.pem"]).notAfter == scannerCertExpiryFromCentral
    }

}

