import static io.restassured.RestAssured.given
import static util.Helpers.withRetry

import com.opencsv.CSVReader
import groovy.transform.EqualsAndHashCode
import groovy.transform.ToString
import io.restassured.response.Response

import objects.Deployment
import objects.Pagination
import objects.SortOption
import services.GraphQLService
import services.ImageService
import util.Env

import spock.lang.Tag
import spock.lang.Unroll

@Unroll
@Tag("BAT")
@Tag("PZ")
class CSVTest extends BaseSpecification {

    private static final IMAGE_SHA = "sha256:6bf47794f923462389f5a2cda49cf5777f736db8563edc3ff78fb9d87e6e22ec"

    private static final CVE_FIELDS_FRAGEMENT = """
    fragment cveFields on EmbeddedVulnerability {
      id: cve
      cvss
      isFixable(query: \$scopeQuery)
      deploymentCount(query: \$query)
      imageCount(query: \$query)
      componentCount(query: \$query)
      __typename
    }
    """

    private static final CVE_POSTGRES_FIELDS_FRAGEMENT = """
        fragment cveFields on ImageVulnerability {
          id: cve
          cvss
          isFixable(query: \$scopeQuery)
          deploymentCount(query: \$query)
          imageCount(query: \$query)
          componentCount: imageComponentCount(query: \$query)
          __typename
        }
        """

    private static final FIXABLE_CVES_IN_IMAGE_QUERY = """
    query getFixableCvesInImage(\$id: ID!, \$query: String, \$scopeQuery: String, \$vulnQuery: String,
     \$vulnPagination: Pagination) {
      result: image(id: \$id) {
        id
        vulnerabilities: vulns(query: \$vulnQuery, pagination: \$vulnPagination) {
          ...cveFields
          __typename
        }
        __typename
      }
    }
    ${CVE_FIELDS_FRAGEMENT}
    """

    private static final FIXABLE_CVES_IN_IMAGE_POSTGRES_QUERY = """
        query getFixableCvesInImage(\$id: ID!, \$query: String, \$scopeQuery: String, \$vulnQuery: String,
         \$vulnPagination: Pagination) {
          result: image(id: \$id) {
            id
            vulnerabilities: imageVulnerabilities(query: \$vulnQuery, pagination: \$vulnPagination) {
              ...cveFields
              __typename
            }
            __typename
          }
        }
        ${CVE_POSTGRES_FIELDS_FRAGEMENT}
        """

    private static final FIXABLE_CVES_IN_COMPONENT_QUERY = """
    query getFixableCvesInComponent(\$id: ID!, \$query: String, \$scopeQuery: String, \$vulnQuery: String,
     \$vulnPagination: Pagination) {
      result: component(id: \$id) {
        id
        vulnerabilities: vulns(query: \$vulnQuery, pagination: \$vulnPagination) {
          ...cveFields
          __typename
        }
        __typename
      }
    }
    ${CVE_FIELDS_FRAGEMENT}
    """

    private static final FIXABLE_CVES_IN_COMPONENT_POSTGRES_QUERY = """
        query getFixableCvesInComponent(\$id: ID!, \$query: String, \$scopeQuery: String, \$vulnQuery: String,
         \$vulnPagination: Pagination) {
          result: imageComponent(id: \$id) {
            id
            vulnerabilities: imageVulnerabilities(query: \$vulnQuery, pagination: \$vulnPagination) {
              ...cveFields
              __typename
            }
            __typename
          }
        }
        ${CVE_POSTGRES_FIELDS_FRAGEMENT}
        """

    private static final FIXABLE_CVES_IN_DEPLOYMENT_QUERY = """
    query getFixableCvesInDeployment(\$id: ID!, \$query: String, \$scopeQuery: String, \$vulnQuery: String,
     \$vulnPagination: Pagination) {
      result: deployment(id: \$id) {
        id
        vulnerabilities: vulns(query: \$vulnQuery, pagination: \$vulnPagination) {
          ...cveFields
          __typename
        }
        __typename
      }
    }
    ${CVE_FIELDS_FRAGEMENT}
    """

    private static final FIXABLE_CVES_IN_DEPLOYMENT_POSTGRES_QUERY = """
        query getFixableCvesInDeployment(\$id: ID!, \$query: String, \$scopeQuery: String, \$vulnQuery: String,
         \$vulnPagination: Pagination) {
          result: deployment(id: \$id) {
            id
            vulnerabilities: imageVulnerabilities(query: \$vulnQuery, pagination: \$vulnPagination) {
              ...cveFields
              __typename
            }
            __typename
          }
        }
        ${CVE_POSTGRES_FIELDS_FRAGEMENT}
        """

    private static final Map<String, String> QUERIES = [
            "FIXABLE_CVES_IN_IMAGE_QUERY"     : FIXABLE_CVES_IN_IMAGE_QUERY,
            "FIXABLE_CVES_IN_COMPONENT_QUERY" : FIXABLE_CVES_IN_COMPONENT_QUERY,
            "FIXABLE_CVES_IN_DEPLOYMENT_QUERY": FIXABLE_CVES_IN_DEPLOYMENT_QUERY,
    ]

    private static final Map<String, String> PG_QUERIES = [
            "FIXABLE_CVES_IN_IMAGE_QUERY"     : FIXABLE_CVES_IN_IMAGE_POSTGRES_QUERY,
            "FIXABLE_CVES_IN_COMPONENT_QUERY" : FIXABLE_CVES_IN_COMPONENT_POSTGRES_QUERY,
            "FIXABLE_CVES_IN_DEPLOYMENT_QUERY": FIXABLE_CVES_IN_DEPLOYMENT_POSTGRES_QUERY,
    ]

    private static final Deployment CVE_DEPLOYMENT = new Deployment()
            .setName("nginx-deployment")
            .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
            .addLabel("app", "test")

    def setupSpec() {
        ImageService.scanImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
        orchestrator.createDeployment(CVE_DEPLOYMENT)
        assert Services.waitForDeployment(CVE_DEPLOYMENT)
        // wait for all image CVEs to be discovered and added to db
        sleep(5000)
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(CVE_DEPLOYMENT)
    }

    def secondarySortByID(List<CVE> list) {
        list.sort {
            a, b -> a.cvss == b.cvss ? (a.id < b.id ? -1 : 1) : 0
        }
    }

    // Non-postgres runs
    // "CVE", "CVE Type(s)", "Fixable", "CVSS Score (version)", "Env Impact (%)", "Impact Score", "Deployments",
    // "Images", "Nodes", "Components", "Scanned", "Published", "Summary"
    // Postgres runs
    // "Image CVE", "Fixable", "CVSS Score", "Env Impact (%s)", "Impact Score", "Deployments", "Images",
    // "Image Components", "Last Scanned", "Published", "Summary"

    def getCVEIndex() {
        return 0
    }

    def getCVSSScoreIndex() {
        return isPostgresRun() ? 2 : 3
    }

    def getDeploymentCountIndex() {
        return isPostgresRun() ? 5 : 6
    }

    def getImageCountIndex() {
        return isPostgresRun() ? 6 : 7
    }

    def getImageComponentCountIndex() {
        return isPostgresRun() ? 7 : 9
    }

    def getComponentId() {
        return isPostgresRun() ? "openssl#1.1.1d-0+deb10u7#debian:10" : "b3BlbnNzbA:MS4wLjFrLTMrZGViOHU1"
    }

    def getComponentQuery() {
        return "COMPONENT ID:" + getComponentId() + "+Fixable:true"
    }

    def getCVETypeImageQuery() {
        return "CVE Type:IMAGE_CVE+"
    }

    Map<String, Object> payload(String id) {
        def pagination = new Pagination(0, 0, new SortOption("cvss", true))
        return [
                id            : id,
                query         : "",
                scopeQuery    : "",
                vulnQuery     : "Fixable:true",
                vulnPagination: pagination,
        ]
    }

    @Tag("BAT")
    def "Verify CVE CSV data scoped by entity is correct #description"() {
        given:
        def graphQLPayload = payload(id)
        def csvQuery = getCVETypeImageQuery() + query
        def graphQLQuery = isPostgresRun() ? PG_QUERIES[description] : QUERIES[description]

        when:
        "Query fixable CVEs from graphQL"
        def gqlService = new GraphQLService()
        def ret = gqlService.Call(graphQLQuery, graphQLPayload)
        assert ret.getCode() == 200
        assert ret.value.result.vulnerabilities.toList().size() > 0

        def graphQLCVEs = ret.value.result.vulnerabilities.collect { def vuln ->
            new CVE(vuln.id, vuln.cvss, vuln.deploymentCount, vuln.imageCount, vuln.componentCount)
        }

        and:
        "Fetch fixable CVE CSV"
        Response response = null
        def csvEndpoint = "/api/vm/export/csv"
        if (isPostgresRun()) {
            csvEndpoint = "/api/export/csv/image/cve"
        }
        def csvURL = "https://${Env.mustGetHostname()}:${Env.mustGetPort()}" + csvEndpoint
        withRetry(10, 3) {
            response = given()
                    .auth().preemptive().basic(Env.mustGetUsername(), Env.mustGetPassword())
                    .relaxedHTTPSValidation()
                    .param("query", csvQuery)
                    .param("pagination.sortOption.field", "cvss")
                    .param("pagination.sortOption.reversed", "true")
                    .urlEncodingEnabled(true)
                    .when()
                    .get(csvURL)
            assert response.statusCode == 200
        }

        List<String[]> lines = []
        CSVReader reader
        try {
            reader = new CSVReader(new InputStreamReader(response.body().asInputStream()))
            lines = reader.readAll()
        } catch (Exception e) {
            log.error("Could not read response body", e)
        } finally {
            try {
                if (reader != null) {
                    reader.close()
                }
            } catch (IOException e) {
                log.error("Could not close reader", e)
            }
        }

        log.info "Number of CVEs received from CSV endpoint: " + lines.size()

        def csvCVEs = new ArrayList<CVE>()
        for (int i = 1; i < lines.size(); i++) {
            // "CVE", "CVE Type(s)", "Fixable", "CVSS Score (version)", "Env Impact (%)", "Impact Score", "Deployments",
            // "Images", "Nodes", "Components", "Scanned", "Published", "Summary"
            csvCVEs.add(
                    new CVE(lines.get(i)[getCVEIndex()],
                            lines.get(i)[getCVSSScoreIndex()].split()[0].toFloat(),
                            lines.get(i)[getDeploymentCountIndex()].toInteger(),
                            lines.get(i)[getImageCountIndex()].toInteger(),
                            lines.get(i)[getImageComponentCountIndex()].toInteger())
            )
        }

        then:
        "Ensure that the CVEs from graphQL and CSV match"
        assert csvCVEs.size() == graphQLCVEs.size()

        secondarySortByID(csvCVEs)
        secondarySortByID(graphQLCVEs)

        for (def i = 0; i < csvCVEs.size(); i++) {
            assert csvCVEs.get(i) == graphQLCVEs.get(i)
        }

        where:
        "Data is"

        description                        | id                           | query
        "FIXABLE_CVES_IN_IMAGE_QUERY"      | IMAGE_SHA                    | "Image Sha:${IMAGE_SHA}+Fixable:true"
        "FIXABLE_CVES_IN_COMPONENT_QUERY"  | getComponentId()             | getComponentQuery()
        "FIXABLE_CVES_IN_DEPLOYMENT_QUERY" | CVE_DEPLOYMENT.deploymentUid |
                "Deployment ID:${CVE_DEPLOYMENT.deploymentUid}+Fixable:true"
    }

    @EqualsAndHashCode(includeFields = true)
    @ToString(includes = "id,cvss")
    class CVE {
        String id
        float cvss
        int deploymentCount
        int imageCount
        int componentCount

        CVE(String id, float cvss, int deploymentCount, int imageCount, int componentCount) {
            this.id = id
            this.cvss = cvss
            this.deploymentCount = deploymentCount
            this.imageCount = imageCount
            this.componentCount = componentCount
        }
    }
}
