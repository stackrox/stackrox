import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import services.PolicyService
import spock.lang.Tag

/**
 * Diagnostic test for ROX-36267: runs the "Fixable CVSS >= 7" build-time scan
 * repeatedly against nginx-2.0.3 and reports which iterations failed, to
 * distinguish vuln-bundle loading lag (early iterations fail) from intermittent
 * scanner state (scattered failures). See also ROX-36688 and ROX-39709.
 */
@Tag("Parallel")
@Tag("BAT")
@Tag("PZ")
class FixableCVSSStressTest extends BaseSpecification {

    private static final String TEST_NAMESPACE = "qa-fixable-cvss-stress"
    private static final int ITERATIONS = 30
    private static final long DELAY_MS = 60000

    def "Fixable CVSS >= 7 repeated scan diagnostic"() {
        given: "clone and enable the Fixable CVSS >= 7 policy"
        Policy clone = PolicyService.clonePolicyAndScopeByNamespace("Fixable CVSS >= 7", TEST_NAMESPACE)
        Services.updatePolicyLifecycleStage(clone.name, [LifecycleStage.BUILD, LifecycleStage.DEPLOY])
        Services.setPolicyDisabled(clone.name, false)

        when: "scan nginx-2.0.3 ${ITERATIONS} times, sampling over ~30 minutes"
        // Each forced scan re-matches against the matcher's current vuln DB state;
        // it does not prompt the updater. The iterations are passive samples that
        // chart when CVSS enrichment becomes usable after the readiness gate passed.
        List<String> failures = []
        List<String> vulnCounts = []
        long startMs = System.currentTimeMillis()
        for (int i = 1; i <= ITERATIONS; i++) {
            def scanResults = Services.requestBuildImageScan("quay.io", "rhacs-eng/qa-multi-arch", "nginx-2.0.3")
            def alerts = scanResults.getAlertsList().findAll { it.getPolicy().name == clone.name }
            def vulnCount = scanResults.getAlertsList().size()
            long elapsed = (System.currentTimeMillis() - startMs) / 1000
            vulnCounts << "iter=${i}:t=${elapsed}s:alerts=${alerts.size()},total_alerts=${vulnCount}"
            if (alerts.size() != 1) {
                failures << "iteration ${i}: ${alerts.size()} alerts for cloned policy (expected 1), total alerts: ${vulnCount}"
            }
            if (i < ITERATIONS) {
                sleep(DELAY_MS)
            }
        }

        then: "all iterations should find exactly 1 alert"
        if (!failures.isEmpty()) {
            log.warn("Fixable CVSS stress test results:\n  ${vulnCounts.join('\n  ')}")
        }
        assert failures.isEmpty() : "Failed ${failures.size()}/${ITERATIONS} iterations:\n  ${failures.join('\n  ')}"

        cleanup:
        if (clone) {
            PolicyService.deletePolicy(clone.id)
        }
    }
}
