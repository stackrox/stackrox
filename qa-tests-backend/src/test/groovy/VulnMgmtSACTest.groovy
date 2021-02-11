import io.stackrox.proto.storage.RoleOuterClass
import org.junit.Assume
import services.FeatureFlagService
import services.GraphQLService

import groups.BAT
import org.junit.experimental.categories.Category

import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import services.ApiTokenService
import services.BaseService
import services.ImageIntegrationService
import services.ImageService
import services.RoleService
import services.SACService
import spock.lang.Retry
import spock.lang.Unroll

@Category(BAT)
class VulnMgmtSACTest extends BaseSpecification {
    static final private String NONE = "None"
    static final private String ALLACCESSTOKEN = "allAccessToken"
    static final private String NOACCESSTOKEN = "noAccess"
    static final private String CENTOS_IMAGE = "stackrox/qa:centos7-base"

    static final private String NODE_ROLE = "node-role"
    static final private String IMAGE_ROLE = "image-role"
    static final private String NODE_IMAGE_ROLE = "node-image-role"

    private static final GET_CVES_QUERY = """
    query getCves(\$query: String, \$pagination: Pagination)
    {
        results: vulnerabilities(query: \$query, pagination: \$pagination) {
            ...cveFields
            __typename
        }
        count: vulnerabilityCount(query: \$query)
    }

    fragment cveFields on EmbeddedVulnerability {
        cve
    }
    """

    private static final GET_COMPONENTS_QUERY = """
    query getComponents(\$query: String, \$pagination: Pagination)
    {
        results: components(query: \$query, pagination: \$pagination) {
            ...componentFields
            __typename
        }
        count: componentCount(query: \$query)
    }

    fragment componentFields on EmbeddedImageScanComponent {
        name
        version
    }
    """

    def createReadRole(String name, List<String> resources) {
        def testRole = RoleOuterClass.Role.newBuilder()
                .setName(name)
        for (String resource: resources) {
            testRole.putResourceToAccess(resource, RoleOuterClass.Access.READ_ACCESS)
        }
        RoleService.createRole(testRole.build())
        assert RoleService.getRole(testRole.name)
        println "Created Role:\n${testRole}"
    }

    def setupSpec() {
        // Purposefully add an image that is not running to check the case
        // where an image is orphaned
        ImageIntegrationService.addStackroxScannerIntegration()
        ImageService.scanImage(CENTOS_IMAGE)

        // Create roles and api tokens for rbac based auth
        createReadRole(NODE_ROLE, ["Node"])
        createReadRole(IMAGE_ROLE, ["Image"])
        createReadRole(NODE_IMAGE_ROLE, ["Node", "Image"])
    }

    def cleanupSpec() {
        BaseService.useBasicAuth()
        ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
    }

    static String getToken(String tokenName, String role = NONE) {
        GenerateTokenResponse token = ApiTokenService.generateToken(tokenName, role)
        return token.token
    }

    @Retry(count = 0)
    @Unroll
    def "Verify role based scoping on vuln mgmt: #roleName #baseQuery"() {
        when:
        "Get Node CVEs and components"
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_HOST_SCANNING"))
        BaseService.useBasicAuth()
        disableAuthzPlugin()

        def gqlService = new GraphQLService()
        def baseVulnCallResult = gqlService.Call(GET_CVES_QUERY, [query: baseQuery])
        assert baseVulnCallResult.hasNoErrors()
        def baseComponentCallResult = gqlService.Call(GET_COMPONENTS_QUERY, [query: baseQuery])
        assert baseComponentCallResult.hasNoErrors()

        and:
        gqlService = new GraphQLService(getToken(roleName, roleName))
        def vulnCallResult = gqlService.Call(GET_CVES_QUERY, [query: ""])
        assert vulnCallResult.hasNoErrors()
        def componentCallResult = gqlService.Call(GET_COMPONENTS_QUERY, [query: ""])
        assert componentCallResult.hasNoErrors()

        then:
        baseVulnCallResult.code == vulnCallResult.code
        baseVulnCallResult.value == vulnCallResult.value

        baseComponentCallResult.code == componentCallResult.code
        baseComponentCallResult.value == componentCallResult.value

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()

        where:
        "Data inputs are: "
        roleName        | baseQuery
        NONE            | "Node:thisdoesntexist"
        NODE_ROLE       | "Node:*"
        IMAGE_ROLE      | "Image:*"
        NODE_IMAGE_ROLE | "Component:*"
    }

    @Retry(count = 0)
    @Unroll
    def "Verify SAC on vuln mgmt shared objects: #tokenName #baseQuery"() {
        when:
        "Get Node CVEs and components"
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_HOST_SCANNING"))
        BaseService.useBasicAuth()
        SACService.addAuthPlugin()

        def gqlService = new GraphQLService()
        def baseVulnCallResult = gqlService.Call(GET_CVES_QUERY, [query: baseQuery])
        assert baseVulnCallResult.hasNoErrors()
        def baseComponentCallResult = gqlService.Call(GET_COMPONENTS_QUERY, [query: baseQuery])
        assert baseComponentCallResult.hasNoErrors()

        and:
        gqlService = new GraphQLService(getToken(tokenName))
        def vulnCallResult = gqlService.Call(GET_CVES_QUERY, [query: ""])
        assert vulnCallResult.hasNoErrors()
        def componentCallResult = gqlService.Call(GET_COMPONENTS_QUERY, [query: ""])
        assert componentCallResult.hasNoErrors()

        then:
        baseVulnCallResult.code == vulnCallResult.code
        baseVulnCallResult.value == vulnCallResult.value

        baseComponentCallResult.code == componentCallResult.code
        baseComponentCallResult.value == componentCallResult.value

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()

        where:
        "Data inputs are: "
        tokenName                    | baseQuery
        NOACCESSTOKEN                | "Node:thisdoesntexist"
        ALLACCESSTOKEN               | ""
        "nodes-only"                 | "Node:*"
        "images-only"                | "Image:*"
        "images-and-nodes-only"      | "Component:*"
    }
}
