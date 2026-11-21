import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import services.ImageService
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

        when: "sample nginx-2.0.3 ${ITERATIONS} times over ~30 minutes"
        // Two probes per iteration:
        // 1. Forced build-time scan (the exact call the failing ImageManagementTest
        //    case makes) - exercises the real failing assertion.
        // 2. Unforced scanImage (cached/scan view) - records max fixable CVSS, to
        //    distinguish incomplete enrichment (scores zero) from policy-eval
        //    failure (scores present, alert absent) and cache divergence.
        List<String> failures = []
        List<String> samples = []
        long startMs = System.currentTimeMillis()
        for (int i = 1; i <= ITERATIONS; i++) {
            def scanResults = Services.requestBuildImageScan("quay.io", "rhacs-eng/qa-multi-arch", "nginx-2.0.3")
            def alerts = scanResults.getAlertsList().findAll { it.getPolicy().name == clone.name }

            def image = ImageService.scanImage("quay.io/rhacs-eng/qa-multi-arch:nginx-2.0.3", true, false)
            def vulns = image?.scan?.componentsList?.collectMany { it.vulnsList } ?: []
            def fixable = vulns.findAll { it.fixedBy }
            def maxFixableCvss = (fixable*.cvss).max() ?: 0

            long elapsed = (System.currentTimeMillis() - startMs) / 1000
            samples << "iter=${i}:t=${elapsed}s:alerts=${alerts.size()}:maxFixableCvss=${maxFixableCvss}:fixableCves=${fixable.size()}"
            if (alerts.size() != 1) {
                failures << "iteration ${i} (t=${elapsed}s): ${alerts.size()} alerts for cloned policy (expected 1), maxFixableCvss=${maxFixableCvss}, fixableCves=${fixable.size()}"
            }
            if (i < ITERATIONS) {
                sleep(DELAY_MS)
            }
        }

        then: "all iterations should find exactly 1 alert"
        // Always log the full sample series - the enrichment-over-time curve is
        // the diagnostic payload, not just pass/fail.
        log.warn("Fixable CVSS stress test samples:\n  ${samples.join('\n  ')}")
        assert failures.isEmpty() : "Failed ${failures.size()}/${ITERATIONS} iterations:\n  ${failures.join('\n  ')}"

        cleanup:
        if (clone) {
            PolicyService.deletePolicy(clone.id)
        }
    }
}
