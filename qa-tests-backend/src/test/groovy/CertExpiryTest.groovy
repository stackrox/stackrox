import groups.BAT
import groups.COMPATIBILITY
import org.junit.experimental.categories.Category
import services.CredentialExpiryService
import util.Cert

@Category([BAT, COMPATIBILITY])
class CertExpiryTest extends BaseSpecification {
    def "Failing test - Delete before merging"() {
        when:
        "This test is supposed to fail"

        then:
        "Check the behavior of the fail-fast flag in Prow"
        assert False
    }

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
        def scannerTLSSecret = orchestrator.getSecret("scanner-tls", "stackrox")
        assert scannerTLSSecret
        def scannerCertExpiryFromCentral = new Date(CredentialExpiryService.getScannerCertExpiry().getSeconds() * 1000)
        assert scannerCertExpiryFromCentral

        then:
        "Make sure they match"
        assert Cert.loadBase64EncodedCert(scannerTLSSecret.data["cert.pem"]).notAfter == scannerCertExpiryFromCentral
    }

}

