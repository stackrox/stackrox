import static Services.getPolicies
import static Services.waitForViolation

import io.grpc.StatusRuntimeException
import spock.lang.Shared
import services.AlertService
import services.CreatePolicyService
import services.FeatureFlagService
import services.ImageIntegrationService
import common.Constants
import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest.RequestGroup
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsGroupResponse
import io.stackrox.proto.storage.AlertOuterClass.ListAlert
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.PolicyOuterClass.PolicyGroup
import io.stackrox.proto.storage.PolicyOuterClass.PolicySection
import io.stackrox.proto.storage.RiskOuterClass
import io.stackrox.proto.storage.RiskOuterClass.Risk.Result

import org.junit.Assume

import groups.BAT
import groups.SMOKE
import org.junit.experimental.categories.Category
import spock.lang.Stepwise
import spock.lang.Unroll
import objects.Deployment
import objects.GCRImageIntegration
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
    static final private String GCR_NGINX = "qadefpolnginx"

    static final private List<String> WHITELISTED_KUBE_SYSTEM_POLICIES = [
            "Fixable CVSS >= 6 and Privileged",
            "Fixable CVSS >= 7",
            "Ubuntu Package Manager in Image",
            "Red Hat Package Manager in Image",
            "Curl in Image",
            "Wget in Image",
            "Mount Docker Socket",
            Constants.ANY_FIXED_VULN_POLICY,
    ]

    static final private Deployment STRUTS_DEPLOYMENT = new Deployment()
            .setName(STRUTS)
            .setImage("stackrox/qa:struts-app")
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
            .setImage("stackrox/qa:ssl-terminator")
            .addLabel("app", "test")
            .setCommand(["sleep", "600"]),
        new Deployment()
            .setName(NGINX_1_10)
            .setImage("docker.io/nginx:1.10")
            .addLabel("app", "test"),
        new Deployment()
            .setName(GCR_NGINX)
            .setImage("us.gcr.io/stackrox-ci/nginx:1.11.1")
            .addLabel ( "app", "test" )
            .setCommand(["sleep", "600"]),
    ]

    @Shared
    private String gcrId
    @Shared
    private String anyFixedPolicyId

    def setupSpec() {
        def anyFixedPolicy = Policy.newBuilder()
        .setName(Constants.ANY_FIXED_VULN_POLICY)
                .addLifecycleStages(LifecycleStage.DEPLOY)
                .addCategories("Test")
                .setDisabled(false)
                .setSeverityValue(2)
                .addPolicySections(
                        PolicySection.newBuilder().addPolicyGroups(
                                PolicyGroup.newBuilder()
                                        .setFieldName("Fixed By")
                                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(".*"))
                        )
                ).build()
        anyFixedPolicyId = CreatePolicyService.createNewPolicy(anyFixedPolicy)
        assert anyFixedPolicyId

        gcrId = GCRImageIntegration.createDefaultIntegration()
        assert gcrId != ""

        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        orchestrator.createService(new Service(STRUTS_DEPLOYMENT))
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
        assert ImageIntegrationService.deleteImageIntegration(gcrId)
        if (anyFixedPolicyId) {
            CreatePolicyService.deletePolicy(anyFixedPolicyId)
        }
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
        // Some of these policies require scans so extend the timeout as the scan will be done inline
        // with our scanner
        assert waitForViolation(deploymentName,  policyName, 60)

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

        "Ubuntu Package Manager in Image"               | STRUTS           | "C931"

        //"30-Day Scan Age"                               | SSL_TERMINATOR | "C941"

        "Fixable CVSS >= 7"                             | GCR_NGINX      | "C933"

        "Shellshock: Multiple CVEs"                     | SSL_TERMINATOR | "C948"

        "Curl in Image"                                 | STRUTS         | "C948"

        "DockerHub NGINX 1.10"                          | NGINX_1_10     | "C823"
    }

    @Category([BAT, SMOKE])
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

        and:
        "The struts deployment details"
        Deployment dep = DEPLOYMENTS.find { it.name == STRUTS }
        RiskOuterClass.Risk risk = Services.getDeploymentWithRisk(dep.deploymentUid).risk

        expect:
        "Risk factors are present"
        Result riskResult = risk.resultsList.find { it.name == riskFactor }
        def waitTime = 30000
        def start = System.currentTimeMillis()
        while (riskResult == null && (System.currentTimeMillis() - start) < waitTime) {
            risk = Services.getDeploymentWithRisk(dep.deploymentUid).risk
            riskResult = risk.resultsList.find { it.name == riskFactor }
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
                "Image \"docker.io/stackrox/qa:struts-app\"" +
                     " contains \\d{2}\\d+ CVEs with CVSS scores ranging between " +
                     "\\d+(\\.\\d{1,2})? and \\d+(\\.\\d{1,2})?" | []

        "Service Configuration"           | 2.0f     |
                "No capabilities were dropped" | null | []

        "Components Useful for Attackers" | 1.5f     |
                "Image \"docker.io/stackrox/qa:struts-app\" " +
                "contains components useful for attackers:" +
                    " apt, bash, curl, wget" | null | []

        // TODO: This really should be stable, but due to ROX-5269, we may get 168 or 169 components
        "Number of Components in Image"   | 1.5f     | null |
                "Image \"docker.io/stackrox/qa:struts-app\"" +
                " contains 16[89] components" | []

        "Image Freshness"                 | 1.5f     | null | null | []

        "RBAC Configuration"              | 1.0f     |
                "Deployment is configured to automatically mount a token for service account \"default\"" | null |
                []
    }

    @Category(BAT)
    def "Verify that built-in services don't trigger unexpected alerts"() {
        expect:
        "Verify unexpected policies are not violated within the kube-system namespace"
        List<ListAlert> kubeSystemViolations = AlertService.getViolations(
          ListAlertsRequest.newBuilder()
            .setQuery("Namespace:kube-system+Policy:!Kubernetes Dashboard").build()
        )
        List<ListAlert> nonWhitelistedKubeSystemViolations = kubeSystemViolations.stream()
             .filter { x -> !WHITELISTED_KUBE_SYSTEM_POLICIES.contains(x.policy.name) }
             .filter {
                     // ROX-5350 - Ignore alerts for deleted policies
            violation -> Boolean exists = false
                try {
                    Services.getPolicy(violation.policy.id)
                    exists = true
                }
                catch (StatusRuntimeException e) {
                    println "Cannot get the policy associated with the alert: ${e}"
                    println violation
                }
                exists
        }.collect()

        if (nonWhitelistedKubeSystemViolations.size() != 0) {
            nonWhitelistedKubeSystemViolations.forEach {
                violation ->
                println "An unexpected kube-system violation:"
                println violation
                println "The policy details:"
                println Services.getPolicy(violation.policy.id)
            }
        }

        nonWhitelistedKubeSystemViolations.size() == 0
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
