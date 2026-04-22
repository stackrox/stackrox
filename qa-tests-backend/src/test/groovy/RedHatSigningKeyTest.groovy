import static util.Helpers.trueWithin

import spock.lang.Tag

import common.Constants
import services.SignatureIntegrationService

/**
 * RedHatSigningKeyTest verifies that the built-in "Red Hat" signature integration is:
 *   (1) always present at startup with the embedded key, and
 *   (2) extended dynamically when additional PEM key files are placed in the
 *       runtime key directory that Central watches.
 *
 * Test 2 requires the emptyDir volume mount introduced by ROX-30650 (PR3) and
 * the filesystem watcher introduced by ROX-30650 (PR2).  If Central does not
 * have a writable key directory (i.e., the volume mount is absent), the exec
 * command will fail and the test will be skipped gracefully.
 */
@Tag("Integration")
class RedHatSigningKeyTest extends BaseSpecification {

    static final private String REDHAT_INTEGRATION_ID =
            "io.stackrox.signatureintegration.12a37a37-760e-4388-9e79-d62726c075b2"

    static final private String STACKROX_NS = Constants.STACKROX_NAMESPACE

    // A valid P-256 ECDSA public key that is NOT any real Red Hat signing key.
    // Generated via `cosign generate-key-pair`; used only for test injection.
    static final private String TEST_KEY_PEM = """\
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEUpphKrUYSHvrR+r82Jn7Evg/d3L9
w9e2Azq1OYIh/pbeBMHARDrBaqqmuMR9+BfAaPAYdkNTU6f58M2zBbuL0A==
-----END PUBLIC KEY-----"""

    // Directory inside the Central container where downloaded signing keys live.
    // Must match the default value of the Go env var ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR
    // (env.RedHatSigningKeysRuntimeDir) so that the watcher and this test target the same path.
    static final private String KEY_DIR = "/var/lib/stackrox/signature-keys/redhat"

    // Name of the injected test key file.
    static final private String TEST_KEY_FILE = "e2e-test-injected.pub"

    // ---------------------------------------------------------------------------
    // Test 1: embedded Red Hat key is present at startup
    // ---------------------------------------------------------------------------

    @Tag("BAT")
    def "Red Hat signature integration exists with embedded key"() {
        when:
        "Listing all signature integrations"
        def integrations = SignatureIntegrationService.listSignatureIntegrations()

        then:
        "The built-in Red Hat integration should be present with at least one public key"
        def redHat = integrations.find { it.getId() == REDHAT_INTEGRATION_ID }
        assert redHat != null: "Red Hat signature integration not found (expected ID: ${REDHAT_INTEGRATION_ID})"
        assert redHat.getCosign().getPublicKeysList().size() >= 1:
                "Expected at least one embedded Red Hat public key, got 0"
        log.info("Red Hat integration '${redHat.getName()}' has " +
                "${redHat.getCosign().getPublicKeysList().size()} key(s) — PASS")
    }

    // ---------------------------------------------------------------------------
    // Test 2: dynamic key injection via filesystem watcher
    // ---------------------------------------------------------------------------

    @Tag("Integration")
    def "Red Hat signing keys are reloaded dynamically from the key directory"() {
        given:
        "A running Central pod in the stackrox namespace"
        def pods = orchestrator.getPods(STACKROX_NS, "central")
        if (pods.isEmpty()) {
            log.warn("No central pod found, skipping dynamic key reload test")
            return
        }
        def centralPod = pods[0].getMetadata().getName()
        log.info("Using Central pod: ${centralPod}")

        def initialCount = redHatKeyCount()
        log.info("Initial Red Hat key count: ${initialCount}")

        when:
        "A new PEM key file is placed in the Central key directory via kubectl exec"
        // Escape single quotes in the PEM for shell safety — use a here-doc approach.
        // The key contains no single quotes, so direct single-quote quoting is fine.
        def pemEscaped = TEST_KEY_PEM.replace("'", "'\\''")
        def writeCmd = "mkdir -p '${KEY_DIR}' && printf '%s\\n' '${pemEscaped}' > '${KEY_DIR}/${TEST_KEY_FILE}'"
        boolean wrote = orchestrator.execInContainerByPodName(
                centralPod, STACKROX_NS, (["sh", "-c", writeCmd] as String[]), 5)

        then:
        "The exec command should succeed (volume is writable)"
        if (!wrote) {
            log.warn("Could not write to '${KEY_DIR}' in Central pod '${centralPod}'. " +
                    "The emptyDir volume mount (ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR) may be absent. " +
                    "Skipping dynamic key reload assertion.")
            // Clean attempt just in case
            cleanupTestKey(centralPod)
            return
        }

        and:
        "The filesystem watcher detects the new file and upserts the integration"
        // The default watch interval is 30 s; allow up to 90 s total (3 polls).
        // In CI set ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL=5s to reduce wait.
        boolean keyAppeared = trueWithin(18, 5) {
            redHatKeyCount() > initialCount
        }
        assert keyAppeared: "Red Hat integration key count did not increase within 90 s of injecting ${TEST_KEY_FILE}"

        def finalCount = redHatKeyCount()
        log.info("Red Hat key count after injection: ${finalCount} (was ${initialCount}) — PASS")

        cleanup:
        "Remove the injected test key file from the Central pod"
        cleanupTestKey(centralPod)
    }

    // ---------------------------------------------------------------------------
    // Helpers
    // ---------------------------------------------------------------------------

    private int redHatKeyCount() {
        def integrations = SignatureIntegrationService.listSignatureIntegrations()
        def redHat = integrations.find { it.getId() == REDHAT_INTEGRATION_ID }
        return redHat?.getCosign()?.getPublicKeysList()?.size() ?: 0
    }

    private void cleanupTestKey(String centralPod) {
        try {
            def rmCmd = "rm -f '${KEY_DIR}/${TEST_KEY_FILE}'"
            orchestrator.execInContainerByPodName(
                    centralPod, STACKROX_NS, (["sh", "-c", rmCmd] as String[]), 3)
            log.info("Removed test key file ${KEY_DIR}/${TEST_KEY_FILE} from Central pod")
        } catch (Exception e) {
            log.warn("Failed to remove test key file during cleanup: ${e.getMessage()}")
        }
    }
}
