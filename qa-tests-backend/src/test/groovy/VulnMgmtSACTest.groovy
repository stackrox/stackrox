import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.RoleOuterClass

import services.ApiTokenService
import services.BaseService
import services.GraphQLService
import services.ImageService
import services.RoleService

import spock.lang.Retry
import spock.lang.Tag
import spock.lang.Unroll

@Retry(count = 3)
@Tag("Begin")

class VulnMgmtSACTest extends BaseSpecification {
    static final private String NONE = "None"

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
    fragment componentFields on ImageComponent {
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
    fragment componentFields on NodeComponent {
        name
        version
    }
    """

    private static final IMAGES_TO_RESCAN = [
        "quay.io/rhacs-eng/qa:centos7-base",
        TEST_IMAGE,
    ]

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
        // Purposefully add an image (centos7-base) that is not running to check the case
        // where an image is orphaned. The image is actually part of the re-scanned image set.
        // Re-scan the images used in previous test cases to ensure pruning did not leave orphan CVEs.
        for ( imageToScan in IMAGES_TO_RESCAN ) {
            ImageService.scanImage(imageToScan)
            log.debug "Scanned Image ${imageToScan}"
        }

        // Create roles and api tokens for rbac based auth
        createReadRole(NODE_ROLE, ["Node", "CVE"])
        createReadRole(IMAGE_ROLE, ["Image", "CVE"])
        createReadRole(NODE_IMAGE_ROLE, ["Node", "Image", "CVE"])
    }

    def cleanupSpec() {
        BaseService.useBasicAuth()
        RoleService.deleteRole(NODE_ROLE)
        RoleService.deleteRole(IMAGE_ROLE)
        RoleService.deleteRole(NODE_IMAGE_ROLE)
    }

    // GraphQL does not provide ordering guarantees, so to compare the results
    // of two GraphQL queries we extract just the CVE names as a Set.
    Set<String> extractCVEs(queryCallResult) {
        return queryCallResult.results*.cve.toSet() as Set<String>
    }

    static String getToken(String tokenName, String role = NONE) {
        GenerateTokenResponse token = ApiTokenService.generateToken(tokenName, role)
        return token.token
    }

    private static List<ImageOuterClass.ListImage> getImagesWithCVE(String queryText) {
        def imageQuery = RawQuery.newBuilder().setQuery("CVE:${queryText}").build()
        def tstImages = ImageService.getImages(imageQuery)
        return tstImages
    }

    def getImageCVEQuery() {
        return isPostgresRun() ? GET_IMAGE_CVES_QUERY : GET_CVES_QUERY
    }

    def getNodeCVEQuery() {
        return isPostgresRun() ? GET_NODE_CVES_QUERY : GET_CVES_QUERY
    }

    def getImageComponentQuery() {
        return isPostgresRun() ? GET_IMAGE_COMPONENTS_QUERY : GET_COMPONENTS_QUERY
    }

    def getNodeComponentQuery() {
        return isPostgresRun() ? GET_NODE_COMPONENTS_QUERY : GET_COMPONENTS_QUERY
    }

    @Unroll
    def "Verify role based scoping on vuln mgmt: node-role Node:*"() {
        when:
        "Get Node CVEs and components"
        BaseService.useBasicAuth()
        def baseQuery = "Node:*"
        def cveQuery = getNodeCVEQuery()
        def componentQuery = getNodeComponentQuery()
        def gqlService = new GraphQLService()
        def baseVulnCallResult = gqlService.Call(cveQuery, [query: baseQuery])
        assert baseVulnCallResult.hasNoErrors()
        def baseComponentCallResult = gqlService.Call(componentQuery, [query: baseQuery])
        assert baseComponentCallResult.hasNoErrors()

        and:
        gqlService = new GraphQLService(getToken(NODE_ROLE, NODE_ROLE))
        def vulnCallResult = gqlService.Call(cveQuery, [query: ""])
        assert vulnCallResult.hasNoErrors()
        def componentCallResult = gqlService.Call(componentQuery, [query: ""])
        assert componentCallResult.hasNoErrors()
        def baseVulns = extractCVEs(baseVulnCallResult.value)
        def vulns = extractCVEs(vulnCallResult.value)
        if ( baseVulns != vulns ) {
            log.error("Item found in baseVulnCallResult but not in vulnCallResults: " + (baseVulns-vulns))
            log.error("Item found in vulnCallResults but not in baseVulnCallResult: " + (vulns-baseVulns))
        }
        def baseComponentVulns = extractCVEs(baseComponentCallResult.value)
        def componentVulns = extractCVEs(componentCallResult.value)

        then:
        baseVulnCallResult.code == vulnCallResult.code
        baseVulns == vulns

        baseComponentCallResult.code == componentCallResult.code
        baseComponentVulns == componentVulns

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
    }

    @Unroll
    def "Verify role based scoping on vuln mgmt: image-role Image:*"() {
        when:
        "Get Image CVEs and components"
        BaseService.useBasicAuth()
        def gqlService = new GraphQLService()
        def baseQuery = "Image:*"
        def cveQuery = getImageCVEQuery()
        def componentQuery = getImageComponentQuery()
        def baseVulnCallResult = gqlService.Call(cveQuery, [query: baseQuery])
        assert baseVulnCallResult.hasNoErrors()
        def baseComponentCallResult = gqlService.Call(componentQuery, [query: baseQuery])
        assert baseComponentCallResult.hasNoErrors()

        and:
        gqlService = new GraphQLService(getToken(IMAGE_ROLE, IMAGE_ROLE))
        def vulnCallResult = gqlService.Call(cveQuery, [query: ""])
        assert vulnCallResult.hasNoErrors()
        def componentCallResult = gqlService.Call(componentQuery, [query: ""])
        assert componentCallResult.hasNoErrors()
        def baseVulns = extractCVEs(baseVulnCallResult.value)
        def vulns = extractCVEs(vulnCallResult.value)
        def baseComponentVulns = extractCVEs(baseComponentCallResult.value)
        def componentVulns = extractCVEs(componentCallResult.value)
        if ( baseVulns != vulns ) {
            (baseVulns-vulns).each {
                String cve ->
                log.error("Item found in baseVulnCallResult but not in vulnCallResults: " + cve)
                for ( img in getImagesWithCVE(cve) ) {
                    log.error("Vulnerability ${cve} is found in image {${img}}")
                }
            }
            (vulns-baseVulns).each {
                String cve ->
                log.error("Item found in vulnCallResults but not in baseVulnCallResult: " + cve)
                for ( img in getImagesWithCVE(cve) ) {
                    log.error("Vulnerability ${cve} is found in image {${img}}")
                }
            }
        }

        then:
        baseVulnCallResult.code == vulnCallResult.code
        baseVulns == vulns

        baseComponentCallResult.code == componentCallResult.code
        baseComponentVulns == componentVulns

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
    }

    @Unroll
    def "Verify role based scoping on vuln mgmt: #roleName #baseQuery"() {
        when:
        "Get Node CVEs and components"
        BaseService.useBasicAuth()
        def gqlService = new GraphQLService()
        def imageCveQuery = getImageCVEQuery()
        def imageComponentQuery = getImageComponentQuery()
        def nodeCveQuery = getNodeCVEQuery()
        def nodeComponentQuery = getNodeComponentQuery()
        def imageBaseQuery = isPostgresRun() ? imageQuery : baseQuery
        def nodeBaseQuery = isPostgresRun() ? nodeQuery : baseQuery
        def baseImageVulnCallResult = gqlService.Call(imageCveQuery, [query: imageBaseQuery])
        assert baseImageVulnCallResult.hasNoErrors()
        def baseImageComponentCallResult = gqlService.Call(imageComponentQuery, [query: imageBaseQuery])
        assert baseImageComponentCallResult.hasNoErrors()
        def baseNodeVulnCallResult = gqlService.Call(nodeCveQuery, [query: nodeBaseQuery])
        assert baseNodeVulnCallResult.hasNoErrors()
        def baseNodeComponentCallResult = gqlService.Call(nodeComponentQuery, [query: nodeBaseQuery])
        assert baseNodeComponentCallResult.hasNoErrors()

        and:
        gqlService = new GraphQLService(getToken(roleName, roleName))
        def imageVulnCallResult = gqlService.Call(imageCveQuery, [query: ""])
        assert imageVulnCallResult.hasNoErrors()
        def imageComponentCallResult = gqlService.Call(imageComponentQuery, [query: ""])
        assert imageComponentCallResult.hasNoErrors()
        def nodeVulnCallResult = gqlService.Call(nodeCveQuery, [query: ""])
        assert nodeVulnCallResult.hasNoErrors()
        def nodeComponentCallResult = gqlService.Call(nodeComponentQuery, [query: ""])
        assert nodeComponentCallResult.hasNoErrors()

        then:
        baseImageVulnCallResult.code == imageVulnCallResult.code
        extractCVEs(baseImageVulnCallResult.value) == extractCVEs(imageVulnCallResult.value)

        baseImageComponentCallResult.code == imageComponentCallResult.code
        extractCVEs(baseImageComponentCallResult.value) == extractCVEs(imageComponentCallResult.value)

        baseNodeVulnCallResult.code == nodeVulnCallResult.code
        extractCVEs(baseNodeVulnCallResult.value) == extractCVEs(nodeVulnCallResult.value)

        baseNodeComponentCallResult.code == nodeComponentCallResult.code
        extractCVEs(baseNodeComponentCallResult.value) == extractCVEs(nodeComponentCallResult.value)

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()

        where:
        "Data inputs are: "
        roleName        | baseQuery     | imageQuery         | nodeQuery
        NODE_IMAGE_ROLE | "Component:*" | "ImageComponent:*" | "NodeComponent:*"
    }

    @Unroll
    def "Verify permissions on vuln mgmt: role with no CVE permissions is rejected"() {
        when:
        "Get CVEs via GraphQL"
        def gqlService = new GraphQLService(getToken("none-role", NONE))
        def vulnCallResult = gqlService.Call(GET_CVES_QUERY, [query: ""])

        then:
        assert !vulnCallResult.hasNoErrors()
    }
}
