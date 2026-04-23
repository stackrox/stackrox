import static util.Helpers.withRetry
import static util.Helpers.trueWithin

import io.stackrox.proto.storage.SignatureIntegrationOuterClass.SignatureIntegration

import objects.Deployment
import common.Constants
import services.BaseService
import services.SignatureIntegrationService

import spock.lang.Tag

@Tag("Integration")
class RedHatSigningKeyTest extends BaseSpecification {

    private static final String RED_HAT_INTEGRATION_ID =
            "io.stackrox.signatureintegration.12a37a37-760e-4388-9e79-d62726c075b2"

    private static final String CENTRAL_CONTAINER_NAME = "central"

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
        "A key bundle JSON with two test keys"
        def bundleJson = """{"keys": [""" +
                """{"name": "test-key-1", "pem": ${escapeJsonString(TEST_PUBLIC_KEY_PEM)}}, """ +
                """{"name": "test-key-2", "pem": ${escapeJsonString(TEST_PUBLIC_KEY_PEM_2)}}]}"""

        def centralPods = orchestrator.getPods(Constants.STACKROX_NAMESPACE, "central")
        assert centralPods.size() > 0
        def centralPodName = centralPods.get(0).metadata.name

        when:
        "The bundle file is written into the Central pod at the watcher path"
        def writeCmd = ["sh", "-c",
                "mkdir -p /tmp/redhat-signing-keys && " +
                "cat > /tmp/redhat-signing-keys/bundle.json << 'BUNDLE_EOF'\n${bundleJson}\nBUNDLE_EOF",
        ] as String[]
        assert orchestrator.execInContainerByPodName(
                centralPodName, Constants.STACKROX_NAMESPACE, writeCmd)

        then:
        "The watcher detects the file and upserts the integration with the bundle keys"
        trueWithin(30, 5) {
            def integration = getRedHatIntegration()
            if (integration == null) {
                return false
            }
            if (integration.cosign.publicKeysCount != 2) {
                return false
            }
            def keyNames = (integration.cosign.publicKeysList*.name).sort()
            return keyNames == ["test-key-1", "test-key-2"]
        }

        cleanup:
        "Remove the test bundle file so subsequent test runs start clean"
        orchestrator.execInContainerByPodName(
                centralPodName, Constants.STACKROX_NAMESPACE,
                ["sh", "-c", "rm -f /tmp/redhat-signing-keys/bundle.json"] as String[])
    }

    @Tag("BAT")
    def "Updater downloads key bundle from remote HTTP server"() {
        given:
        "An nginx pod serving a key bundle JSON in the stackrox namespace"
        def bundleJson = """{"keys": [""" +
                """{"name": "updater-key-1", "pem": ${escapeJsonString(TEST_PUBLIC_KEY_PEM)}}, """ +
                """{"name": "updater-key-2", "pem": ${escapeJsonString(TEST_PUBLIC_KEY_PEM_2)}}]}"""

        def configMapName = "rh-signing-key-bundle-test"
        orchestrator.createConfigMap(configMapName,
                ["bundle.json": bundleJson],
                Constants.STACKROX_NAMESPACE)

        def nginxDeployment = new Deployment()
                .setName("key-bundle-server")
                .setNamespace(Constants.STACKROX_NAMESPACE)
                .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-19-alpine")
                .addLabel("app", "key-bundle-server")
                .addPort(80, "TCP")
                .setExposeAsService(true)
        nginxDeployment.addVolumeFromConfigMap(
                new objects.ConfigMap(name: configMapName, namespace: Constants.STACKROX_NAMESPACE),
                "/usr/share/nginx/html",
        )
        orchestrator.createDeployment(nginxDeployment)

        def serviceIP = orchestrator.getServiceIP("key-bundle-server", Constants.STACKROX_NAMESPACE)
        def bundleURL = "http://${serviceIP}/bundle.json"

        when:
        "Central is configured with the bundle URL and a short update interval"
        orchestrator.updateDeploymentEnv(
                Constants.STACKROX_NAMESPACE, "central", CENTRAL_CONTAINER_NAME,
                "ROX_REDHAT_SIGNING_KEY_BUNDLE_URL", bundleURL,
        )
        orchestrator.updateDeploymentEnv(
                Constants.STACKROX_NAMESPACE, "central", CENTRAL_CONTAINER_NAME,
                "ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL", "10s",
        )

        then:
        "Central restarts and the updater downloads the bundle, watcher upserts the keys"
        // Rolling update + updater download + watcher poll: allow up to 5 minutes.
        trueWithin(60, 5) {
            try {
                def integration = getRedHatIntegration()
                if (integration == null) {
                    return false
                }
                if (integration.cosign.publicKeysCount != 2) {
                    return false
                }
                def keyNames = (integration.cosign.publicKeysList*.name).sort()
                return keyNames == ["updater-key-1", "updater-key-2"]
            } catch (Exception ignored) {
                // Central may be restarting — gRPC calls can fail temporarily.
                return false
            }
        }

        cleanup:
        "Remove the env vars from Central and delete the nginx server"
        orchestrator.removeDeploymentEnv(
                Constants.STACKROX_NAMESPACE, "central", CENTRAL_CONTAINER_NAME,
                "ROX_REDHAT_SIGNING_KEY_BUNDLE_URL",
        )
        orchestrator.removeDeploymentEnv(
                Constants.STACKROX_NAMESPACE, "central", CENTRAL_CONTAINER_NAME,
                "ROX_REDHAT_SIGNING_KEY_UPDATE_INTERVAL",
        )
        // Wait for Central to become ready again after the rolling update.
        trueWithin(60, 5) {
            orchestrator.deploymentReady(Constants.STACKROX_NAMESPACE, "central")
        }
        orchestrator.deleteDeployment(nginxDeployment)
        orchestrator.deleteConfigMap(configMapName, Constants.STACKROX_NAMESPACE)
    }

    private static String escapeJsonString(String s) {
        def escaped = s.replace("\\", "\\\\")
                .replace("\"", "\\\"")
                .replace("\n", "\\n")
                .replace("\r", "\\r")
                .replace("\t", "\\t")
        return "\"${escaped}\""
    }
}
