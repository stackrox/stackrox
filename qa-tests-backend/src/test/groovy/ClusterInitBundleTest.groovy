import static java.util.UUID.randomUUID

import io.stackrox.proto.api.v1.ApiTokenService

import services.BaseService
import services.ClusterInitBundleService
import services.ClusterService

import org.junit.Assume
import spock.lang.Shared
import spock.lang.Tag

@Tag("BAT")
@Tag("PZ")
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

    def "Test that revoke cluster init bundle requires impacted clusters"() {
        BaseService.useApiToken(adminToken.token)

        def cluster = ClusterService.getCluster()
        Assume.assumeTrue(cluster.hasHelmConfig())

        when:
        "making a request for the cluster init bundle"
        def bundles = ClusterInitBundleService.getInitBundles()

        then:
        "there is a bundle for current cluster"
        def bundle = bundles.find { b -> b.impactedClustersList.find { c -> c.id == cluster.id } }
        assert bundle

        when:
        "try to delete used init bundle not confirming impacted clusters"
        def response = ClusterInitBundleService.revokeInitBundle(bundle.id)

        then:
        "no bundle is revoked"
        assert response.initBundleRevokedIdsCount == 0
        and:
        "impacted cluster is listed"
        assert response.initBundleRevocationErrorsList.first().impactedClustersList*.id.contains(cluster.id)
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
