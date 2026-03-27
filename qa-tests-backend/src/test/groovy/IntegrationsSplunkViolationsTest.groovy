import static util.Helpers.withRetry
import static util.SplunkUtil.postToSplunk
import static util.SplunkUtil.tearDownSplunk
import static util.SplunkUtil.waitForSplunkReady

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

import spock.lang.IgnoreIf
import spock.lang.Tag

// ROX-14228 skipping tests for 1st release on power & z
@IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
class IntegrationsSplunkViolationsTest extends BaseSpecification {
    private static final String TEST_NAMESPACE = Constants.SPLUNK_TEST_NAMESPACE
    private static final String SPLUNK_INPUT_NAME = "stackrox-violations-input"
    private static final String SPLUNK_TA_CONVERSION_JOB_NAME =
            "Threat - Create Notable from RHACS Alert - Rule"

    @spock.lang.Shared
    private SplunkDeployment splunkDeployment

    def setupSpec() {
        orchestrator.deleteNamespace(TEST_NAMESPACE)
        orchestrator.ensureNamespaceExists(TEST_NAMESPACE)
        addStackroxImagePullSecret(orchestrator, TEST_NAMESPACE)

        splunkDeployment = SplunkUtil.createSplunk(orchestrator, TEST_NAMESPACE)
        waitForSplunkReady(splunkDeployment.splunkPortForward.getLocalPort())
    }

    def cleanupSpec() {
        if (splunkDeployment) {
            tearDownSplunk(orchestrator, splunkDeployment)
        }
        orchestrator.deleteNamespace(TEST_NAMESPACE)
    }

    private void configureSplunkTA(SplunkUtil.SplunkDeployment splunkDeployment, String centralHost) {
        log.info "Configuring Stackrox TA"
        int port = splunkDeployment.splunkPortForward.getLocalPort()

        def tokenResp = ApiTokenService.generateToken("splunk-token-${splunkDeployment.uid}", "Analyst")
        postToSplunk(port, "/servicesNS/nobody/TA-stackrox/configs/conf-ta_stackrox_settings/additional_parameters",
                ["central_endpoint": "${centralHost}:443",
                 "api_token": tokenResp.getToken(),])
        // create new input to search violations from
        // UCC-based TAs register inputs via custom REST handlers, not data/inputs/
        postToSplunk(port, "/servicesNS/nobody/TA-stackrox/TA_stackrox_stackrox_violations",
                ["name": SPLUNK_INPUT_NAME, "interval": "5", "from_checkpoint": "2000-01-01T00:00:00.000Z",
                 "index": "main",])
    }

    @Tag("Integration")
    def "Verify Splunk violations: StackRox violations reach Splunk TA"() {
        given:
        "Splunk TA is installed and configured, network and process violations triggered"
        String centralHost = orchestrator.getServiceIP("central", "stackrox")

        configureSplunkTA(splunkDeployment, centralHost)
        triggerProcessViolation(splunkDeployment)
        triggerNetworkFlowViolation(splunkDeployment, centralHost)

        when:
        "Search for violations in Splunk"
        // The TA polls Central every 5s (input interval). Violations may not be indexed yet.
        List<Map<String, String>> results = Collections.emptyList()
        boolean hasNetworkViolation = false
        boolean hasProcessViolation = false
        def port = splunkDeployment.splunkPortForward.getLocalPort()
        withRetry(40, 15) {
            def searchId = SplunkUtil.createSearch(port, "search sourcetype=stackrox-violations")
            Response response = SplunkUtil.getSearchResults(port, searchId)
            // We should have at least one violation in the response
            assert response != null
            results = response.getBody().jsonPath().getList("results")
            assert !results.isEmpty()
            hasNetworkViolation = results.any { isNetworkViolation(it) }
            hasProcessViolation = results.any { isProcessViolation(it) }
            log.info "Found ${results.size()} violations in Splunk — " +
                    "Network: ${hasNetworkViolation}, Process: ${hasProcessViolation}"
            assert hasNetworkViolation && hasProcessViolation
        }

        // Check for Alerts
        // The conversion job (savedsearches.conf) is async — dispatch returns immediately
        // while the search runs in the background. We re-dispatch on each retry because:
        //   1. The default search window is -5m which test setup can exceed (overridden to -30m)
        //   2. If the job ran before violations were fully indexed, it produces zero notables
        //   3. Re-dispatching with force_dispatch creates a fresh run each time
        // We query via the Alerts data model (not index=notable directly) because
        // CIM field extractions (app, severity, dest, etc.) are only applied by the data model.
        List<Map<String, String>> alerts = Collections.emptyList()
        boolean hasNetworkAlert = false
        boolean hasProcessAlert = false
        withRetry(40, 15) {
            log.info "Dispatching conversion job to create Splunk alerts from ACS violations"
            postToSplunk(port, "/services/saved/searches/" + SPLUNK_TA_CONVERSION_JOB_NAME + "/dispatch", [
                    "dispatch.now": "true",
                    "force_dispatch": "true",
                    "dispatch.earliest_time": "-30m",
            ])

            // "| from datamodel" is a generating command that ignores the search job's
            // earliest_time, so we pipe to a where clause to filter by time.
            def vSearchId = SplunkUtil.createSearch(port,
                    "| from datamodel Alerts.Alerts | where _time>relative_time(now(),\"-30m\")")
            Response vResponse = SplunkUtil.getSearchResults(port, vSearchId)
            assert vResponse != null
            alerts = vResponse.getBody().jsonPath().getList("results")
            assert !alerts.isEmpty()
            hasNetworkAlert = alerts.any { isNetworkViolation(it) }
            hasProcessAlert = alerts.any { isProcessViolation(it) }
            log.info "Found ${alerts.size()} alerts in Splunk — " +
                    "Network: ${hasNetworkAlert}, Process: ${hasProcessAlert}"
            assert hasNetworkAlert && hasProcessAlert
        }

        then:
        "StackRox violations are in Splunk and have been converted to alerts"
        assert !alerts.isEmpty()
        log.info "Validating CIM mappings for alerts"
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

        // user — when processInfo fields are absent (e.g. K8S_EVENT violations),
        // Splunk's EVAL-user concatenation produces "unknown" rather than null
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

        // dest — CIM data model fills missing dest with "unknown"
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
