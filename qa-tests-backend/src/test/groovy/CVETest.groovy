import com.google.protobuf.util.Timestamps
import groups.BAT
import objects.Deployment
import org.junit.experimental.categories.Category
import services.GraphQLService
import services.ImageService
import spock.lang.Ignore
import spock.lang.IgnoreIf
import spock.lang.Unroll
import util.Env

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
        discoveredAtImage(query: \$scopeQuery)
        envImpact
        publishedOn
        isFixable(query: \$scopeQuery)
        deploymentCount(query: \$query)
        imageCount(query: \$query)
        componentCount(query: \$query)
        __typename
    }
    """

    private static final COMPONENT_CVE_QUERY = """
    query getComponentCVE(\$id: ID!, \$pagination: Pagination, \$query: String, \$scopeQuery: String)
    {
        result: component(id: \$id) {
            id
            name
            version
            vulns(query: \$query, pagination: \$pagination) {
            ...cveFields
            __typename
            }
            unusedVarSink(query: \$scopeQuery)
            __typename
        }
    }

    fragment cveFields on EmbeddedVulnerability {
        id: cve
        cve
        cvss
        vulnerabilityType
        scoreVersion
        envImpact
        impactScore
        summary
        fixedByVersion
        isFixable(query: \$scopeQuery)
        createdAt
        discoveredAtImage(query: \$scopeQuery)
        __typename
    }
    """

    private static final IMAGE_CVE_QUERY = """
    query getImageCVE(\$id: ID!, \$pagination: Pagination, \$query: String, \$policyQuery: String, \$scopeQuery: String)
    {
        result: image(id: \$id) {
            id
            lastUpdated
            vulnCount(query: \$query)
            vulns(query: \$query, pagination: \$pagination) {
            ...cveFields
            __typename
            }
            unusedVarSink(query: \$policyQuery)
            unusedVarSink(query: \$scopeQuery)
            __typename
        }
    }

    fragment cveFields on EmbeddedVulnerability {
        id: cve
        cve
        cvss
        vulnerabilityType
        scoreVersion
        envImpact
        impactScore
        summary
        fixedByVersion
        isFixable(query: \$scopeQuery)
        createdAt
        discoveredAtImage(query: \$scopeQuery)
        publishedOn
        deploymentCount(query: \$query)
        imageCount(query: \$query)
        componentCount(query: \$query)
        __typename
    }
    """

    private static final FIXABLE_CVES_BY_ENTITY_QUERY = """
    query getFixableCvesForEntity(\$id: ID!, \$query: String, \$scopeQuery: String, \$vulnQuery: String,
     \$vulnPagination: Pagination) {
      result: image(id: \$id) {
        id
        vulnCounter {
          all {
            fixable
            __typename
          }
          __typename
        }
        vulnerabilities: vulns(query: \$vulnQuery, pagination: \$vulnPagination) {
          ...cveFields
          __typename
        }
        __typename
      }
    }

    fragment cveFields on EmbeddedVulnerability {
      id: cve
      cve
      cvss
      vulnerabilityType
      scoreVersion
      envImpact
      impactScore
      summary
      fixedByVersion
      isFixable(query: \$scopeQuery)
      createdAt
      publishedOn
      deploymentCount(query: \$query)
      imageCount(query: \$query)
      componentCount(query: \$query)
      __typename
    }
    """

    private static final SCOPED_FIXABLE_QUERY = """
    query getCve(\$id: ID!, \$scopeQuery: String) {
      result: vulnerability(id: \$id) {
        isFixable(query: \$scopeQuery)
      }
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
    static final private UBUNTU_IMAGE =
            "docker.io/library/ubuntu:latest@sha256:ffc76f71dd8be8c9e222d420dc96901a07b61616689a44c7b3ef6a10b7213de4"

    static final private FIXABLE_VULN_IMAGE_DIGEST =
            "sha256:b74ad76891c58909fa57534513cefe11fb5917f1f1095ceef80a6343666c096f"

    static final private FIXABLE_VULN_IMAGE =
            "docker.io/sandyg1/om-cred-auto@${FIXABLE_VULN_IMAGE_DIGEST}"

    static final private UNFIXABLE_VULN_IMAGE_DIGEST =
            "sha256:df7d71b9a1ce0fa0b774f52ac7a0d966b483f0650185cc3594ff7d367d5c6a55"

    static final private UNFIXABLE_VULN_IMAGE =
            "docker.io/library/debian@${UNFIXABLE_VULN_IMAGE_DIGEST}"

    def setupSpec() {
        ImageService.scanImage("us.gcr.io/stackrox-ci/nginx:1.9")
        ImageService.scanImage(NGINX_1_10_2_IMAGE)
        ImageService.scanImage(RED_HAT_IMAGE)
        ImageService.scanImage(UBUNTU_IMAGE)
        ImageService.scanImage(FIXABLE_VULN_IMAGE)
        ImageService.scanImage(UNFIXABLE_VULN_IMAGE)
        orchestrator.createDeployment(CVE_DEPLOYMENT)
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(CVE_DEPLOYMENT)
    }

    @Unroll
    @Category(BAT)
    @Ignore("The CVE(s) that these tests depend upon stopped serving the purpose. ROX-6518 ROX-6348")
    def "Verify the results of the CVE GraphQL Query with single specific CVE match - #cve"() {
        when:
        "Fetch the CVEs using GraphQL"
        def gqlService = new GraphQLService()
        def resultRet = gqlService.Call(GET_CVES_QUERY,
                [query: "Image:${image}+CVE:${cve}", scopeQuery: ""])
        assert resultRet.getCode() == 200
        log.info "return code " + resultRet.getCode()

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
    @Ignore("The CVE(s) that these tests depend upon stopped serving the purpose. ROX-6518 ROX-6348")
    def "Verify the results of the CVE GraphQL Query lots of parameters - #query #checkImageCount"() {
        when:
        "Fetch the CVEs using GraphQL"
        def gqlService = new GraphQLService()

        def resultRet = gqlService.Call(GET_CVES_QUERY, [query: "${query}", scopeQuery: ""])
        assert resultRet.getCode() == 200
        log.info "return code " + resultRet.getCode()

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

        query                                                                  |
                cve             | checkImageCount
        "Deployment:${CVE_DEPLOYMENT_NAME}+Image:quay.io/rhacs-eng/qa:nginx-1-9+CVE:CVE-2005-2541"  |
                "CVE-2005-2541" | true
        "Label:name=cve-deployment+CVE:CVE-2005-2541"                          |
                "CVE-2005-2541" | true
        "Image:quay.io/rhacs-eng/qa:nginx-1-9+CVE:CVE-2005-2541"               |
                "CVE-2005-2541" | true
        "CVSS:10+CVE:CVE-2005-2541"                                            |
                "CVE-2005-2541" | false
        "Component:tar+CVE:CVE-2005-2541"                                      |
                "CVE-2005-2541" | false
        "CVE:CVE-2005-2541"                                                    |
                "CVE-2005-2541" | false
    }

    @Unroll
    @Category(BAT)
    @Ignore("The CVE(s) that these tests depend upon stopped serving the purpose. ROX-6518 ROX-6348")
    def "Verify IsFixable when scoped by images"() {
        when:
        "Scan two images that have CVE-2019-14866, but have differing CVE status (ubuntu is fixable, centos is not)"
        def gqlService = new GraphQLService()
        def centosRet = gqlService.Call(IMAGE_CVE_QUERY, [
                id: "sha256:4ec83eee30dfbaba2e93f59d36cc360660d13f73c71af179eeb9456dd95d1798",
                query: "CVE:CVE-2019-14866",
                scopeQuery: "IMAGE SHA:sha256:4ec83eee30dfbaba2e93f59d36cc360660d13f73c71af179eeb9456dd95d1798",
        ])
        assert centosRet.getCode() == 200

        def ubuntuRet = gqlService.Call(IMAGE_CVE_QUERY, [
                id: "sha256:ffc76f71dd8be8c9e222d420dc96901a07b61616689a44c7b3ef6a10b7213de4",
                query: "CVE:CVE-2019-14866",
                scopeQuery: "IMAGE SHA:sha256:ffc76f71dd8be8c9e222d420dc96901a07b61616689a44c7b3ef6a10b7213de4",
        ])

        assert ubuntuRet.getCode() == 200

        then:
        "Verify centos is not fixable, but ubuntu is"
        assert centosRet.value.result.vulns.size() == 1
        assert !centosRet.value.result.vulns[0].isFixable

        assert ubuntuRet.value.result.vulns.size() == 1
        assert ubuntuRet.value.result.vulns[0].isFixable
    }

    @Unroll
    @Category(BAT)
    @Ignore("The CVE(s) that these tests depend upon stopped serving the purpose. ROX-6518 ROX-6348")
    def "Verify CreatedAt(DiscoveredAtSystem) and DiscoveredAtImage when scoped by images"() {
        when:
        "Scan two images having same CVE but different scan time and CVE is queried nested through image resolver"
        def gqlService = new GraphQLService()
        def centosRet = gqlService.Call(IMAGE_CVE_QUERY, [
                id: "sha256:4ec83eee30dfbaba2e93f59d36cc360660d13f73c71af179eeb9456dd95d1798",
                query: "CVE:CVE-2019-14866",
        ])
        assert centosRet.getCode() == 200
        assert centosRet.value.result.vulns.size() == 1

        def ubuntuRet = gqlService.Call(IMAGE_CVE_QUERY, [
                id: "sha256:ffc76f71dd8be8c9e222d420dc96901a07b61616689a44c7b3ef6a10b7213de4",
                query: "CVE:CVE-2019-14866",
        ])
        assert ubuntuRet.getCode() == 200
        assert ubuntuRet.value.result.vulns.size() == 1

        def centosCVECreatedAt = Timestamps.parse(centosRet.value.result.vulns[0].createdAt)
        def ubuntuCVECreatedAt = Timestamps.parse(ubuntuRet.value.result.vulns[0].createdAt)
        def centoCVEDiscovery = Timestamps.parse(centosRet.value.result.vulns[0].discoveredAtImage)
        def ubuntuCVEDiscovery = Timestamps.parse(ubuntuRet.value.result.vulns[0].discoveredAtImage)

        then:
        "Verify CVE discovery time (System) is same"
        assert Timestamps.compare(centosCVECreatedAt, ubuntuCVECreatedAt) == 0

        and:
        "Verify CVE discovery time (Image) for centos and ubuntu image is not the same"
        assert Timestamps.compare(centoCVEDiscovery, ubuntuCVEDiscovery) != 0
        assert Timestamps.compare(centosCVECreatedAt, centoCVEDiscovery) <= 0 &&
                Timestamps.compare(centoCVEDiscovery, Timestamps.parse(centosRet.value.result.lastUpdated)) <= 0
        assert Timestamps.compare(ubuntuCVECreatedAt, ubuntuCVEDiscovery) <= 0 &&
                Timestamps.compare(ubuntuCVEDiscovery, Timestamps.parse(ubuntuRet.value.result.lastUpdated)) <= 0
    }

    @Unroll
    @Category(BAT)
    @Ignore("The CVE(s) that these tests depend upon stopped serving the purpose. ROX-6518 ROX-6348")
    def "Verify CreatedAt(DiscoveredAtSystem) and DiscoveredAtImage when not scoped by images"() {
        when:
        "Scan centos image and CVE is queried directly using vulnerability resolver"
        def gqlService = new GraphQLService()
        def ret = gqlService.Call(GET_CVES_QUERY, [
                query: "CVE:CVE-2019-14866",
        ])
        assert ret.getCode() == 200
        assert ret.value.results.size() == 1

        def centosRet = gqlService.Call(IMAGE_CVE_QUERY, [
                id: "sha256:4ec83eee30dfbaba2e93f59d36cc360660d13f73c71af179eeb9456dd95d1798",
                query: "CVE:CVE-2019-14866",
        ])
        assert centosRet.getCode() == 200
        assert centosRet.value.result.vulns.size() == 1

        then:
        "Verify CVE discovery time (System) is same as image scoped query response"
        assert ret.value.results[0].createdAt == centosRet.value.result.vulns[0].createdAt

        and:
        "Verify CVE discovery time (Image) is null"
        assert !ret.value.results[0].discoveredAtImage
    }

    @Unroll
    @Category(BAT)
    @Ignore("The CVE(s) that these tests depend upon stopped serving the purpose. ROX-6518 ROX-6348")
    def "Verify CreatedAt(DiscoveredAtSystem) and DiscoveredAtImage when scoped by resources other than images"() {
        when:
        "Scan centos image and CVE is queried nested through image component resolver"
        def gqlService = new GraphQLService()
        def ret = gqlService.Call(COMPONENT_CVE_QUERY, [
                id: "Y3Bpbw:Mi4xMi04LmVsOA", // cpio 2.12-8.el8
                query: "CVE:CVE-2019-14866",
                scopeQuery: "IMAGE SHA:sha256:4ec83eee30dfbaba2e93f59d36cc360660d13f73c71af179eeb9456dd95d1798",
        ])
        assert ret.getCode() == 200
        assert ret.value.result.vulns.size() == 1

        def centosRet = gqlService.Call(IMAGE_CVE_QUERY, [
                id: "sha256:4ec83eee30dfbaba2e93f59d36cc360660d13f73c71af179eeb9456dd95d1798",
                query: "CVE:CVE-2019-14866",
        ])
        assert centosRet.getCode() == 200
        assert centosRet.value.result.vulns.size() == 1

        then:
        "Verify CVE discovery time (System) of component-cve query is same as image-cve query"
        assert ret.value.result.vulns[0].createdAt == centosRet.value.result.vulns[0].createdAt

        and:
        "Verify CVE discovery time (Image) of component-cve query is same as image-cve query"
        assert ret.value.result.vulns[0].discoveredAtImage == centosRet.value.result.vulns[0].discoveredAtImage
    }

    @Category(BAT)
    @IgnoreIf({ Env.CI_JOBNAME.contains("postgres") })
    def "Verify IsFixable for entities when scoped by CVE is still correct"() {
        when:
        "Query fixable CVEs by a specific CVE in the image"
        def gqlService = new GraphQLService()
        def ret = gqlService.Call(FIXABLE_CVES_BY_ENTITY_QUERY, [
                id: "sha256:4ec83eee30dfbaba2e93f59d36cc360660d13f73c71af179eeb9456dd95d1798",
                query: "",
                scopeQuery: "CVE:CVE-2020-8285",
                vulnQuery: "Fixable:true",
        ])

        then:
        "Ensure that other CVEs are fixable despite the CVE scope"
        ret.getCode() == 200
        ret.value.result.vulnerabilities.toList().findAll { x -> x.isFixable }.size() > 1
    }

    @Unroll
    @Category(BAT)
    @IgnoreIf({ Env.CI_JOBNAME.contains("postgres") })
    def "Verify IsFixable is correct when scoped (#digest, #fixable)"() {
        when:
        "Query fixable CVEs by a specific CVE in the image"
        def gqlService = new GraphQLService()
        def ret = gqlService.Call(SCOPED_FIXABLE_QUERY, [
                id: "CVE-2019-9893",
                scopeQuery: "Image Sha:${digest}+CVE:CVE-2019-9893",
        ])

        then:
        "Ensure the fixable status matches expectations"
        ret.getCode() == 200
        assert ret.value.result.isFixable == fixable

        where:
        "data inputs"

        digest | fixable
        FIXABLE_VULN_IMAGE_DIGEST | true
        UNFIXABLE_VULN_IMAGE_DIGEST | false
    }

}
