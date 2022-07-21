import static org.junit.Assume.assumeFalse
import groups.BAT
import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import io.stackrox.proto.storage.RoleOuterClass
import org.junit.experimental.categories.Category
import services.ApiTokenService
import services.BaseService
import services.GraphQLService
import services.ImageIntegrationService
import services.ImageService
import services.RoleService

import spock.lang.IgnoreIf
import spock.lang.Retry
import spock.lang.Unroll
import util.Env

@Category(BAT)
class VulnMgmtSACTest extends BaseSpecification {
    static final private String NONE = "None"
    static final private String CENTOS_IMAGE = "quay.io/rhacs-eng/qa:centos7-base"

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

    private static final GET_IMAGE_CVES_QUERY = """
    query getCves(\$query: String, \$pagination: Pagination)
    {
        results: imageVulnerabilities(query: \$query, pagination: \$pagination) {
            ...cveFields
            __typename
        }
        count: imageVulnerabilityCount(query: \$query)
    }

    fragment cveFields on ImageVulnerability {
        cve
    }
    """

    private static final GET_NODE_CVES_QUERY = """
    query getCves(\$query: String, \$pagination: Pagination)
    {
        results: nodeVulnerabilities(query: \$query, pagination: \$pagination) {
            ...cveFields
            __typename
        }
        count: nodeVulnerabilityCount(query: \$query)
    }

    fragment cveFields on NodeVulnerability {
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

    private static final GET_IMAGE_COMPONENTS_QUERY = """
    query getComponents(\$query: String, \$pagination: Pagination)
    {
        results: imageComponents(query: \$query, pagination: \$pagination) {
            ...componentFields
            __typename
        }
        count: imageComponentCount(query: \$query)
    }

    fragment componentFields on EmbeddedImageScanComponent {
        name
        version
    }
    """

    private static final GET_NODE_COMPONENTS_QUERY = """
    query getComponents(\$query: String, \$pagination: Pagination)
    {
        results: nodeComponents(query: \$query, pagination: \$pagination) {
            ...componentFields
            __typename
        }
        count: nodeComponentCount(query: \$query)
    }

    fragment componentFields on EmbeddedNodeScanComponent {
        name
        version
    }
    """

    def createReadRole(String name, List<String> resources) {
        Map<String, RoleOuterClass.Access> resourceToAccess = resources.collectEntries {
            [it, RoleOuterClass.Access.READ_ACCESS]
        }
        def testRole = RoleService.createRoleWithScopeAndPermissionSet(name,
            UNRESTRICTED_SCOPE_ID, resourceToAccess)
        assert RoleService.getRole(testRole.name)
        log.info "Created Role:\n${testRole}"
    }

    def setupSpec() {
        assumeFalse("This test is skipped in this environment", skipThisTest())

        // Purposefully add an image that is not running to check the case
        // where an image is orphaned
        ImageIntegrationService.addStackroxScannerIntegration()
        ImageService.scanImage(CENTOS_IMAGE)

        // Create roles and api tokens for rbac based auth
        createReadRole(NODE_ROLE, ["Node", "CVE"])
        createReadRole(IMAGE_ROLE, ["Image", "CVE"])
        createReadRole(NODE_IMAGE_ROLE, ["Node", "Image", "CVE"])
    }

    def cleanupSpec() {
        assumeFalse("This test is skipped in this environment", skipThisTest())

        BaseService.useBasicAuth()
        ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
        RoleService.deleteRole(NODE_ROLE)
        RoleService.deleteRole(IMAGE_ROLE)
        RoleService.deleteRole(NODE_IMAGE_ROLE)
    }

    // GraphQL does not provide ordering guarantees, so to compare the results
    // of two GraphQL queries we extract just the CVE names and sort them.
    def extractCVEsAndSort(queryCallResult) {
        return queryCallResult.results*.cve.sort()
    }

    static String getToken(String tokenName, String role = NONE) {
        GenerateTokenResponse token = ApiTokenService.generateToken(tokenName, role)
        return token.token
    }

    @Retry(count = 0)
    @Unroll
    def "Verify role based scoping on vuln mgmt: node-role Node:*"() {
        when:
        "Get Node CVEs and components"
        BaseService.useBasicAuth()
        def baseQuery = "Node:*"
        def roleName = "node-role"
        def cveQuery = ""
        if (Env.CI_JOBNAME.contains("postgres")) {
            cveQuery = GET_NODE_CVES_QUERY
        } else {
            cveQuery = GET_CVES_QUERY
        }
        def componentQuery = ""
        if (Env.CI_JOBNAME.contains("postgres")) {
            componentQuery = GET_NODE_COMPONENTS_QUERY
        } else {
            componentQuery = GET_COMPONENTS_QUERY
        }
        def gqlService = new GraphQLService()
        def baseVulnCallResult = gqlService.Call(cveQuery, [query: baseQuery])
        assert baseVulnCallResult.hasNoErrors()
        def baseComponentCallResult = gqlService.Call(componentQuery, [query: baseQuery])
        assert baseComponentCallResult.hasNoErrors()

        and:
        gqlService = new GraphQLService(getToken(roleName, roleName))
        def vulnCallResult = gqlService.Call(cveQuery, [query: ""])
        assert vulnCallResult.hasNoErrors()
        def componentCallResult = gqlService.Call(componentQuery, [query: ""])
        assert componentCallResult.hasNoErrors()

        then:
        baseVulnCallResult.code == vulnCallResult.code
        extractCVEsAndSort(baseVulnCallResult.value) == extractCVEsAndSort(vulnCallResult.value)

        baseComponentCallResult.code == componentCallResult.code
        extractCVEsAndSort(baseComponentCallResult.value) == extractCVEsAndSort(componentCallResult.value)

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
    }

    @Retry(count = 0)
    @Unroll
    def "Verify role based scoping on vuln mgmt: image-role Image:*"() {
        when:
        "Get Node CVEs and components"
        BaseService.useBasicAuth()
        def baseQuery = "Image:*"
        def roleName = "image-role"
        def cveQuery = ""
        if (Env.CI_JOBNAME.contains("postgres")) {
            cveQuery = GET_IMAGE_CVES_QUERY
        } else {
            cveQuery = GET_CVES_QUERY
        }
        def componentQuery = ""
        if (Env.CI_JOBNAME.contains("postgres")) {
            componentQuery = GET_IMAGE_COMPONENTS_QUERY
        } else {
            componentQuery = GET_COMPONENTS_QUERY
        }
        def gqlService = new GraphQLService()
        def baseVulnCallResult = gqlService.Call(cveQuery, [query: baseQuery])
        assert baseVulnCallResult.hasNoErrors()
        def baseComponentCallResult = gqlService.Call(componentQuery, [query: baseQuery])
        assert baseComponentCallResult.hasNoErrors()

        and:
        gqlService = new GraphQLService(getToken(roleName, roleName))
        def vulnCallResult = gqlService.Call(cveQuery, [query: ""])
        assert vulnCallResult.hasNoErrors()
        def componentCallResult = gqlService.Call(componentQuery, [query: ""])
        assert componentCallResult.hasNoErrors()

        then:
        baseVulnCallResult.code == vulnCallResult.code
        extractCVEsAndSort(baseVulnCallResult.value) == extractCVEsAndSort(vulnCallResult.value)

        baseComponentCallResult.code == componentCallResult.code
        extractCVEsAndSort(baseComponentCallResult.value) == extractCVEsAndSort(componentCallResult.value)

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
    }

    @Retry(count = 0)
    @Unroll
    @IgnoreIf({ Env.CI_JOBNAME.contains("postgres") })
    def "Verify role based scoping on vuln mgmt: #roleName #baseQuery"() {
        when:
        "Get Node CVEs and components"
        BaseService.useBasicAuth()
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
        extractCVEsAndSort(baseVulnCallResult.value) == extractCVEsAndSort(vulnCallResult.value)

        baseComponentCallResult.code == componentCallResult.code
        extractCVEsAndSort(baseComponentCallResult.value) == extractCVEsAndSort(componentCallResult.value)

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()

        where:
        "Data inputs are: "
        roleName        | baseQuery
        NODE_IMAGE_ROLE | "Component:*"
    }

    @Retry(count = 0)
    @Unroll
    def "Verify permissions on vuln mgmt: role with no CVE permissions is rejected"() {
        when:
        "Get CVEs via GraphQL"
        def gqlService = new GraphQLService(getToken("none-role", NONE))
        def vulnCallResult = gqlService.Call(GET_CVES_QUERY, [query: ""])

        then:
        assert !vulnCallResult.hasNoErrors()
    }

    private static Boolean skipThisTest() {
        // This test consistently fails with RHEL -race (ROX-6584)
        return Env.get("IS_RACE_BUILD", null) == "true" &&
                Env.CI_JOBNAME && Env.CI_JOBNAME.contains("-rhel")
    }
}
