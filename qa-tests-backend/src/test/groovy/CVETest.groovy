import groups.BAT
import objects.Deployment
import org.junit.experimental.categories.Category
import services.GraphQLService
import services.ImageService
import spock.lang.Unroll

class CVETest extends BaseSpecification {
    private static final GET_CVES_QUERY = """
    query getCves(\$query: String, \$scopeQuery:String, \$pagination: Pagination)
    {
        results: vulnerabilities(query: \$query, pagination: \$pagination) {
            ...cveFields
            __typename
        }
        count: vulnerabilityCount(query: \$query)
    }

    fragment cveFields on EmbeddedVulnerability {
        id: cve
        cve
        cvss
        scoreVersion
        impactScore
        summary
        fixedByVersion
        createdAt
        envImpact
        publishedOn
        isFixable(query: \$scopeQuery)
        deploymentCount(query: \$query)
        imageCount(query: \$query)
        componentCount(query: \$query)
        __typename
    }
    """

    static final private String CVE_DEPLOYMENT_NAME = "cve-deployment"

    static final private Deployment CVE_DEPLOYMENT = new Deployment()
            .setName(CVE_DEPLOYMENT_NAME)
            .setImage("us.gcr.io/stackrox-ci/nginx:1.9")
            .addLabel("app", "test")

    static final private NGINX_1_10_2_IMAGE = "us.gcr.io/stackrox-ci/nginx:1.10.2"
    static final private RED_HAT_IMAGE =
            "centos:8@sha256:4ec83eee30dfbaba2e93f59d36cc360660d13f73c71af179eeb9456dd95d1798"

    def setupSpec() {
        ImageService.scanImage("us.gcr.io/stackrox-ci/nginx:1.9")
        ImageService.scanImage(NGINX_1_10_2_IMAGE)
        ImageService.scanImage(RED_HAT_IMAGE)
        orchestrator.createDeployment(CVE_DEPLOYMENT)
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(CVE_DEPLOYMENT)
    }

    @Unroll
    @Category(BAT)
    def "Verify the results of the CVE GraphQL Query with single specific CVE match - #cve"() {
        when:
        "Fetch the CVEs using GraphQL"
        def gqlService = new GraphQLService()
        def resultRet = gqlService.Call(GET_CVES_QUERY,
                [query: "Image:${image}+CVE:${cve}", scopeQuery: ""])
        assert resultRet.getCode() == 200
        println "return code " + resultRet.getCode()

        then:
        "Verify specific CVE data"
        assert resultRet.value.count == 1
        assert resultRet.value.results.size() == 1
        def foundCVE = resultRet.value.results[0]
        assert foundCVE.id == cve
        assert foundCVE.cve == cve
        assert foundCVE.cvss > cvss - 0.1 &&
                foundCVE.cvss < cvss + 0.1
        assert foundCVE.scoreVersion == scoreVersion
        assert foundCVE.impactScore > impactScore - 0.1 &&
                foundCVE.impactScore < impactScore + 0.1
        assert foundCVE.isFixable == isFixable
        assert foundCVE.deploymentCount == deploymentCount
        assert foundCVE.imageCount == imageCount
        assert foundCVE.componentCount == componentCount
        assert foundCVE.summary != ""

        where:
        "data inputs"

        cve              | cvss | scoreVersion | impactScore | publishedOn            |
                isFixable | deploymentCount | imageCount | componentCount | image
        "CVE-2005-2541"  | 10   | "V2"         | 10          | "2005-08-10T04:00:00Z" |
                false     | 0               | 1          | 1              | NGINX_1_10_2_IMAGE
        "CVE-2019-9232"  | 7.5  | "V3"         | 3.6         | "2019-09-27T19:15:00Z" |
                true      | 0               | 1          | 1              | NGINX_1_10_2_IMAGE
// TODO(ROX-5653)
//        "CVE-2020-8177"  | 5.4  | "V3"         | 4.2         | "2020-06-24T00:00:00Z" |
//                false     | 0               | 1          | 2              | RED_HAT_IMAGE
//        "CVE-2019-14866" | 6.7  | "V3"         | 5.9         | "2019-08-30T00:00:00Z" |
//                false     | 0               | 1          | 1              | RED_HAT_IMAGE
    }

    @Unroll
    @Category(BAT)
    def "Verify the results of the CVE GraphQL Query lots of parameters - #query #checkImageCount"() {
        when:
        "Fetch the CVEs using GraphQL"
        def gqlService = new GraphQLService()
        def resultRet = gqlService.Call(GET_CVES_QUERY, [query: "${query}", scopeQuery: ""])
        assert resultRet.getCode() == 200
        println "return code " + resultRet.getCode()

        then:
        "Verify specific CVE data"
        assert resultRet.value.count == 1
        assert resultRet.value.results.size() == 1
        def foundCVE = resultRet.value.results[0]
        assert foundCVE.id == cve
        assert foundCVE.cve == cve
        assert foundCVE.envImpact > 0
        // Use ranges so any new image doesn't break it
        assert foundCVE.deploymentCount > 0 && foundCVE.deploymentCount < 10
        assert !checkImageCount || (foundCVE.imageCount > 0 && foundCVE.imageCount < 20)
        assert foundCVE.componentCount > 0 && foundCVE.componentCount < 10
        assert foundCVE.summary != ""

        where:
        "data inputs"

        query                                                                  | cve             | checkImageCount
        "Deployment:${CVE_DEPLOYMENT_NAME}+Image:nginx:1.9+CVE:CVE-2005-2541"  | "CVE-2005-2541" | true
        "Label:name=cve-deployment+CVE:CVE-2005-2541"                          | "CVE-2005-2541" | true
        "Image:nginx:1.9+CVE:CVE-2005-2541"                                    | "CVE-2005-2541" | true
        "CVSS:10+CVE:CVE-2005-2541"                                            | "CVE-2005-2541" | false
        "Component:tar+CVE:CVE-2005-2541"                                      | "CVE-2005-2541" | false
        "CVE:CVE-2005-2541"                                                    | "CVE-2005-2541" | false
    }
}
