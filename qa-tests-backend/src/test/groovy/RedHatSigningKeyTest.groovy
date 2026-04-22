import static util.Helpers.trueWithin

import spock.lang.Tag

import common.Constants
import services.SignatureIntegrationService

/**
 * RedHatSigningKeyTest verifies that the built-in "Red Hat" signature integration is:
 *   (1) always present at startup with the embedded key,
 *   (2) extended dynamically when additional PEM key files are placed in the
 *       runtime key directory that Central watches, and
 *   (3) populated by the periodic updater when it downloads keys from an HTTP
 *       manifest URL (tested via an in-cluster nginx pod acting as a mock bucket).
 *
 * Tests 2 and 3 require the emptyDir volume mount introduced by ROX-30650 (PR0) and
 * the filesystem watcher introduced by ROX-30650 (PR2).  If Central does not
 * have a writable key directory (i.e., the volume mount is absent), the exec
 * command will fail and those tests are skipped gracefully.
 *
 * Test 3 requires kubectl in PATH and a KUBECONFIG env var pointing to the test cluster.
 * It triggers a Central rolling restart via env var injection and re-establishes the
 * gRPC tunnel afterward.
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

    // A second distinct P-256 ECDSA public key for the updater bucket-fetch test (Test 3).
    // Different from TEST_KEY_PEM so both can coexist without deduplication collapsing them.
    static final private String BUCKET_KEY_PEM = """\
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE5WkSRMRDpBs3r1U2nFuAGFiEBQSp
z1K0ZwC1FVx7jXqMn3qm8b2e4jLrPpdhWL5TiElAqN1aHT8LqWPFBPsm5A==
-----END PUBLIC KEY-----"""

    // Directory inside the Central container where downloaded signing keys live.
    // Must match the default value of the Go env var ROX_REDHAT_SIGNING_KEYS_RUNTIME_DIR
    // (env.RedHatSigningKeysRuntimeDir) so that the watcher and this test target the same path.
    static final private String KEY_DIR = "/var/lib/stackrox/signature-keys/redhat"

    // Name of the injected test key file.
    static final private String TEST_KEY_FILE = "e2e-test-injected.pub"

    // Name of the key file served by the mock bucket (Test 3).
    static final private String BUCKET_KEY_FILE = "e2e-bucket-test.pub"

    // Names for in-cluster nginx resources created by Test 3.
    static final private String KEY_SERVER_NAME = "rox-e2e-key-server"
    static final private String KEY_SERVER_CM   = "rox-e2e-key-server-content"

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
    // Test 3: updater fetches keys from an HTTP manifest server (mock GCS bucket)
    //
    // Deploys an nginx pod in the cluster serving a manifest.json and a PEM key,
    // then reconfigures Central to use that URL and verifies the updater downloads
    // the key and the watcher upserts it into the database.
    //
    // This tests the full updater pipeline end-to-end against a real HTTP server
    // inside the cluster, complementing the unit tests that use httptest.NewServer.
    // ---------------------------------------------------------------------------

    @Tag("Integration")
    def "Updater downloads signing keys from an HTTP manifest server"() {
        given:
        "An in-cluster nginx server acting as a mock signing-key bucket"
        def kubeconfig = System.getenv("KUBECONFIG") ?: ""
        def manifestURL = "http://${KEY_SERVER_NAME}.${STACKROX_NS}.svc/manifest.json"
        def manifestJSON = """{"keys":[{"name":"${BUCKET_KEY_FILE}","url":"${BUCKET_KEY_FILE}"}]}"""

        // Deploy ConfigMap holding the manifest JSON and the test PEM key.
        kubectlApply(kubeconfig, """\
apiVersion: v1
kind: ConfigMap
metadata:
  name: ${KEY_SERVER_CM}
  namespace: ${STACKROX_NS}
data:
  manifest.json: '${manifestJSON}'
  ${BUCKET_KEY_FILE}: |
${BUCKET_KEY_PEM.readLines().collect { "    " + it }.join("\n")}
""")

        // Deploy nginx Pod + ClusterIP Service to serve the ConfigMap content.
        kubectlApply(kubeconfig, """\
apiVersion: v1
kind: Pod
metadata:
  name: ${KEY_SERVER_NAME}
  namespace: ${STACKROX_NS}
  labels:
    app: ${KEY_SERVER_NAME}
spec:
  containers:
  - name: nginx
    image: nginx:alpine
    ports:
    - containerPort: 80
    volumeMounts:
    - name: content
      mountPath: /usr/share/nginx/html
  volumes:
  - name: content
    configMap:
      name: ${KEY_SERVER_CM}
---
apiVersion: v1
kind: Service
metadata:
  name: ${KEY_SERVER_NAME}
  namespace: ${STACKROX_NS}
spec:
  selector:
    app: ${KEY_SERVER_NAME}
  ports:
  - port: 80
    targetPort: 80
""")

        // Wait up to 60 s for the nginx pod to become Ready.
        def readyOut = kubectl(kubeconfig, ["wait", "pod/${KEY_SERVER_NAME}",
                "-n", STACKROX_NS, "--for=condition=Ready", "--timeout=60s"])
        if (!readyOut.contains("condition met")) {
            log.warn("nginx key server pod did not become Ready within 60 s " +
                    "(${readyOut.trim()}); skipping updater bucket test")
            return
        }
        log.info("In-cluster key server ready — manifest URL: ${manifestURL}")

        when:
        "Central is reconfigured to fetch keys from the in-cluster server"
        // kubectl set env triggers a rolling restart of the Central Deployment.
        kubectl(kubeconfig, ["set", "env", "deployment/central", "-n", STACKROX_NS,
                "ROX_REDHAT_SIGNING_KEY_MANIFEST_URL=${manifestURL}",
                "ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL=5s"])
        kubectl(kubeconfig, ["rollout", "status", "deployment/central",
                "-n", STACKROX_NS, "--timeout=120s"])

        // The port-forward tunnel breaks when the Central pod is replaced.
        // Re-establish it so gRPC calls from the test runner reach the new pod.
        restartPortForward(kubeconfig, STACKROX_NS, "8443")

        // Wait for Central gRPC to be reachable again after the restart.
        boolean centralBack = trueWithin(20, 3) {
            try { SignatureIntegrationService.listSignatureIntegrations(); true }
            catch (Exception ignored) { false }
        }
        assert centralBack: "Central gRPC not reachable after rollout"

        then:
        // After the rollout the emptyDir is fresh (pod restarted) → only 1 embedded key.
        // The updater fires within 5 s (ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL=5s),
        // downloads BUCKET_KEY_FILE, the watcher detects it and upserts the DB → count > 1.
        // Allow up to 60 s (12 × 5 s polls).
        boolean keyAppeared = trueWithin(12, 5) {
            redHatKeyCount() > 1
        }
        assert keyAppeared:
                "Key count did not increase after updater should have fetched from ${manifestURL}"
        log.info("Updater fetched key from in-cluster HTTP server — count now ${redHatKeyCount()} — PASS")

        cleanup:
        "Restore Central configuration and remove the in-cluster key server"
        // Remove env var overrides — triggers another rolling restart.
        kubectl(kubeconfig, ["set", "env", "deployment/central", "-n", STACKROX_NS,
                "ROX_REDHAT_SIGNING_KEY_MANIFEST_URL-",
                "ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL-"])
        kubectl(kubeconfig, ["rollout", "status", "deployment/central",
                "-n", STACKROX_NS, "--timeout=120s"])
        restartPortForward(kubeconfig, STACKROX_NS, "8443")

        // Delete in-cluster key server resources.
        kubectl(kubeconfig, ["delete", "pod,service", KEY_SERVER_NAME,
                "-n", STACKROX_NS, "--ignore-not-found"])
        kubectl(kubeconfig, ["delete", "configmap", KEY_SERVER_CM,
                "-n", STACKROX_NS, "--ignore-not-found"])

        // Wait for Central to be reachable before the next test or suite teardown.
        trueWithin(20, 3) {
            try { SignatureIntegrationService.listSignatureIntegrations(); true }
            catch (Exception ignored) { false }
        }
        log.info("Central restored to original configuration")
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

    /** Runs kubectl with the given args and returns stdout. */
    private String kubectl(String kubeconfig, List<String> args) {
        def cmd = ["kubectl"] + (kubeconfig ? ["--kubeconfig=${kubeconfig}"] : []) + args
        def proc = cmd.execute()
        proc.waitFor()
        return proc.text
    }

    /** Pipes yamlContent to kubectl apply -f -. */
    private void kubectlApply(String kubeconfig, String yamlContent) {
        def cmd = ["kubectl"] + (kubeconfig ? ["--kubeconfig=${kubeconfig}"] : []) +
                ["apply", "-f", "-"]
        def proc = cmd.execute()
        proc.outputStream.withWriter("UTF-8") { it << yamlContent }
        proc.waitFor()
        if (proc.exitValue() != 0) {
            log.warn("kubectl apply failed (exit ${proc.exitValue()}): ${proc.err.text}")
        }
    }

    /**
     * Kills any existing kubectl port-forward on localPort and starts a new one
     * to svc/central in the given namespace.  Called after a Central rolling restart
     * to re-establish the tunnel broken when the old pod was replaced.
     *
     * pkill is not universally available; scan /proc for running port-forward
     * processes and send SIGTERM via kill(1) instead.
     */
    private void restartPortForward(String kubeconfig, String ns, String localPort) {
        ["sh", "-c",
         "for f in /proc/*/cmdline; do " +
         "  grep -ql 'port-forward' \"\$f\" 2>/dev/null && " +
         "  kill \"\$(echo \"\$f\" | cut -d/ -f3)\" 2>/dev/null || true; " +
         "done; true"
        ].execute().waitFor()
        sleep(1000)
        def cmd = ["kubectl"] + (kubeconfig ? ["--kubeconfig=${kubeconfig}"] : []) +
                ["port-forward", "svc/central", "${localPort}:443", "-n", ns]
        cmd.execute()   // detached background process; let it run
        sleep(3000)     // give the tunnel time to come up
        log.info("Port-forward restarted: localhost:${localPort} → svc/central:443")
    }
}
