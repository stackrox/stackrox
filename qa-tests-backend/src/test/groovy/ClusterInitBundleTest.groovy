import static java.util.UUID.randomUUID

import io.stackrox.proto.api.v1.ApiTokenService

import services.BaseService
import services.ClusterInitBundleService
import util.Env

import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Tag

@Tag("BAT")
@Tag("PZ")
@IgnoreIf({ Env.IS_BYODB })
class ClusterInitBundleTest extends BaseSpecification {

    @Shared
    private ApiTokenService.GenerateTokenResponse adminToken

    def setupSpec() {
        adminToken = services.ApiTokenService.generateToken(randomUUID().toString(), "Admin")
    }

    def cleanupSpec() {
        if (adminToken != null) {
            services.ApiTokenService.revokeToken(adminToken.metadata.id)
        }
    }

    def "Test that cluster init bundle can be revoked when it has no impacted clusters"() {
        BaseService.useApiToken(adminToken.token)

        given:
        "init bundle with no impacted cluster"
        def bundle = ClusterInitBundleService.generateInintBundle("qa-test").getMeta()
        when:
        "revoke it"
        def response = ClusterInitBundleService.revokeInitBundle(bundle.id)

        then:
        "no errors"
        assert response.initBundleRevocationErrorsList.empty
        and:
        "id is revoked"
        assert response.initBundleRevokedIdsList == [bundle.id]
        assert !ClusterInitBundleService.initBundles.find { it.id == bundle.id }
    }
}
