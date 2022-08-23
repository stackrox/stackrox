import static com.jayway.restassured.RestAssured.given
import com.jayway.restassured.response.Response
import com.opencsv.CSVReader
import groovy.transform.EqualsAndHashCode
import groups.BAT
import objects.Deployment
import objects.Pagination
import objects.SortOption
import org.junit.experimental.categories.Category
import services.GraphQLService
import services.ImageService
import spock.lang.IgnoreIf
import spock.lang.Retry
import spock.lang.Unroll
import util.Env

@Retry(count = 0)
@Unroll
@Category(BAT)
class CSVTest extends BaseSpecification {
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

    static final private Deployment CVE_DEPLOYMENT = new Deployment()
            .setName("nginx-deployment")
            .setImage("quay.io/rhacs-eng/qa:nginx-1-9")
            .addLabel("app", "test")

    def setupSpec() {
        ImageService.scanImage("quay.io/rhacs-eng/qa:nginx-1-9")
        orchestrator.createDeployment(CVE_DEPLOYMENT)
        assert Services.waitForDeployment(CVE_DEPLOYMENT)
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(CVE_DEPLOYMENT)
    }

    def secondarySortByID(List<CVE> list) {
        list.sort {
            a, b -> a.cvss == b.cvss ? (a.id < b.id ? -1 : 1) : 0
        }
    }

    @Category(BAT)
    def "Verify CVE CSV data scoped by entity is correct"() {
        when:
        "Query fixable CVEs from graphQL"
        def gqlService = new GraphQLService()
        def graphQLQuery = ""
        if (Env.CI_JOBNAME.contains("postgres")) {
            graphQLQuery = postgresGraphQLQuery
        } else {
            graphQLQuery = baseGraphQLQuery
        }
        def ret = gqlService.Call(graphQLQuery, graphQLPayload)
        assert ret.getCode() == 200
        assert ret.value.result.vulnerabilities.toList().size() > 0

        def graphQLCVEs = new ArrayList<CVE>()
        for (def vuln : ret.value.result.vulnerabilities) {
            graphQLCVEs.add(new CVE(vuln.id, vuln.cvss, vuln.deploymentCount, vuln.imageCount, vuln.componentCount))
        }

        and:
        "Fetch fixable CVE CSV"
        Response response = null
        def csvEndpoint = "/api/vm/export/csv"
        if (Env.CI_JOBNAME.contains("postgres")) {
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
                    new CVE(lines.get(i)[0],
                            lines.get(i)[3].split()[0].toFloat(),
                            lines.get(i)[6].toInteger(),
                            lines.get(i)[7].toInteger(),
                            lines.get(i)[9].toInteger())
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

        where :
        "Data is"

        baseGraphQLQuery                | postgresGraphQLQuery                       | graphQLPayload | csvQuery
        FIXABLE_CVES_IN_IMAGE_QUERY     | FIXABLE_CVES_IN_IMAGE_POSTGRES_QUERY       | [
                id        : "sha256:e18c5814a9f7ddd5fe410f17417a48d2de562325e9d71337274134f4a6654e3f",
                query: "",
                // must scope without scope query since graphQL is hitting sub-resolver
                scopeQuery: "",
                vulnQuery : "Fixable:true",
                vulnPagination: new Pagination(0, 0, new SortOption("cvss", true)),
        ] | "Image Sha:sha256:e18c5814a9f7ddd5fe410f17417a48d2de562325e9d71337274134f4a6654e3f+Fixable:true"
        FIXABLE_CVES_IN_COMPONENT_QUERY | FIXABLE_CVES_IN_COMPONENT_POSTGRES_QUERY   | [
                // openssl 1.0.1k-3+deb8u5
                id        : "b3BlbnNzbA:MS4wLjFrLTMrZGViOHU1",
                query: "",
                scopeQuery: "",
                vulnQuery : "Fixable:true",
                vulnPagination: new Pagination(0, 0, new SortOption("cvss", true)),
        ] | "COMPONENT ID:b3BlbnNzbA:MS4wLjFrLTMrZGViOHU1+Fixable:true"
        FIXABLE_CVES_IN_DEPLOYMENT_QUERY | FIXABLE_CVES_IN_DEPLOYMENT_POSTGRES_QUERY | [
                id        : CVE_DEPLOYMENT.deploymentUid,
                query: "",
                scopeQuery: "",
                vulnQuery : "Fixable:true",
                vulnPagination: new Pagination(0, 0, new SortOption("cvss", true)),
        ] | "Deployment ID:${CVE_DEPLOYMENT.deploymentUid}+Fixable:true"
    }

    @EqualsAndHashCode(includeFields = true)
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
