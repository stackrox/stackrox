import static Services.getPolicies
import static Services.waitForViolation

import spock.lang.Shared
import io.stackrox.proto.api.v1.PaginationOuterClass
import services.AlertService
import services.FeatureFlagService
import common.Constants
import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest.RequestGroup
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsGroupResponse
import io.stackrox.proto.storage.AlertOuterClass.ListAlert
import io.stackrox.proto.storage.DeploymentOuterClass
import io.stackrox.proto.storage.DeploymentOuterClass.Risk.Result

import org.junit.Assume

import groups.BAT
import groups.SMOKE
import org.junit.experimental.categories.Category
import spock.lang.Stepwise
import spock.lang.Unroll
import objects.Deployment
import objects.Service
import java.util.stream.Collectors

@Stepwise // We need to verify all of the expected alerts are present before other tests.
class DefaultPoliciesTest extends BaseSpecification {
    // Deployment names
    static final private String NGINX_LATEST = "qadefpolnginxlatest"
    static final private String STRUTS = "qadefpolstruts"
    static final private String SSL_TERMINATOR = "qadefpolsslterm"
    static final private String NGINX_1_10 = "qadefpolnginx110"
    static final private String K8S_DASHBOARD = "kubernetes-dashboard"
    static final private String GCR_NGINX = "qagcrnginx"

    static final private List<String> WHITELISTED_KUBE_SYSTEM_POLICIES = [
            "Fixable CVSS >= 6 and Privileged",
            "Fixable CVSS >= 7",
            "Ubuntu Package Manager in Image",
    ]

    static final private Deployment STRUTS_DEPLOYMENT = new Deployment()
            .setName(STRUTS)
            .setImage("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
            .addLabel("app", "test")
            .addPort(80)

    static final private List<Deployment> DEPLOYMENTS = [
        new Deployment()
            .setName (NGINX_LATEST)
            .setImage ("nginx")
            .addPort (22)
            .addLabel ("app", "test")
            .setEnv([SECRET: 'true']),
        STRUTS_DEPLOYMENT,
        new Deployment()
            .setName(SSL_TERMINATOR)
            .setImage("apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest")
            .addLabel("app", "test")
            .setCommand(["sleep", "600"]),
        new Deployment()
            .setName(NGINX_1_10)
            .setImage("nginx:1.10")
            .addLabel("app", "test"),
        new Deployment()
            .setName(GCR_NGINX)
            .setImage("us.gcr.io/stackrox-ci/nginx:1.11")
            .addLabel ( "app", "test" )
            .setCommand(["sleep", "600"]),
    ]

    @Shared
    private String gcrId

    def setupSpec() {
        for (int i = 1; i <= 6; i++) {
            DEPLOYMENTS.add(
                    new Deployment()
                            .setName("pagination${i}")
                            .setImage("busybox:latest")
                            .addLabel("app", "pagination${i}")
                            .setCommand(["sleep", "600"])
            )
        }
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        orchestrator.createService(new Service(STRUTS_DEPLOYMENT))
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
        gcrId = Services.addGcrRegistryAndScanner()
        assert gcrId != null
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
        assert Services.deleteGcrRegistryAndScanner(gcrId)
    }

    @Unroll
    @Category([BAT, SMOKE])
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

        //"Heartbleed: CVE-2014-0160"                     | SSL_TERMINATOR | "C947"

        "Wget in Image"                                 | STRUTS         | "C939"

        "90-Day Image Age"                              | STRUTS         | "C810"

        "Ubuntu Package Manager in Image"               | STRUTS         | "C931"

        //"30-Day Scan Age"                               | SSL_TERMINATOR | "C941"

        "Fixable CVSS >= 7"                             | GCR_NGINX      | "C933"

        "Shellshock: Multiple CVEs"                     | SSL_TERMINATOR | "C948"

        "Curl in Image"                                 | STRUTS         | "C948"

        "DockerHub NGINX 1.10"                          | NGINX_1_10     | "C823"
    }

    @Category(BAT)
    def "Verify violation pagination"() {
        given:
        "6 violations exist for pagination"
        for (int i = 1; i <= 6; i++) {
            assert Services.waitForViolation("pagination${i}", "Latest tag", 30)
        }

        when:
        "Set pagination limit to 3"
        ListAlertsRequest request = ListAlertsRequest.newBuilder()
                .setQuery("Deployment:pagination+Policy:Latest tag")
                .setPagination(
                PaginationOuterClass.Pagination.newBuilder()
                        .setLimit(3)
                        .setOffset(0)
        ).build()
        def alerts = AlertService.getViolations(request)

        then:
        "verify result set is 3"
        assert alerts.size() == 3

        and:
        "Set limit to 10 with offset to 5 on a total count of 10"
        ListAlertsRequest request2 = ListAlertsRequest.newBuilder()
                .setQuery("Deployment:pagination+Policy:Latest tag")
                .setPagination(
                PaginationOuterClass.Pagination.newBuilder()
                        .setLimit(6)
                        .setOffset(3)
        ).build()
        def alerts2 = AlertService.getViolations(request2)

        then:
        "Verify result set is 3"
        assert alerts2.size() == 3

        and:
        "Get the same violation set in reversed and non-reversed order"
        ListAlertsRequest request3 = ListAlertsRequest.newBuilder()
                .setQuery("Deployment:pagination+Policy:Latest tag,No resource requests or limits specified")
                .setPagination(
                        PaginationOuterClass.Pagination.newBuilder()
                                .setSortOption(
                                        PaginationOuterClass.SortOption.newBuilder()
                                                .setField("Policy")
                                                .setReversed(false))
                ).build()
        def alerts3 = AlertService.getViolations(request3).collect { it.policy.name }
        ListAlertsRequest request4 = ListAlertsRequest.newBuilder()
                .setQuery("Deployment:pagination+Policy:Latest tag,No resource requests or limits specified")
                .setPagination(
                PaginationOuterClass.Pagination.newBuilder()
                        .setSortOption(
                        PaginationOuterClass.SortOption.newBuilder()
                                .setField("Policy")
                                .setReversed(true))
        ).build()
        def alerts4 = AlertService.getViolations(request4).collect { it.policy.name }

        then:
        "make sure the results are the same, just reversed"
        assert alerts3 == alerts4.reverse()
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
        def violations = AlertService.getViolations(
                ListAlertsRequest.newBuilder().setQuery("Namespace:stackrox,Violation State:*").build()
        )
        def unexpectedViolations = violations.findAll {
            def deploymentName = it.deployment.name
            def policyName = it.policy.name
            !Constants.VIOLATIONS_WHITELIST.containsKey(deploymentName) ||
                    !Constants.VIOLATIONS_WHITELIST.get(deploymentName).contains(policyName)
        }
        for (ListAlert violation: unexpectedViolations) {
            print AlertService.getViolation(violation.getId())
        }
        unexpectedViolations == []
    }

    @Unroll
    @Category([BAT])
    def "Verify risk factors on struts deployment: #riskFactor"() {
        given:
        "Check Feature Flags"
        featureDependancies.each {
            Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled(it))
        }

        "The struts deployment details"
        Deployment dep = DEPLOYMENTS.find { it.name == STRUTS }
        DeploymentOuterClass.Deployment deployment = Services.getDeployment(dep.deploymentUid)

        expect:
        "Risk factors are present"
        Result riskResult = deployment.risk.resultsList.find { it.name == riskFactor }
        def waitTime = 30000
        def start = System.currentTimeMillis()
        while (riskResult == null && (System.currentTimeMillis() - start) < waitTime) {
            deployment = Services.getDeployment(dep.deploymentUid)
            riskResult = deployment.risk.resultsList.find { it.name == riskFactor }
            sleep 2000
        }
        riskResult != null
        println "Risk Factor found in ${System.currentTimeMillis() - start}ms: ${riskFactor}"
        riskResult.score <= maxScore
        riskResult.score >= 1.0f

        message == null ?: riskResult.factorsList.get(0).message == message
        regex == null ?: riskResult.factorsList.get(0).message.matches(regex)

        where:
        "data inputs"

        riskFactor                        | maxScore | message   | regex | featureDependancies
        "Policy Violations"               | 4.0f     | null      | null | []

        "Service Reachability"            | 2.0f     |
                "Port 80 is exposed in the cluster"  | null | []

        "Image Vulnerabilities"           | 4.0f     | null |
                // This makes sure it has at least a 100 CVEs.
                "Image \"apollo-dtr.rox.systems/legacy-apps/struts-app:latest\" \\(container \"" +
                     STRUTS + "\"\\) contains \\d{2}\\d+ CVEs with CVSS scores ranging between " +
                     "\\d+(\\.\\d{1,2})? and \\d+(\\.\\d{1,2})?" | []

        "Service Configuration"           | 2.0f     |
                "No capabilities were dropped" | null | []

        "Components Useful for Attackers" | 1.5f     |
                "Image apollo-dtr.rox.systems/legacy-apps/struts-app:latest contains components useful for attackers:" +
                    " apt, bash, curl, wget" | null | []

        "Number of Components in Image"   | 1.5f     |
                "Image apollo-dtr.rox.systems/legacy-apps/struts-app:latest contains 206 components" | null | []

        "Image Freshness"                 | 1.5f     | null | null | []

        "RBAC Configuration"              | 1.0f     |
                "Deployment is configured to automatically mount a token for service account \"default\"" | null |
                [Constants.K8SRBAC_FEATURE_FLAG]
    }

    @Category(BAT)
    def "Verify that built-in services don't trigger unexpected alerts"() {
        expect:
        "Verify unexpected policies are not violated within the kube-system namespace"
        AlertService.getViolations(
          ListAlertsRequest.newBuilder()
            .setQuery("Namespace:kube-system+Policy:!Kubernetes Dashboard").build()
        ).stream().filter { x -> !WHITELISTED_KUBE_SYSTEM_POLICIES.contains(x.policy.name) }.collect().size() == 0
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
        def c = AlertService.getAlertCounts(
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
        def violations = AlertService.getViolations(alertReq)
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
        def groups = AlertService.getAlertGroups(alertReq)
        def flat = flattenGroups(groups)

        then:
        "Verify expected groups have non-zero counts"
        flat.size() >= 3
        flat["Latest tag"] != 0
        flat["Secure Shell (ssh) Port Exposed"] != 0
        flat["Don't use environment variables with secrets"] != 0
    }

}
