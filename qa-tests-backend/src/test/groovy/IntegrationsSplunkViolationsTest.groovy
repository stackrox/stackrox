import static util.SplunkUtil.SPLUNK_ADMIN_PASSWORD
import static util.SplunkUtil.postToSplunk
import static util.SplunkUtil.tearDownSplunk

import java.nio.file.Paths
import java.util.concurrent.TimeUnit

import io.restassured.path.json.JsonPath
import io.restassured.response.Response

import io.stackrox.proto.api.v1.AlertServiceOuterClass

import common.Constants
import objects.Deployment
import services.AlertService
import services.ApiTokenService
import services.NetworkBaselineService
import util.Env
import util.NetworkGraphUtil
import util.SplunkUtil
import util.SplunkUtil.SplunkDeployment
import util.Timer

import org.junit.Rule
import org.junit.rules.Timeout
import spock.lang.Ignore
import spock.lang.IgnoreIf
import spock.lang.Tag

// ROX-14228 skipping tests for 1st release on power & z
@IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
class IntegrationsSplunkViolationsTest extends BaseSpecification {
    @Rule
    @SuppressWarnings(["JUnitPublicProperty"])
    Timeout globalTimeout = new Timeout(1000 + Constants.TEST_FEATURE_TIMEOUT_PAD, TimeUnit.SECONDS)

    private static final String ASSETS_DIR = Paths.get(
            System.getProperty("user.dir"), "artifacts", "splunk-violations-test")
    private static final String PATH_TO_SPLUNK_TA_SPL = Paths.get(ASSETS_DIR,
    "2023-07-10-TA-stackrox-2.0.0.spl")
    // CIM downloaded from https://classic.splunkbase.splunk.com/app/1621/
    private static final String PATH_TO_CIM_TA_TGZ = Paths.get(ASSETS_DIR,
    "splunk-common-information-model-cim_511.tgz")
    private static final String STACKROX_REMOTE_LOCATION = "/tmp/stackrox.spl"
    private static final String CIM_REMOTE_LOCATION = "/tmp/cim.tgz"
    private static final String TEST_NAMESPACE = Constants.SPLUNK_TEST_NAMESPACE
    private static final String SPLUNK_INPUT_NAME = "stackrox-violations-input"

    private SplunkDeployment splunkDeployment

    def setupSpec() {
        orchestrator.deleteNamespace(TEST_NAMESPACE)

        orchestrator.ensureNamespaceExists(TEST_NAMESPACE)
        addStackroxImagePullSecret(TEST_NAMESPACE)
    }

    def cleanupSpec() {
        orchestrator.deleteNamespace(TEST_NAMESPACE)
    }

    def setup() {
        splunkDeployment = SplunkUtil.createSplunk(orchestrator, TEST_NAMESPACE, false)
    }

    def cleanup() {
        if (splunkDeployment) {
            tearDownSplunk(orchestrator, splunkDeployment)
        }
    }

    private void configureSplunkTA(SplunkUtil.SplunkDeployment splunkDeployment, String centralHost) {
        log.info "Starting Splunk TA configuration"
        def podName = orchestrator
                .getPods(TEST_NAMESPACE, splunkDeployment.deployment.getName())
                .get(0)
                .getMetadata()
                .getName()
        int port = splunkDeployment.splunkPortForward.getLocalPort()

        log.info "Copying TA and CIM app files to splunk pod"
        orchestrator.copyFileToPod(PATH_TO_SPLUNK_TA_SPL, TEST_NAMESPACE, podName, STACKROX_REMOTE_LOCATION)
        orchestrator.copyFileToPod(PATH_TO_CIM_TA_TGZ, TEST_NAMESPACE, podName, CIM_REMOTE_LOCATION)
        log.info "Installing TA"
        postToSplunk(port, "/services/apps/local",
                ["name": STACKROX_REMOTE_LOCATION, "filename": "true"])
        log.info "Installing CIM app"
        postToSplunk(port, "/services/apps/local",
                ["name": CIM_REMOTE_LOCATION, "filename": "true"])
        // fix minimum free disk space parameter
        // default value is 5Gb and CircleCI free disk space is less than that
        // that can prevent data from being indexed
        orchestrator.execInContainer(splunkDeployment.deployment,
                "sudo /opt/splunk/bin/splunk set minfreemb 200 -auth admin:${SPLUNK_ADMIN_PASSWORD}"
        )
        // Splunk needs to be restarted after TA installation
        postToSplunk(splunkDeployment.splunkPortForward.getLocalPort(), "/services/server/control/restart", [:])

        log.info("Configuring Stackrox TA")
        def tokenResp = ApiTokenService.generateToken("splunk-token-${splunkDeployment.uid}", "Analyst")
        postToSplunk(port, "/servicesNS/nobody/TA-stackrox/configs/conf-ta_stackrox_settings/additional_parameters",
                ["central_endpoint": "${centralHost}:443",
                 "api_token": tokenResp.getToken(),])
        // create new input to search violations from
        postToSplunk(port, "/servicesNS/nobody/TA-stackrox/data/inputs/stackrox_violations",
                ["name": SPLUNK_INPUT_NAME, "interval": "1", "from_checkpoint": "2000-01-01T00:00:00.000Z"])
    }

    @Tag("BAT")  // Potential FIXME: Turn back to only integration tests.
    def "Verify Splunk violations: StackRox violations reach Splunk TA"() {
        given:
        "Splunk TA is installed and configured, network and process violations triggered"
        String centralHost = orchestrator.getServiceIP("central", "stackrox")

        configureSplunkTA(splunkDeployment, centralHost)
        triggerProcessViolation(splunkDeployment)
        triggerNetworkFlowViolation(splunkDeployment, centralHost)

        when:
        "Search for violations in Splunk"
        // Splunk search for violations is volatile for some reason.
        // We added retries to make this test less flaky.
        // Check for violations first
        List<Map<String, String>> results = Collections.emptyList()
        boolean hasNetworkViolation = false
        boolean hasProcessViolation = false
        def port = splunkDeployment.splunkPortForward.getLocalPort()
        for (int i = 0; i < 15; i++) {
            log.info "Attempt ${i} to get raw violations from Splunk"
            def searchId = SplunkUtil.createSearch(port, "search sourcetype=stackrox-violations")
            TimeUnit.SECONDS.sleep(15)
            Response response = SplunkUtil.getSearchResults(port, searchId)
            // We should have at least one violation in the response
            if (response != null) {
                results = response.getBody().jsonPath().getList("results")
                if (!results.isEmpty()) {
                    for (result in results) {
                        hasNetworkViolation |= isNetworkViolation(result)
                        hasProcessViolation |= isProcessViolation(result)
                    }
                    log.info "Found violations in Splunk: \n${results}" // TODO: Remove debug log
                    if (hasNetworkViolation && hasProcessViolation) {
                        log.info "Success!"
                        break
                    }
                }
            }
        }

        // FIXME: After we know that violations are there, POST to manually run the conversion cronjob
        // OR: Edit the conversion cronjob to run every 20 seconds in setup()

        // Check for Alerts
        List<Map<String, String>> alerts = Collections.emptyList()
        boolean hasNetworkAlert = false
        boolean hasProcessAlert = false
        for (int i = 0; i < 41; i++) { // FIXME: We must try for at least 10 minutes, as the conversion cron runs every 5 minutes. Try calling the search manually.
            log.info "Attempt ${i} to get Alerts from Splunk"
            def searchId = SplunkUtil.createSearch(port, "| from datamodel Alerts.Alerts")
            TimeUnit.SECONDS.sleep(15)
            Response v_response = SplunkUtil.getSearchResults(port, searchId)
            // We should have at least one violation in the response
            if (v_response != null) {
                alerts = v_response.getBody().jsonPath().getList("results")
                if (!alerts.isEmpty()) {
                    for (alert in alerts) {
                        hasNetworkAlert |= isNetworkViolation(alert)
                        hasProcessAlert |= isProcessViolation(alert)
                    }
                    log.info "Found Alerts in Splunk: \n${alerts}"
                    if (hasNetworkAlert && hasProcessAlert) {
                        log.info "Success!"
                        break
                    }
                }
            }
        }

        then:
        "StackRox violations are in Splunk"
        assert !alerts.isEmpty()
        assert hasNetworkAlert
        assert hasProcessAlert
        for (alert in alerts) {
            validateCimMappings(alert)
        }
    }

    private static void validateCimMappings(Map<String, String> result) {
        def originalEvent = new JsonPath(result.get("_raw"))
        Map<String, String> violationInfo = originalEvent.getMap("violationInfo") ?: [:]
        Map<String, String> policyInfo = originalEvent.getMap("policyInfo") ?: [:]
        Map<String, String> processInfo = originalEvent.getMap("processInfo") ?: [:]

        assert result.get("app") == "stackrox"
        assert result.get("type") == "alert"
        verifyRequiredResultKey(result, "id", violationInfo.get("violationId"))
        verifyRequiredResultKey(result, "description", violationInfo.get("violationMessage"))
        verifyRequiredResultKey(result, "signature_id", policyInfo.get("policyName"))
        // Note that policyDescription and signature might be absent, i.e. null
        assert result.get("signature") == policyInfo.get("policyDescription")

        // user
        def processUid = processInfo.get("processUid")
        def processGid = processInfo.get("processGid")
        def expectedUser = processUid == null || processGid == null
                ? "unknown" : processUid + ":" + processGid
        verifyRequiredResultKey(result, "user", expectedUser)

        // severity
        String severity = coalesce(extractNestedString(originalEvent, "policyInfo.policySeverity"), "unknown")
                .replace("UNSET_", "unknown_")
                .replace("_SEVERITY", "")
                .toLowerCase()
        assert result.get("severity") == severity

        // dest_type
        String destType = coalesce(
                extractNestedString(originalEvent, "networkFlowInfo.destination.deploymentType"),
                extractNestedString(originalEvent, "networkFlowInfo.destination.entityType")
        )
        assert result.get("dest_type") == destType

        // src_type
        String srcType = coalesce(
                extractNestedString(originalEvent, "networkFlowInfo.source.deploymentType"),
                extractNestedString(originalEvent, "networkFlowInfo.source.entityType"),
                extractNestedString(originalEvent, "deploymentInfo.deploymentType"),
                extractNestedString(originalEvent, "resourceInfo.resourceType")
        )
        verifyRequiredResultKey(result, "src_type", srcType)

        // dest
        String dest = coalesce(
                extractDestOrSrc(originalEvent, "destination"),
                extractNestedString(originalEvent, "networkFlowInfo.destination.name"),
                "unknown")
        assert result.get("dest") == dest

        // src
        String src = coalesce(
                extractDestOrSrc(originalEvent, "source"),
                extractNestedString(originalEvent, "networkFlowInfo.source.name"),
                extractSourceViaDeploymentInfo(originalEvent),
                extractSourceViaResourceInfo(originalEvent)
        )
        verifyRequiredResultKey(result, "src", src)
    }

    private static void verifyRequiredResultKey(Map<String, String> result, String key, String expectedValue) {
        assert Objects.requireNonNull(result.get(key)) == expectedValue
    }

    private static <T> T coalesce(T... args) {
        for (T arg : args) {
            if (arg != null) {
                return arg
            }
        }
        return null
    }

    @SuppressWarnings(["ReturnNullFromCatchBlock"])
    private static String extractNestedString(JsonPath jsonPath, String path) {
        try {
            return jsonPath.getString(path)
        } catch (IllegalArgumentException ignored) {
            return null
        }
    }

    private static String extractDestOrSrc(JsonPath originalEvent, String prefix) {
        String clusterName = extractNestedString(originalEvent, "deploymentInfo.clusterName")
        if (clusterName == null) {
            return null
        }
        String deploymentNamespace = extractNestedString(originalEvent, "networkFlowInfo.${prefix}.deploymentNamespace")
        if (deploymentNamespace == null) {
            return null
        }
        String deploymentType = extractNestedString(originalEvent, "networkFlowInfo.${prefix}.deploymentType")
        if (deploymentType == null) {
            return null
        }
        String delimiter = deploymentType == "Pod" ? " > " : "/"
        String name = extractNestedString(originalEvent, "networkFlowInfo.${prefix}.name")

        return name == null ? null : "${clusterName}/${deploymentNamespace}${delimiter}${deploymentType}:${name}"
    }

    static String extractSourceViaDeploymentInfo(JsonPath originalEvent) {
        String clusterName = extractNestedString(originalEvent, "deploymentInfo.clusterName")
        if (clusterName == null) {
            return null
        }
        String deploymentNamespace = extractNestedString(originalEvent, "deploymentInfo.deploymentNamespace")
        if (deploymentNamespace == null) {
            return null
        }
        String deploymentType = extractNestedString(originalEvent, "deploymentInfo.deploymentType")
        if (deploymentType == null) {
            return null
        }
        String deploymentName = extractNestedString(originalEvent, "deploymentInfo.deploymentName")
        if (deploymentName == null) {
            return null
        }
        String podId = extractNestedString(originalEvent, "violationInfo.podId")
        String podPart = podId == null ? "" : " > ${podId}"
        String containerName = extractNestedString(originalEvent, "violationInfo.containerName")
        String containerPart = containerName == null ? "" : "/${containerName}"
        String podDescription = deploymentType == "Pod"
            ? " > ${deploymentType}:${deploymentName}"
            : "/${deploymentType}:${deploymentName}${podPart}"
        return "${clusterName}/${deploymentNamespace}${podDescription}${containerPart}"
    }

    @SuppressWarnings(["IfStatementCouldBeTernary"]) // much more readable this way
    static String extractSourceViaResourceInfo(JsonPath originalEvent) {
        String clusterName = extractNestedString(originalEvent, "resourceInfo.clusterName")
        if (clusterName == null) {
            return null
        }
        String namespace = extractNestedString(originalEvent, "resourceInfo.namespace")
        if (namespace == null) {
            return null
        }
        String resourceType = extractNestedString(originalEvent, "resourceInfo.resourceType")
        if (resourceType == null) {
            return null
        }
        String resourceName = extractNestedString(originalEvent, "resourceInfo.name")
        if (resourceName == null) {
            return null
        }

        return "${clusterName}/${namespace}/${resourceType}:${resourceName}"
    }

    def triggerProcessViolation(SplunkUtil.SplunkDeployment splunkDeployment) {
        orchestrator.execInContainer(splunkDeployment.deployment, "curl http://127.0.0.1:10248/ --max-time 2")
        assert waitForAlertWithPolicyId(splunkDeployment.getDeployment().getName(),
                                        "86804b96-e87e-4eae-b56e-1718a8a55763")
    }

    def triggerNetworkFlowViolation(SplunkUtil.SplunkDeployment splunkDeployment, String centralService) {
        final String splunkUid = splunkDeployment.getDeployment().getDeploymentUid()
        final String centralUid = orchestrator.getDeploymentId(new Deployment(name: "central", namespace: "stackrox"))
        final String apiserverIP = orchestrator.getServiceIP("kubernetes", "default")

        assert retryUntilTrue({
            // Trigger https traffic to Central (note that service port 443 translates to pod port 8443) to ensure
            // StackRox would see it and we can later include it in the baseline.
            // Our Splunk TA would generate the same traffic to central but it may not be fully running at this
            // point therefore we help StackRox to see requests from Splunk by just making a call with curl.
            orchestrator.execInContainer(splunkDeployment.deployment,
                    "curl --insecure --head --request GET --max-time 5 https://${centralService}:443")
            return NetworkGraphUtil.checkForEdge(splunkUid, null)
                    .any { it.targetID == centralUid && it.getPort() == 8443 }
        }, 15)

        def baseline = NetworkBaselineService.getNetworkBaseline(splunkUid)
        log.debug("Network baseline before lock call: ${baseline}")

        // Lock the baseline so that any different requests from (and to) Splunk pod would make a violation.
        NetworkBaselineService.lockNetworkBaseline(splunkUid)

        baseline = NetworkBaselineService.getNetworkBaseline(splunkUid)
        log.debug("Network baseline after lock call: ${baseline}")

        // Make anomalous request from Splunk towards Kube API server. This should trigger a network flow violation.
        assert retryUntilTrue({
            orchestrator.execInContainer(splunkDeployment.deployment,
                    "curl --insecure --head --request GET --max-time 5 https://${apiserverIP}:443")
            return NetworkGraphUtil.checkForEdge(splunkUid, null)
                    .any { it.targetID != centralUid && it.getPort() == 443 }
        }, 15)

        assert waitForAlertWithPolicyId(splunkDeployment.getDeployment().getName(),
                "1b74ffdd-8e67-444c-9814-1c23863c8ccb")
    }

    private boolean waitForAlertWithPolicyId(String deploymentName, String policyId) {
        retryUntilTrue({
            AlertService.getViolations(AlertServiceOuterClass.ListAlertsRequest.newBuilder()
                    .setQuery("Namespace:${TEST_NAMESPACE}+Violation State:*+Deployment:${deploymentName}")
                    .build())
                    .asList()
                    .any { a -> a.getPolicy().getId() == policyId }
        }, 10)
    }

    boolean isNetworkViolation(Map<String, String> result) {
        return isViolationOfType(result, "NETWORK_FLOW")
    }

    boolean isProcessViolation(Map<String, String> result) {
        return isViolationOfType(result, "PROCESS_EVENT")
    }

    boolean isViolationOfType(Map<String, String> result, String type) {
        Map<String, String> violationInfo = new JsonPath(result.get("_raw")).getMap("violationInfo") ?: [:]
        return violationInfo.get("violationType") == type
    }

    // returns whether true condition was achieved
    boolean retryUntilTrue(Closure<Boolean> closure, int retries) {
        Timer timer = new Timer(retries, 10)
        while (timer.IsValid()) {
            def result = closure()
            if (result) {
                return true
            }
        }
        return false
    }
}
