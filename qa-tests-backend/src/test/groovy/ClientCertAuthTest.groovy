import common.Constants
import groups.BAT
import io.grpc.StatusRuntimeException
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.AuthProviderService
import services.AuthService
import services.BaseService
import services.FeatureFlagService
import spock.lang.Shared
import spock.lang.Unroll
import util.Env

import java.nio.file.Files
import java.nio.file.Paths

@Category(BAT)
class ClientCertAuthTest extends BaseSpecification {

    @Shared
    private String providerID
    @Shared
    private String certToken

    def setupSpec() {
        Assume.assumeTrue(
            FeatureFlagService.isFeatureFlagEnabled(Constants.CLIENT_CA_AUTH_FEATURE_FLAG)
        )

        String caPath = Env.mustGetClientCAPath()
        byte[] encoded = Files.readAllBytes(Paths.get(caPath))
        def cert = new String(encoded)
        providerID = AuthProviderService.createAuthProvider("Test Client CA Auth", "userpki", ["keys" : cert])
        println "Client cert auth provider ID is ${providerID}"
        certToken = AuthProviderService.getAuthProviderLoginToken(providerID)
        println "Certificate token is ${certToken}"
    }

    def cleanupSpec() {
        if (providerID) {
            AuthProviderService.deleteAuthProvider(providerID)
        }
    }

    private static getAuthProviderType() {
        try {
            return AuthService.getAuthStatus().authProvider.type
        } catch (StatusRuntimeException ex) {
            println "Error getting auth status: ${ex.toString()}"
            return "error"
        }
    }

    @Unroll
    def "Test authentication result with client cert: #useClientCert and auth header #authHeader"() {
        when:
        "Set up channel"
        BaseService.setUseClientCert(useClientCert)
        switch (authHeader) {
            case "basic":
                BaseService.useBasicAuth()
                break
            case "certtoken":
                BaseService.useApiToken(certToken)
                break
            case "none":
                BaseService.useNoAuthorizationHeader()
                break
        }

        then:
        "Verify authorized user has correct provider type"
        assert getAuthProviderType() == expectedProviderType

        where:
        "Data inputs"

        useClientCert | authHeader  | expectedProviderType
        false         | "none"      | "error"
        false         | "basic"     | "basic"
        true          | "basic"     | "basic"
        true          | "none"      | "userpki"
        true          | "certtoken" | "userpki"
        false         | "certtoken" | "error"
    }
}
