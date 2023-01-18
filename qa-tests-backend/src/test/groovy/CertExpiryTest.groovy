import services.CredentialExpiryService
import util.Cert

import spock.lang.Tag
import spock.lang.IgnoreIf
import util.Env

@Tag("BAT")
@Tag("COMPATIBILITY")
// ROX-14228 skipping tests for 1st release on power & z
@IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
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
        def scannerTLSSecret = orchestrator.getSecret("scanner-tls", "stackrox")
        assert scannerTLSSecret
        def scannerCertExpiryFromCentral = new Date(CredentialExpiryService.getScannerCertExpiry().getSeconds() * 1000)
        assert scannerCertExpiryFromCentral

        then:
        "Make sure they match"
        assert Cert.loadBase64EncodedCert(scannerTLSSecret.data["cert.pem"]).notAfter == scannerCertExpiryFromCentral
    }

}

