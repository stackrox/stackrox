import java.nio.file.Files
import java.nio.file.Paths

import io.grpc.StatusRuntimeException

import io.stackrox.proto.api.v1.GroupServiceOuterClass

import services.AuthProviderService
import services.AuthService
import services.BaseService
import services.GroupService
import util.Env

import spock.lang.Shared
import spock.lang.Stepwise
import spock.lang.Tag
import spock.lang.Unroll

@Tag("BAT")
@Stepwise
class ClientCertAuthTest extends BaseSpecification {

    @Shared
    private String[] providerIDs = []
    @Shared
    private String[] certTokens = []

    def setupSpec() {
        BaseService.useBasicAuth()

        String caPath = Env.mustGetClientCAPath()
        byte[] encoded = Files.readAllBytes(Paths.get(caPath))
        def cert = new String(encoded)

        providerIDs = new String[2]
        certTokens = new String[2]
        for (int i = 0; i < 2; i++) {
            providerIDs[i] = AuthProviderService.createAuthProvider(
                    "Test Client CA Auth ${i}", "userpki", ["keys": cert])
            log.info "Client cert auth provider ID is ${providerIDs[i]}"
            GroupService.addDefaultMapping(providerIDs[i], "Continuous Integration")
            certTokens[i] = AuthProviderService.getAuthProviderLoginToken(providerIDs[i])
            log.info "Certificate token is ${certTokens[i]}"
        }
    }

    def cleanupSpec() {
        BaseService.useBasicAuth()
        for (String providerID : providerIDs) {
            if (providerID) {
                GroupService.removeAllMappingsForProvider(providerID)
                AuthProviderService.deleteAuthProvider(providerID)
            }
        }
    }

    private getAuthProviderType() {
        try {
            return AuthService.getAuthStatus().authProvider.type
        } catch (StatusRuntimeException ex) {
            log.error("Error getting auth status", ex)
            return "error"
        }
    }

    private getAuthProviderID() {
        try {
            return AuthService.getAuthStatus().authProvider.id
        } catch (StatusRuntimeException ex) {
            log.error("Error getting auth status", ex)
            return ""
        }
    }

    private static GroupServiceOuterClass.GetGroupsResponse getAuthProviderGroups(String providerID) {
        def request = GroupServiceOuterClass.GetGroupsRequest.newBuilder()
                .setAuthProviderId(providerID)
                .build()
        return GroupService.getGroups(request)
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
            case "certtoken0":
                BaseService.useApiToken(certTokens[0])
                break
            case "certtoken1":
                BaseService.useApiToken(certTokens[1])
                break
            case "none":
                BaseService.useNoAuthorizationHeader()
                break
        }

        then:
        "Verify authorized user has correct provider type"
        assert getAuthProviderType() == expectedProviderType

        and:
        "Verify auth provider has acceptable ID"
        def providerID = getAuthProviderID()

        if (acceptableProviderIndices) {
            def acceptableIDs = acceptableProviderIndices.collect { providerIDs[it] }
            assert acceptableIDs.contains(providerID)
        } else {
            def unacceptableIDs = providerIDs.toList()
            assert !unacceptableIDs.contains(providerID)
        }

        where:
        "Data inputs"

        useClientCert | authHeader   | expectedProviderType | acceptableProviderIndices
        false         | "none"       | "error"              | []
        false         | "basic"      | "basic"              | []
        true          | "basic"      | "basic"              | []
        true          | "none"       | "userpki"            | [0, 1]
        true          | "certtoken0" | "userpki"            | [0]
        true          | "certtoken1" | "userpki"            | [1]
        false         | "certtoken0" | "error"              | []
        false         | "certtoken1" | "error"              | []
    }

    def "Delete Auth provider"() {
        when:
        "Delete auth provider"
        for (String providerID : providerIDs) {
            if (providerID) {
                AuthProviderService.deleteAuthProvider(providerID)
                def resp = getAuthProviderGroups(providerID)
                assert resp.getGroupsCount() == 0
            }
        }
        providerIDs = []

        then:
        "Deletion should have taken place"
        assert !providerIDs
    }

    @Unroll
    def "Test authentication fails with client cert: #useClientCert and auth header #authHeader after deletion"() {
        when:
        "Set up channel"
        BaseService.setUseClientCert(useClientCert)
        switch (authHeader) {
            case "certtoken0":
                BaseService.useApiToken(certTokens[0])
                break
            case "certtoken1":
                BaseService.useApiToken(certTokens[1])
                break
            case "none":
                BaseService.useNoAuthorizationHeader()
                break
        }

        then:
        "Verify that authorization fails"
        assert getAuthProviderType() == "error"

        where:
        "Data inputs"

        useClientCert | authHeader
        true          | "none"
        true          | "certtoken0"
        true          | "certtoken1"
        false         | "certtoken0"
        false         | "certtoken1"
    }
}
