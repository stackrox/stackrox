import static util.Helpers.waitForTrue
import static util.Helpers.withRetry
import static util.Helpers.trueWithin

import groovy.json.JsonOutput

import io.stackrox.proto.storage.SignatureIntegrationOuterClass.SignatureIntegration

import common.Constants
import services.BaseService
import services.SignatureIntegrationService

import spock.lang.Tag

@Tag("Integration")
class RedHatSigningKeyTest extends BaseSpecification {

    private static final String RED_HAT_INTEGRATION_ID =
            "io.stackrox.signatureintegration.12a37a37-760e-4388-9e79-d62726c075b2"

    private static final String TEST_PUBLIC_KEY_PEM = """\
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE16IoQbiiB5exTRLTkl2rn5FuyXys
4TbDn4+GhQD1JmLZnAiA0cXktX+gFdxu/0JM9pcjjaqT7pdXztbBs78cXg==
-----END PUBLIC KEY-----
"""

    private static final String TEST_PUBLIC_KEY_PEM_2 = """\
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEQq1X/6XxCA4s0++8Tvl8k+Z0G/GN
LKpdYJEldXnyRE4ppY5d7vnRZHvdZQMSE3KoRSMvVnzZtc9LTKLB3DlS/w==
-----END PUBLIC KEY-----
"""

    private static final String WATCH_INTERVAL_ENV = "ROX_REDHAT_SIGNING_KEY_WATCH_INTERVAL"
    private static final String SHORT_WATCH_INTERVAL = "10s"

    private static SignatureIntegration getRedHatIntegration() {
        def resp = SignatureIntegrationService.getSignatureIntegrationClient()
                .listSignatureIntegrations(BaseService.EMPTY)
        return resp.integrationsList.find { it.id == RED_HAT_INTEGRATION_ID }
    }

    @Tag("BAT")
    def "Red Hat signature integration exists with at least one key"() {
        when:
        "Fetching the Red Hat signature integration"
        SignatureIntegration integration = null
        withRetry(10, 3) {
            integration = getRedHatIntegration()
            assert integration != null
        }

        then:
        "The integration has the expected name and at least one cosign public key"
        integration.name == "Red Hat"
        integration.cosign.publicKeysCount >= 1
        integration.cosign.publicKeysList.every { it.name && it.publicKeyPemEnc }
    }

    @Tag("BAT")
    def "Watcher picks up key bundle file written to Central pod"() {
        given:
        "Central is configured with a short watch interval for testing"
        orchestrator.updateDeploymentEnv(Constants.STACKROX_NAMESPACE, "central", "central",
                WATCH_INTERVAL_ENV, SHORT_WATCH_INTERVAL)

        def bundleJson = JsonOutput.toJson([keys: [
                [name: "test-key-1", pem: TEST_PUBLIC_KEY_PEM],
                [name: "test-key-2", pem: TEST_PUBLIC_KEY_PEM_2],
        ]])

        // Wait for the new pod (with the updated env var) to be Running and ready.
        // Checking only phase == "Running" can match the old pod before the rollout begins.
        String centralPodName = null
        waitForTrue(10, 15) {
            def pods = orchestrator.getPods(Constants.STACKROX_NAMESPACE, "central")
            if (pods.size() != 1) { return false }
            def pod = pods.get(0)
            boolean hasNewEnv = pod.spec.containers.get(0).env.any {
                it.name == WATCH_INTERVAL_ENV && it.value == SHORT_WATCH_INTERVAL
            }
            boolean isReady = pod.status.containerStatuses?.getAt(0)?.ready ?: false
            if (hasNewEnv && isReady) {
                centralPodName = pod.metadata.name
                return true
            }
            return false
        }
        assert centralPodName != null

        when:
        "The bundle file is written into the Central pod at the watcher path"
        def b64 = bundleJson.bytes.encodeBase64().toString()
        def writeCmd = ["sh", "-c",
                "mkdir -p /tmp/redhat-signing-keys && echo ${b64} | base64 -d > /tmp/redhat-signing-keys/bundle.json",
        ] as String[]
        assert orchestrator.execInContainerByPodName(
                centralPodName, Constants.STACKROX_NAMESPACE, writeCmd)

        then:
        "The watcher detects the file and upserts the integration with the bundle keys"
        trueWithin(12, 5) {
            try {
                def integration = getRedHatIntegration()
                if (integration == null) {
                    return false
                }
                if (integration.cosign.publicKeysCount != 2) {
                    return false
                }
                def keyNames = (integration.cosign.publicKeysList*.name).sort()
                return keyNames == ["test-key-1", "test-key-2"]
            } catch (io.grpc.StatusRuntimeException ignored) {
                // Central may be briefly unavailable after the rolling restart triggered
                // by the env var update in given:; gRPC will reconnect on the next retry.
                return false
            }
        }

        cleanup:
        "Remove the test bundle file and the watch interval env var"
        if (centralPodName) {
            orchestrator.execInContainerByPodName(
                    centralPodName, Constants.STACKROX_NAMESPACE,
                    ["sh", "-c", "rm -f /tmp/redhat-signing-keys/bundle.json"] as String[])
        }
        orchestrator.removeDeploymentEnv(Constants.STACKROX_NAMESPACE, "central", "central",
                WATCH_INTERVAL_ENV)
        waitForTrue(6, 10) {
            orchestrator.deploymentReady(Constants.STACKROX_NAMESPACE, "central")
        }
    }
}
