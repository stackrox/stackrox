import static Services.getAlertCounts
import static Services.getAlertGroups
import static Services.getPolicies
import static Services.getViolations
import static Services.waitForViolation
import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest.RequestGroup
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsGroupResponse
import io.stackrox.proto.storage.AlertOuterClass.ListAlert
import org.junit.Assume

import groups.BAT
import org.junit.experimental.categories.Category
import spock.lang.Stepwise
import spock.lang.Unroll
import objects.Deployment
import java.util.stream.Collectors

@Stepwise // We need to verify all of the expected alerts are present before other tests.
class DefaultPoliciesTest extends BaseSpecification {

    // Deployment names
    static final private String NGINX_LATEST = "qadefpolnginxlatest"
    static final private String STRUTS = "qadefpolstruts"
    static final private String SSL_TERMINATOR = "qadefpolsslterm"
    static final private String NGINX_1_10 = "qadefpolnginx110"
    static final private String K8S_DASHBOARD = "kubernetes-dashboard"

    static final private List<Deployment> DEPLOYMENTS = [
        new Deployment()
            .setName (NGINX_LATEST)
            .setImage ("nginx")
            .addPort (22)
            .addLabel ("app", "test")
            .setEnv([SECRET: 'true']),
        new Deployment()
            .setName(STRUTS)
            .setImage("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
            .addLabel("app", "test"),
        new Deployment()
            .setName(SSL_TERMINATOR)
            .setImage("apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest")
            .addLabel("app", "test"),
        new Deployment()
            .setName(NGINX_1_10)
            .setImage("nginx:1.10")
            .addLabel("app", "test"),
    ]

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Unroll
    @Category(BAT)
    def "Verify policy #policyName is triggered" (String policyName, String deploymentName,
                                                  String testId) {
        when:
        "Validate if policy is present"
        assert getPolicies().stream()
                .filter { f -> f.getName() == policyName }
                .collect(Collectors.toList()).size() == 1

        then:
        "Verify Violation for #policyName is triggered"
        assert waitForViolation(deploymentName,  policyName, 30)

        where:
        "Data inputs are:"

        policyName                                      | deploymentName | testId

        "Secure Shell (ssh) Port Exposed"               | NGINX_LATEST   | "C311"

        "Latest tag"                                    | NGINX_LATEST   | ""

        "Environment Variable Contains Secret"          | NGINX_LATEST   | ""

        "Apache Struts: CVE-2017-5638"                  | STRUTS         | "C938"

        "Heartbleed: CVE-2014-0160"                     | SSL_TERMINATOR | "C947"

        "Wget in Image"                                 | STRUTS         | "C939"

        "90-Day Image Age"                              | STRUTS         | "C810"

        "Ubuntu Package Manager in Image"               | STRUTS         | "C931"

        "30-Day Scan Age"                               | SSL_TERMINATOR | "C941"

        "CVSS >= 7"                                     | STRUTS         | "C933"

        "Shellshock: CVE-2014-6271"                     | SSL_TERMINATOR | "C948"

        "Curl in Image"                                 | STRUTS         | "C948"

        "DockerHub NGINX 1.10"                          | NGINX_1_10     | "C823"
    }

    @Category(BAT)
    def "Verify that Kubernetes Dashboard violation is generated"() {
        given:
        "Orchestrator is K8S"
        Assume.assumeTrue(orchestrator.isKubeDashboardRunning())

        expect:
        "Verify Kubernetes Dashboard violation exists"
        waitForViolation(K8S_DASHBOARD,  "Kubernetes Dashboard Deployed", 30)
    }

    @Category(BAT)
    def "Verify that StackRox services don't trigger alerts"() {
        expect:
        "Verify policies are not violated within the stackrox namespace"
        getViolations(
                AlertServiceOuterClass.ListAlertsRequest.newBuilder().setQuery("Namespace:stackrox").build()
        ).size() == 0
    }

    @Category(BAT)
    def "Verify that built-in services don't trigger unexpected alerts"() {
        expect:
        "Verify unexpected policies are not violated within the kube-system namespace"
        getViolations(
          AlertServiceOuterClass.ListAlertsRequest.newBuilder()
            .setQuery("Namespace:kube-system+Policy:!Kubernetes Dashboard").build()
        ).size() == 0
    }

    def queryForDeployments() {
        def query = "Violation State:Active+Deployment:"
        def names = new ArrayList<String>()
        DEPLOYMENTS.each { d ->
            names.add(d.name)
        }
        query += names.join(',')
        return ListAlertsRequest.newBuilder().setQuery(query).build()
    }

    def numUniqueCategories(List<ListAlert> alerts) {
        def m = [] as Set
        alerts.each { a ->
            a.getPolicy().getCategoriesList().each { c ->
                m.add(c)
            }
        }
        return m.size()
    }

    def countAlerts(ListAlertsRequest req, RequestGroup group) {
        def c = getAlertCounts(
                GetAlertsCountsRequest.newBuilder().setRequest(req).setGroupBy(group).build()
        )
        return c
    }

    def totalAlerts(AlertServiceOuterClass.GetAlertsCountsResponse resp) {
        def total = 0
        resp.getGroupsList().each { g ->
            g.getCountsList().each { c ->
                total += c.getCount()
            }
        }
        return total
    }

    @Category(BAT)
    def "Verify that alert counts API is consistent with alerts"()  {
        given:
        def alertReq = queryForDeployments()
        def violations = getViolations(alertReq)
        def uniqueCategories = numUniqueCategories(violations)

        when:
        def ungrouped = countAlerts(alertReq, RequestGroup.UNSET)
        def byCluster = countAlerts(alertReq, RequestGroup.CLUSTER)
        def byCategory = countAlerts(alertReq, RequestGroup.CATEGORY)

        then:
        "Verify counts match expected value"
        ungrouped.getGroupsCount() == 1
        totalAlerts(ungrouped) == violations.size()

        byCluster.getGroupsCount() == 1
        totalAlerts(byCluster) == violations.size()

        byCategory.getGroupsCount() == uniqueCategories
        // Policies can have multiple categories, so the count is _at least_
        // the number of total violations, but usually is more.
        totalAlerts(byCategory) >= violations.size()
    }

    def flattenGroups(GetAlertsGroupResponse resp) {
        def m = [:]
        resp.getAlertsByPoliciesList().each { group ->
            m.put(group.getPolicy().getName(), group.getNumAlerts())
        }
        return m
    }

    @Category(BAT)
    def "Verify that alert groups API is consistent with alerts"()  {
        given:
        def alertReq = queryForDeployments()

        when:
        def groups = getAlertGroups(alertReq)
        def flat = flattenGroups(groups)

        then:
        "Verify expected groups have non-zero counts"
        flat.size() >= 3
        flat["Latest tag"] != 0
        flat["Secure Shell (ssh) Port Exposed"] != 0
        flat["Don't use environment variables with secrets"] != 0
    }
}
