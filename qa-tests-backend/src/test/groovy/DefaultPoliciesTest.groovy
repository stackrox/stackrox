import static Services.getPolicies
import static Services.waitForViolation

import java.util.concurrent.TimeUnit
import java.util.stream.Collectors

import io.grpc.StatusRuntimeException

import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest.RequestGroup
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsGroupResponse
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.api.v1.PolicyServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.AlertOuterClass.ListAlert
import io.stackrox.proto.storage.DeploymentOuterClass
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.PolicyOuterClass.PolicyGroup
import io.stackrox.proto.storage.PolicyOuterClass.PolicySection
import io.stackrox.proto.storage.RiskOuterClass
import io.stackrox.proto.storage.RiskOuterClass.Risk.Result

import common.Constants
import objects.Deployment
import objects.GCRImageIntegration
import objects.Service
import services.AlertService
import services.DeploymentService
import services.FeatureFlagService
import services.ImageIntegrationService
import services.ImageService
import services.PolicyService
import util.Env
import util.Helpers
import util.SlackUtil

import org.junit.Assume
import org.junit.Rule
import org.junit.rules.Timeout
import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Stepwise
import spock.lang.Tag
import spock.lang.Unroll

@Tag("PZDebug")
// TODO(ROX-13738): Re-enable these tests in compatibility-test step
@Stepwise // We need to verify all of the expected alerts are present before other tests.
class DefaultPoliciesTest extends BaseSpecification {
    // Deployment names
    static final private String NGINX_LATEST = "qadefpolnginxlatest"
    static final private String STRUTS = "qadefpolstruts"
    //static final private String SSL_TERMINATOR = "qadefpolsslterm"
    static final private String TRIGGER_MOST = "qadefpoltriggermost"
    static final private String K8S_DASHBOARD = "kubernetes-dashboard"
    static final private String GCR_NGINX = "qadefpolnginx"
    static final private String WGET_CURL = ((Env.REMOTE_CLUSTER_ARCH == "x86_64") ? STRUTS:TRIGGER_MOST)
    static final private String STRUTS_IMAGE = ((Env.REMOTE_CLUSTER_ARCH == "x86_64") ?
        "quay.io/rhacs-eng/qa:struts-app":"quay.io/rhacs-eng/qa-multi-arch:struts-app")
    static final private String COMPONENTS = ((Env.REMOTE_CLUSTER_ARCH == "x86_64") ?
        " apt, bash, curl, wget":" apt, bash, curl")

    @Shared
    private String componentCount = ""

    static final private List<String> WHITELISTED_KUBE_SYSTEM_POLICIES = [
            "Fixable CVSS >= 6 and Privileged",
            "Privileged Container(s) with Important and Critical CVE(s)",
            "Fixable Severity at least Important",
            "Ubuntu Package Manager in Image",
            "Red Hat Package Manager in Image",
            "Curl in Image",
            "Wget in Image",
            "Mount Container Runtime Socket",
            "Docker CIS 5.15: Ensure that the host's process namespace is not shared",
            "Docker CIS 5.7: Ensure privileged ports are not mapped within containers",
            Constants.ANY_FIXED_VULN_POLICY,
    ]

    static final private List<String> WHITELISTED_KUBE_SYSTEM_DEPLOYMENTS_AND_POLICIES = [
            "tunnelfront - Secure Shell Server (sshd) Execution",
            "tunnelfront - Docker CIS 4.7: Alert on Update Instruction",
            "webhookserver - Kubernetes Actions: Port Forward to Pod",
    ]

    static final private Deployment STRUTS_DEPLOYMENT = new Deployment()
            .setName(STRUTS)
            .setImage(STRUTS_IMAGE)
            .addLabel("app", "test")
            .addPort(80)

    static final private List<Deployment> DEPLOYMENTS = [
        new Deployment()
            .setName (NGINX_LATEST)
            // this is docker.io/nginx:1.23.3 but tagged as latest
            .setImage ("quay.io/rhacs-eng/qa-multi-arch-nginx:latest")
            .addPort (22)
            .addLabel ("app", "test")
            .setEnv([SECRET: 'true']),
        STRUTS_DEPLOYMENT,
        // new Deployment()
        //     .setName(SSL_TERMINATOR)
        //     .setImage("quay.io/rhacs-eng/qa:ssl-terminator")
        //     .addLabel("app", "test")
        //     .setCommand(["sleep", "600"]),
        new Deployment()
            .setName(TRIGGER_MOST)
            .setImage("quay.io/rhacs-eng/qa-multi-arch:trigger-policy-violations-most-v1")
            .addLabel("app", "test"),
        new Deployment()
            .setName(GCR_NGINX)
            .setImage("us.gcr.io/stackrox-ci/qa-multi-arch:nginx-1.12")
            .addLabel ( "app", "test" )
            .setCommand(["sleep", "600"]),
    ]

    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT = 300

    // Override the global JUnit test timeout to cover a test instance waiting
    // WAIT_FOR_VIOLATION_TIMEOUT over three test tries and the appprox. 6
    // minutes it can take to gather debug when the first test run fails plus
    // some padding.
    @Rule
    @SuppressWarnings(["JUnitPublicProperty"])
    Timeout globalTimeout = new Timeout(3*WAIT_FOR_VIOLATION_TIMEOUT + 300 + 120, TimeUnit.SECONDS)

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
                                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(".*")
                                        .build())
                                .build()
                        ).build()
                ).build()
        anyFixedPolicyId = PolicyService.createNewPolicy(anyFixedPolicy)
        assert anyFixedPolicyId

        gcrId = GCRImageIntegration.createDefaultIntegration()
        assert gcrId != ""

        ImageService.clearImageCaches()
        for (Deployment deployment : DEPLOYMENTS) {
            ImageService.deleteImages(
                    SearchServiceOuterClass.RawQuery.newBuilder().setQuery("Image:${deployment.getImage()}").build(),
                    true)
        }
        ImageService.deleteImages(
                SearchServiceOuterClass.RawQuery.newBuilder().setQuery("Image:${STRUTS_DEPLOYMENT.getImage()}").build(),
                true)

        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        orchestrator.createService(new Service(STRUTS_DEPLOYMENT))
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
        Helpers.collectImageScanForDebug(
                STRUTS_DEPLOYMENT.getImage(), 'default-policies-test-struts-app.json'
        )

        switch (Env.REMOTE_CLUSTER_ARCH) {
            case "s390x":
                componentCount=92
                break
            case "ppc64le":
                componentCount=91
                break
            default:
                componentCount=169
                break
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
        assert ImageIntegrationService.deleteImageIntegration(gcrId)
        if (anyFixedPolicyId) {
            PolicyService.deletePolicy(anyFixedPolicyId)
        }
    }

    @Unroll
    @Tag("BAT")
    @Tag("SMOKE")
    def "Verify policy #policyName is triggered" (String policyName, String deploymentName,
                                                  String testId) {
        when:
        "Validate if policy is present"
        def policies = getPolicies().stream()
                .filter { f -> f.getName() == policyName }
                .collect(Collectors.toList())

        assert policies.size() == 1

        and:
        "Policy is temporarily enabled"
        def policy = policies.get(0)
        def policyEnabled = false
        if (policy.disabled) {
            // Use patchPolicy instead of Services.setPolicyDisabled since this already has a reference to policy id.
            // No need to find policy id and refetch it this way.
            PolicyService.patchPolicy(
                    PolicyServiceOuterClass.PatchPolicyRequest.newBuilder().setId(policy.id).setDisabled(false).build()
            )
            log.info "Temporarily enabled policy '${policyName}'"
            policyEnabled = true
        }
        //TODO ROX-11612 debugging to see if the test fails due to incomplete scan
        if (policyName == "Apache Struts: CVE-2017-5638") {
            def image = ImageService.scanImage(STRUTS_DEPLOYMENT.getImage(), true)
            if (!hasApacheStrutsVuln(image)) {
                log.warn("[ROX-11612] CVE-2017-5638 is absent from image scan")
            }
        }

        then:
        "Verify Violation for #policyName is triggered"
        // Some of these policies require scans so extend the timeout as the scan will be done inline
        // with our scanner
        assert waitForViolation(deploymentName,  policyName, WAIT_FOR_VIOLATION_TIMEOUT)

        cleanup:
        if (policyEnabled) {
            PolicyService.patchPolicy(
                    PolicyServiceOuterClass.PatchPolicyRequest.newBuilder().setId(policy.id).setDisabled(true).build()
            )
            log.info "Re-disabled policy '${policyName}'"
        }

        where:
        "Data inputs are:"

        policyName                                      | deploymentName | testId

        "Secure Shell (ssh) Port Exposed"               | NGINX_LATEST   | "C311"

        "Latest tag"                                    | NGINX_LATEST   | ""

        "Environment Variable Contains Secret"          | NGINX_LATEST   | ""

        "Apache Struts: CVE-2017-5638"                  | STRUTS         | "C938"

        "Wget in Image"                                 | WGET_CURL      | "C939"

        "90-Day Image Age"                              | STRUTS         | "C810"

        "Ubuntu Package Manager in Image"               | STRUTS           | "C931"

        //"30-Day Scan Age"                               | SSL_TERMINATOR | "C941"

        "Fixable CVSS >= 7"                             | GCR_NGINX      | "C933"

        "Curl in Image"                                 | WGET_CURL      | "C948"
    }

    def hasApacheStrutsVuln(image) {
        def strutsComponent = image?.getScan()?.getComponentsList()?.find { it.name == "struts" }
        if (strutsComponent == null) {
            log.warn("[Apache struts]struts component is absent from image scan")
            return false
        }
        return strutsComponent.getVulnsList().find { it.cve == "CVE-2017-5638" } != null
    }

    @Tag("BAT")
    @Tag("SMOKE")
    def "Verify that Kubernetes Dashboard violation is generated"() {
        given:
        "Orchestrator is K8S"
        Assume.assumeTrue(orchestrator.isKubeDashboardRunning())

        expect:
        "Verify Kubernetes Dashboard violation exists"
        waitForViolation(K8S_DASHBOARD,  "Kubernetes Dashboard Deployed", 30)
    }

    @Tag("BAT")
    @IgnoreIf({ Env.BUILD_TAG == null || !Env.BUILD_TAG.contains("nightly") })
    def "Notifier for StackRox images with fixable vulns"() {
        when:
        "Verify policies are not violated within the stackrox namespace"
        def violations = AlertService.getViolations(
                ListAlertsRequest.newBuilder().setQuery("Namespace:stackrox,Violation State:*").build()
        )
        log.info "${violations.size()} violation(s) were found in the stackrox namespace"
        def unexpectedViolations = violations.findAll {
            def deploymentName = it.deployment.name
            def policyName = it.policy.name

            (!Constants.VIOLATIONS_ALLOWLIST.containsKey(deploymentName) ||
                    !Constants.VIOLATIONS_ALLOWLIST.get(deploymentName).contains(policyName)) &&
                    !Constants.VIOLATIONS_BY_POLICY_ALLOWLIST.contains(policyName)
        }
        log.info "${unexpectedViolations.size()} violation(s) were not expected"
        if (unexpectedViolations.isEmpty()) {
            return
        }

        String slackPayload = ":rotating_light: " +
                "Fixable Vulnerabilities found in StackRox Images (build tag: ${Env.BUILD_TAG})! " +
                ":rotating_light:"

        Map<String, Set<String>> deploymentPolicyMap = [:]
        Map<String, Set<String>> resourcePolicyMap = [:]
        Map<String, Set<String>> imageFixableVulnMap = [:]
        Boolean hadGetErrors = false
        unexpectedViolations.each {
            if (it.hasResource()) {
                def key = "${it.commonEntityInfo.resourceType}/${it.resource.name}".toString()
                if (!resourcePolicyMap.containsKey(key)) {
                    resourcePolicyMap.put(key, [] as Set)
                }
                resourcePolicyMap.get(key).add(it.policy.name)
                // Alerts with resources don't have images and thus no image vulns so no point in continuing
                // But it's not entirely being ignored so that we can catch issues with unexpected resource alerts
                return
            }

            if (!deploymentPolicyMap.containsKey(it.deployment.name)) {
                deploymentPolicyMap.put(it.deployment.name, [] as Set)
            }
            deploymentPolicyMap.get(it.deployment.name).add(it.policy.name)

            DeploymentOuterClass.Deployment dep
            try {
                dep = DeploymentService.getDeployment(it.deployment.id)
            }
            catch (Exception e) {
                hadGetErrors = true
                log.info "Could not get the deployment with id ${it.deployment.id}, name ${it.deployment.name}: ${e}"
                return it
            }

            dep.containersList.each {
                ImageOuterClass.Image image = ImageService.getImage(it.image.id)
                Set<String> fixables = []
                image.scan.componentsList*.vulnsList*.each {
                    if (it.fixedBy != null && it.fixedBy != "") {
                        fixables.add(it.cve)
                    }
                }
                if (!fixables.isEmpty()) {
                    imageFixableVulnMap.containsKey(image.name.fullName) ?
                            imageFixableVulnMap.get(image.name.fullName).addAll(fixables) :
                            imageFixableVulnMap.putIfAbsent(image.name.fullName, fixables)
                }
            }
        }
        if (imageFixableVulnMap.isEmpty()) {
            assert !hadGetErrors
            log.info "There are no fixable vulns to report"
            return
        }

        if (!deploymentPolicyMap.isEmpty()) {
            slackPayload += "\nDeployments and violated policies: "
            deploymentPolicyMap.each { k, v ->
                slackPayload += "${k}: ${v}  "
            }
        }
        if (!resourcePolicyMap.isEmpty()) {
            slackPayload += " \nResources and violated policies: "
            resourcePolicyMap.each { k, v ->
                slackPayload += "${k}: ${v}  "
            }
        }

        imageFixableVulnMap.each { k, v ->
            slackPayload += "\n${k}: ${v} ${team(k)}"
        }
        SlackUtil.sendMessage(slackPayload)

        imageFixableVulnMap.keySet().collect().each { imageFullName ->
            Helpers.collectImageScanForDebug(
                    imageFullName, imageFullName.replaceAll("\\W", "-")+".json"
            )
        }

        then:
        assert !hadGetErrors
    }

    String team(String img) {
        // To notify slack group we need to provide its ID.
        // It can be found with https://app.slack.com/client/T030RBGDB/browse-user-groups/user_groups
        // To validate it's correct you can use: https://app.slack.com/block-kit-builder
        if (img.contains('scanner')) {
            return '<!subteam^S0499T54CAC>'
        }
        if (img.contains('collector')) {
            return '<!subteam^S01HCU3RQ0H>'
        }
        return img.contains('roxctl') ? '<!subteam^S02KY64PK8U>' : '<!subteam^STZRGPQ78>'
    }

    @Unroll
    @Tag("BAT")
    def "Verify risk factors on struts deployment: #riskFactor"() {
        given:
        "Check Feature Flags"
        featureDependencies.each {
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
        log.info "Risk Factor found in ${System.currentTimeMillis() - start}ms: ${riskFactor}"
        riskResult.score <= maxScore
        riskResult.score >= 1.0f

        message == null ?: riskResult.factorsList.get(0).message == message
        regex == null ?: riskResult.factorsList.get(0).message.matches(regex)

        where:
        "data inputs"

        riskFactor                        | maxScore | message   | regex | featureDependencies
        "Policy Violations"               | 4.0f     | null      | null | []

        "Service Reachability"            | 2.0f     |
                "Port 80 is exposed in the cluster"  | null | []

        "Image Vulnerabilities"           | 4.0f     | null |
                // This makes sure it has at least 100 CVEs.
                "Image \"" + STRUTS_IMAGE + "\\\"" +
                     " contains \\d{3,} CVEs with severities ranging between " +
                     "Low and Critical" | []

        "Service Configuration"           | 2.0f     |
                "No capabilities were dropped" | null | []

        "Components Useful for Attackers" | 1.5f     |
                "Image \"" + STRUTS_IMAGE + "\"" +
                " contains components useful for attackers:" +
                    COMPONENTS | null | []

        "Number of Components in Image"   | 1.5f     | null |
                "Image \"" + STRUTS_IMAGE + "\\\"" +
                " contains " + componentCount + " components" | []

        "Image Freshness"                 | 1.5f     | null | null | []
        // TODO(ROX-9637)
//         "RBAC Configuration"              | 1.0f     |
//                 "Deployment is configured to automatically mount a token for service account \"default\"" | null |
//                 []
    }

    @Tag("BAT")
    def "Verify that built-in services don't trigger unexpected alerts"() {
        expect:
        "Verify unexpected policies are not violated within the kube-system namespace"
        List<ListAlert> kubeSystemViolations = AlertService.getViolations(
          ListAlertsRequest.newBuilder()
            .setQuery("Namespace:kube-system+Policy:!Kubernetes Dashboard").build()
        )
        List<ListAlert> nonWhitelistedKubeSystemViolations = kubeSystemViolations.stream()
             .filter { x -> !WHITELISTED_KUBE_SYSTEM_POLICIES.contains(x.policy.name) }
             .filter { x -> !WHITELISTED_KUBE_SYSTEM_DEPLOYMENTS_AND_POLICIES.contains(
                 x.deployment.name + ' - ' + x.policy.name) }
             .filter { x -> ignoreAlertsForDeletedPolicies(x) }
             .filter { x -> ignoreAlertsByAdminWithKubectl(x) }
             .collect()

        if (nonWhitelistedKubeSystemViolations.size() != 0) {
            nonWhitelistedKubeSystemViolations.forEach {
                violation ->
                log.info "An unexpected kube-system violation:"
                log.info violation.toString()
                log.info "The policy details:"
                log.info Services.getPolicy(violation.policy.id).toString()
                log.debug "The attribute list:"
                AlertService.getViolation(violation.id).getViolationsList().forEach { v ->
                    v.getKeyValueAttrs().getAttrsList().forEach { a ->
                        log.debug "\t${a.getKey()}: ${a.getValue()}"
                    }
                }
            }
        }

        nonWhitelistedKubeSystemViolations.size() == 0
    }

    // The OpenShift: Kubeadmin Secret Accessed policy can sometimes
    // get triggered by the CI. This happens when the CI scripts use
    // kubectl to pull resources to save on an earlier test failure.
    // Ignore this alert iff _all_ violations was by admin,
    // kube:admin or system:admin using kubectl. Do not ignore for
    // any other violations. See
    // https://issues.redhat.com/browse/ROX-10018
    private boolean ignoreAlertsByAdminWithKubectl(ListAlert alert) {
        if (alert.policy.getName() != "OpenShift: Kubeadmin Secret Accessed") {
            return true
        }
        return !AlertService.getViolation(alert.id).getViolationsList().
                stream().allMatch { v ->
            def user = v.getKeyValueAttrs().getAttrsList().find { a ->
                a.getKey() == "Username" && (a.getValue() == "admin" ||
                        a.getValue() =~ /(kube|system)\:admin/)
            }
            def ua = v.getKeyValueAttrs().getAttrsList().find { a ->
                a.getKey() == "User Agent" && a.getValue().startsWith("kubectl/")
            }
            user != null && ua != null
        }
    }

    // ROX-5350 - Ignore alerts for deleted policies
    def ignoreAlertsForDeletedPolicies(ListAlert violation) {
        try {
            Services.getPolicy(violation.policy.id)
            return true
        } catch (StatusRuntimeException e) {
            log.info("Cannot get the policy associated with the alert ${violation}", e)
        }
        return false
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

    @Tag("BAT")
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

    @Tag("BAT")
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
