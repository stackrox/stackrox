import static org.junit.Assume.assumeFalse
import groups.BAT
import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.RoleOuterClass
import org.junit.experimental.categories.Category
import services.ApiTokenService
import services.BaseService
import services.GraphQLService
import services.ImageIntegrationService
import services.ImageService
import services.RoleService

import spock.lang.Retry
import spock.lang.Unroll
import util.Env

@Category(BAT)
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
        "busybox",
        "busybox:latest",
        "docker.io/istio/proxyv2@sha256:134e99aa9597fdc17305592d13add95e2032609d23b4c508bd5ebd32ed2df47d",
        "docker.io/jenkins/jenkins:2.220-alpine",
        "docker.io/jenkins/jenkins:lts",
        "docker.io/docker/kube-compose-controller:v0.4.23",
        "docker.io/library/alpine:3.10.0",
        "docker.io/library/busybox:1.32.0",
        "docker.io/library/busybox:latest",
        "docker.io/library/centos:centos8.2.2004",
        "docker.io/library/fedora:33",
        "docker.io/library/nginx@sha256:204a9a8e65061b10b92ad361dd6f406248404fe60efd5d6a8f2595f18bb37aad",
        "docker.io/library/nginx:1.10",
        "docker.io/library/nginx:1.19",
        "docker.io/library/nginx:latest",
        "docker.io/library/ubuntu:14.04",
        "docker.io/nginx@sha256:63aa22a3a677b20b74f4c977a418576934026d8562c04f6a635f0e71e0686b6d",
        "gcr.io/distroless/base@sha256:bc217643f9c04fc8131878d6440dd88cf4444385d45bb25995c8051c29687766",
        "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init"+
            "@sha256:79f768d28ff9af9fcbf186f9fc1b8e9f88835dfb07be91610a1f17cf862db89e",
        "gke.gcr.io/heapster:v1.7.2",
        "k8s.gcr.io/ip-masq-agent-amd64:v2.4.1",
        "library/nginx:1.10",
        "mcr.microsoft.com/dotnet/core/runtime:2.1-alpine",
        "mysql@sha256:de2913a0ec53d98ced6f6bd607f487b7ad8fe8d2a86e2128308ebf4be2f92667",
        "mysql@sha256:f7985e36c668bb862a0e506f4ef9acdd1254cdf690469816f99633898895f7fa",
        "nginx",
        "nginx@sha256:204a9a8e65061b10b92ad361dd6f406248404fe60efd5d6a8f2595f18bb37aad",
        "nginx:1.17@sha256:86ae264c3f4acb99b2dee4d0098c40cb8c46dcf9e1148f05d3a51c4df6758c12",
        "nginx:latest",
        "nginx:latest@sha256:86ae264c3f4acb99b2dee4d0098c40cb8c46dcf9e1148f05d3a51c4df6758c12",
        "perl:5.32.1",
        "quay.io/rhacs-eng/qa:apache-dns",
        "quay.io/rhacs-eng/qa:apache-dns",
        "quay.io/rhacs-eng/qa:apache-server",
        "quay.io/rhacs-eng/qa:busybox",
        "quay.io/rhacs-eng/qa:busybox-1-25",
        "quay.io/rhacs-eng/qa:busybox-1-26",
        "quay.io/rhacs-eng/qa:busybox-1-27",
        "quay.io/rhacs-eng/qa:busybox-1-28",
        "quay.io/rhacs-eng/qa:busybox-1-29",
        "quay.io/rhacs-eng/qa:busybox-1-30",
        "quay.io/rhacs-eng/qa:centos-fc2476ccae2a5186313f2d1dadb4a969d6d2d4c6b23fa98b6c7b0a1faad67685",
        "quay.io/rhacs-eng/qa:centos7-base",
        "quay.io/rhacs-eng/qa:centos7-base-echo",
        "quay.io/rhacs-eng/qa:docker-io-nginx-1-10",
        "quay.io/rhacs-eng/qa:elasticsearch-cdeb134689bb0318a773e03741f4414b3d1d0ee443b827d5954f957775db57eb",
        "quay.io/rhacs-eng/qa:enforcement",
        "quay.io/rhacs-eng/qa:fedora-6fb84ba634fe68572a2ac99741062695db24b921d0aa72e61ee669902f88c187",
        "quay.io/rhacs-eng/qa:mongo-dec7f10108a87ff660a0d56cb71b0c5ae1f33cba796a33c88b50280fc0707116",
        "quay.io/rhacs-eng/qa:nginx",
        "quay.io/rhacs-eng/qa:nginx-1-7-9",
        "quay.io/rhacs-eng/qa:nginx-1-9",
        "quay.io/rhacs-eng/qa:nginx-1-12-1",
        "quay.io/rhacs-eng/qa:nginx-1.14-alpine",
        "quay.io/rhacs-eng/qa:nginx-1-15-4-alpine",
        "quay.io/rhacs-eng/qa:nginx-1.15.4-alpine",
        "quay.io/rhacs-eng/qa:nginx-1.19-alpine",
        "quay.io/rhacs-eng/qa:nginx-204a9a8e65061b10b92ad361dd6f406248404fe60efd5d6a8f2595f18bb37aad",
        "quay.io/rhacs-eng/qa:oci-manifest",
        "quay.io/rhacs-eng/qa:redis-96be1b5b6e4fe74dfe65b2b52a0fee254c443184b34fe448f3b3498a512db99e",
        "quay.io/rhacs-eng/qa:registry-image-0-3",
        "quay.io/rhacs-eng/qa:ROX4751",
        "quay.io/rhacs-eng/qa:ROX4979",
        "quay.io/rhacs-eng/qa:socat",
        "quay.io/rhacs-eng/qa:ssl-terminator",
        "quay.io/rhacs-eng/qa:struts-app",
        "quay.io/rhacs-eng/qa:struts-app",
        "richxsl/rhel7@sha256:8f3aae325d2074d2dc328cb532d6e7aeb0c588e15ddf847347038fe0566364d6",
        "stackroxci.azurecr.io/stackroxci/registry-image:0.3",
        TEST_IMAGE,
        "gcr.io/distroless/base@sha256:bc217643f9c04fc8131878d6440dd88cf4444385d45bb25995c8051c29687766",
        "us.gcr.io/stackrox-ci/nginx:1.9.1",
        "us.gcr.io/stackrox-ci/nginx:1.10.1",
        "us.gcr.io/stackrox-ci/nginx:1.10.1@sha256:b53e7ca2f567bdb7f23dad7d183a3466532d32f7ddf82847783fad14f425e5d3",
        "us.gcr.io/stackrox-ci/nginx:1.11",
        "us.gcr.io/stackrox-ci/nginx:1.11.1",
        "us.gcr.io/stackrox-ci/nginx:1.12",
        "us.gcr.io/stackrox-ci/qa/fail-compliance/ssh:0.1",
        "us.gcr.io/stackrox-ci/qa/registry-image:0.2",
        "us.gcr.io/stackrox-ci/qa/registry-image:0.3",
        "us.gcr.io/stackrox-ci/qa/trigger-policy-violations/alpine:0.6",
        "us.gcr.io/stackrox-ci/qa/trigger-policy-violations/more:0.3",
        "us.gcr.io/stackrox-ci/qa/trigger-policy-violations/most:0.19",
        "us-west1-docker.pkg.dev/stackrox-ci/artifact-registry-test1/nginx:1.17",
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
        assumeFalse("This test is skipped in this environment", skipThisTest())

        // Purposefully add an image (centos7-base) that is not running to check the case
        // where an image is orphaned. The image is actually part of the re-scanned image set.
        ImageIntegrationService.addStackroxScannerIntegration()
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

    @Retry(count = 0)
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
        def baseSortedVulns = extractCVEsAndSort(baseVulnCallResult.value)
        def sortedVulns = extractCVEsAndSort(vulnCallResult.value)
        if ( baseSortedVulns != sortedVulns ) {
            log.error("Item found in baseVulnCallResult but not in vulnCallResults: " + (baseSortedVulns-sortedVulns))
            log.error("Item found in vulnCallResults but not in baseVulnCallResult: " + (sortedVulns-baseSortedVulns))
        }

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
        def baseSortedVulns = extractCVEsAndSort(baseVulnCallResult.value)
        def sortedVulns = extractCVEsAndSort(vulnCallResult.value)
        if ( baseSortedVulns != sortedVulns ) {
            (baseSortedVulns-sortedVulns).each {
                item ->
                log.error("Item found in baseVulnCallResult but not in vulnCallResults: " + item.cve)
                for ( img in getImagesWithCVE(item.cve) ) {
                    log.error("Vulnerability ${item.cve} is found in image {${img}}")
                }
            }
            (sortedVulns-baseSortedVulns).each {
                item ->
                log.error("Item found in vulnCallResults but not in baseVulnCallResult: " + item.cve)
                for ( img in getImagesWithCVE(item.cve) ) {
                    log.error("Vulnerability ${item.cve} is found in image {${img}}")
                }
            }
        }

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
        extractCVEsAndSort(baseImageVulnCallResult.value) == extractCVEsAndSort(imageVulnCallResult.value)

        baseImageComponentCallResult.code == imageComponentCallResult.code
        extractCVEsAndSort(baseImageComponentCallResult.value) == extractCVEsAndSort(imageComponentCallResult.value)

        baseNodeVulnCallResult.code == nodeVulnCallResult.code
        extractCVEsAndSort(baseNodeVulnCallResult.value) == extractCVEsAndSort(nodeVulnCallResult.value)

        baseNodeComponentCallResult.code == nodeComponentCallResult.code
        extractCVEsAndSort(baseNodeComponentCallResult.value) == extractCVEsAndSort(nodeComponentCallResult.value)

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()

        where:
        "Data inputs are: "
        roleName        | baseQuery     | imageQuery         | nodeQuery
        NODE_IMAGE_ROLE | "Component:*" | "ImageComponent:*" | "NodeComponent:*"
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
