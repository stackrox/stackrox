import static util.Helpers.withRetry

import java.nio.charset.Charset
import java.security.cert.X509Certificate

import com.google.gson.JsonObject
import com.google.gson.JsonPrimitive
import io.fabric8.kubernetes.api.model.Secret
import io.grpc.StatusRuntimeException
import org.bouncycastle.asn1.x500.X500Name
import org.bouncycastle.asn1.x500.style.BCStyle
import org.bouncycastle.asn1.x500.style.IETFUtils
import org.yaml.snakeyaml.Yaml

import io.stackrox.proto.storage.ClusterOuterClass.ClusterUpgradeStatus.UpgradeProcessStatus.UpgradeProcessType
import io.stackrox.proto.storage.ClusterOuterClass.UpgradeProgress.UpgradeState

import services.ClusterService
import services.DirectHTTPService
import services.SensorUpgradeService
import util.Cert
import util.Env

import org.junit.Assume
import spock.lang.IgnoreIf
import spock.lang.Tag

@Tag("BAT")
@Tag("PZ")
// skip if executed in a test environment with just secured-cluster deployed in the test cluster
// i.e. central is deployed elsewhere
@IgnoreIf({ Env.ONLY_SECURED_CLUSTER == "true" })
class CertRotationTest extends BaseSpecification {

    def generateCerts(String path, String expectedFileName, JsonObject data = null) {
        def resp = DirectHTTPService.post(path, data)
        assert resp.getResponseCode() == 200
        assert resp.getHeaderField("Content-Disposition") == "attachment; filename=\"${expectedFileName}\""
        def regeneratedCentralTLSContents = resp.getInputStream().getText()
        assert regeneratedCentralTLSContents

        def regeneratedCentralTLSYAML = new Yaml()
        return regeneratedCentralTLSYAML.loadAll(regeneratedCentralTLSContents).toList()
    }

    def verifyCertProperties(String certBase64, String principalShouldContain, Long timestampBeforeGeneration) {
        def cert = Cert.loadBase64EncodedCert(certBase64)
        assert cert.subjectX500Principal.getName().contains(principalShouldContain)
        // Subtract one day to allow for DST/leap years etc.
        def oneYearAfterStart = new Date(timestampBeforeGeneration).toInstant().plusSeconds(364 * 24 * 60 * 60)
        assert cert.notAfter.after(Date.from(oneYearAfterStart))
        return true
    }

    def assertSameKeysAndSameValuesExcept(Map<String, Object> current, Map<String, Object> regenerated,
                                          Set<String> keysWithDifferentValues) {
        assert current.keySet() == regenerated.keySet()
        for (k in current.keySet()) {
            if (!keysWithDifferentValues.contains(k)) {
                // The regenerated file may contain a trailing newline, which we ignore for the purposes of
                // this comparison since it has no functional impact.
                assert current[k] == ((String)regenerated[k]).trim()
            }
        }
        return true
    }

    def testMatchingSecretFoundWithExpectedProperties(
        List<Object> regeneratedSecrets, Secret currentSecret, String certFileName, String keyFileName,
        String principalShouldContain, Long timestampBeforeGeneration
    ){
        def regeneratedSecretObj = regeneratedSecrets.
            find { it["metadata"]["name"] == currentSecret.metadata.name }
        assert regeneratedSecretObj
        Map<String, String> regeneratedSecretData =
                (Map<String, String>) regeneratedSecretObj["data"] ?:
                ((Map<String, String>) regeneratedSecretObj["stringData"])?.collectEntries { key, value ->
            [(key): Base64.encoder.encodeToString(value.getBytes(Charset.defaultCharset()))]
        }
        assert regeneratedSecretData
        assertSameKeysAndSameValuesExcept(currentSecret.data, regeneratedSecretData,
            [certFileName, keyFileName] as Set<String>)
        verifyCertProperties((String)regeneratedSecretData[certFileName], principalShouldContain,
            timestampBeforeGeneration)
    }

    def "Test Central cert rotation"() {
        when:
        "Fetch the current central-tls secret, and regenerate new certs"
        def centralTLSSecret = orchestrator.getSecret("central-tls", "stackrox")
        assert centralTLSSecret
        def start = System.currentTimeMillis()
        def regeneratedSecrets = generateCerts("api/extensions/certgen/central", "central-tls.yaml")

        then:
        "Validate contents of the cert"
        testMatchingSecretFoundWithExpectedProperties(regeneratedSecrets, centralTLSSecret,
            "cert.pem", "key.pem", "CENTRAL_SERVICE", start)
    }

    def "Test Scanner cert rotation"() {
        when:
        "Fetch the current scanner-tls and scanner-db-tls secrets, and regenerate new certs"
        def scannerTLSSecret = orchestrator.getSecret("scanner-tls", "stackrox")
        assert scannerTLSSecret
        def scannerDBTLSSecret = orchestrator.getSecret("scanner-db-tls", "stackrox")
        assert scannerDBTLSSecret
        def start = System.currentTimeMillis()
        def regeneratedSecrets = generateCerts("api/extensions/certgen/scanner", "scanner-tls.yaml")

        then:
        testMatchingSecretFoundWithExpectedProperties(regeneratedSecrets, scannerTLSSecret,
            "cert.pem", "key.pem", "SCANNER_SERVICE", start)
        testMatchingSecretFoundWithExpectedProperties(regeneratedSecrets, scannerDBTLSSecret,
            "cert.pem", "key.pem", "SCANNER_DB_SERVICE", start)
    }

    def "Test sensor cert rotation"() {
        when:
        "Fetch the current sensor-tls, collector-tls and admission-control-tls secrets, and regenerate certs"
        def sensorTLSSecret = orchestrator.getSecret("sensor-tls", "stackrox")
        assert sensorTLSSecret
        def collectorTLSSecret = orchestrator.getSecret("collector-tls", "stackrox")
        assert collectorTLSSecret
        def admissionControlTLSSecret = orchestrator.getSecret("admission-control-tls", "stackrox")
        // Admission control secret may or may not be present, depending on how the cluster was deployed.
        def admissionControlSecretPresent = (admissionControlTLSSecret != null)
        log.info "Admission control secret present: ${admissionControlSecretPresent}"
        def cluster = ClusterService.getCluster()
        assert cluster
        def reqObject = new JsonObject()
        reqObject.add("id", new JsonPrimitive(cluster.getId()))
        def start = System.currentTimeMillis()
        def regeneratedSecrets = generateCerts("api/extensions/certgen/cluster",
            "cluster-${cluster.getName()}-tls.yaml", reqObject)
        assert regeneratedSecrets.size() == 2 + (admissionControlSecretPresent ? 1 : 0)

        then:
        assert cluster.getAdmissionController() == admissionControlSecretPresent
        testMatchingSecretFoundWithExpectedProperties(regeneratedSecrets, sensorTLSSecret,
            "sensor-cert.pem", "sensor-key.pem", "SENSOR_SERVICE: ${cluster.getId()}", start)
        testMatchingSecretFoundWithExpectedProperties(regeneratedSecrets, collectorTLSSecret,
            "collector-cert.pem", "collector-key.pem", "COLLECTOR_SERVICE: ${cluster.getId()}", start)
        if (admissionControlSecretPresent) {
            testMatchingSecretFoundWithExpectedProperties(regeneratedSecrets, admissionControlTLSSecret,
                "admission-control-cert.pem", "admission-control-key.pem",
                "ADMISSION_CONTROL_SERVICE: ${cluster.getId()}", start)
        }
    }

    String extractCNFromCert(X509Certificate cert) {
        return IETFUtils.valueToString(
                X500Name.getInstance(cert.subjectX500Principal.encoded).getRDNs(BCStyle.CN)[0].first.value)
    }

    def checkCurrentValueOfSecretIdenticalExceptNewCerts(String expectedCertCN, Secret previousSecret, String name) {
        def currentSecret = orchestrator.getSecret(name, "stackrox")
        if (previousSecret == null) {
            assert currentSecret == null
            return true
        }
        assert previousSecret.metadata.name == name // Just an assertion on the test code itself.
        assert currentSecret.data.keySet() == previousSecret.data.keySet()
        for (k in currentSecret.data.keySet()) {
            if (k.endsWith("cert.pem")) {
                def currentCert = Cert.loadBase64EncodedCert(currentSecret.data[k])
                def previousCert = Cert.loadBase64EncodedCert(previousSecret.data[k])
                assert currentCert.notAfter.after(previousCert.notAfter)
                assert currentCert.getSerialNumber() != previousCert.getSerialNumber()
                assert extractCNFromCert(currentCert) == expectedCertCN
            } else if (!k.endsWith("key.pem")) {
                assert currentSecret.data[k] == previousSecret.data[k]
            }
        }

        return true
    }

    def "Test sensor cert rotation with upgrader succeeds for non-Helm clusters"() {
        when:
        "Check that the cluster is not Helm-managed"
        def cluster = ClusterService.getCluster()
        assert cluster

        Assume.assumeFalse(cluster.hasHelmConfig())

        and:
        "Fetch the current sensor-tls, collector-tls and admission-control-tls secrets, and trigger cert rotation"
        def sensorTLSSecret = orchestrator.getSecret("sensor-tls", "stackrox")
        assert sensorTLSSecret
        def collectorTLSSecret = orchestrator.getSecret("collector-tls", "stackrox")
        assert collectorTLSSecret
        def admissionControlTLSSecret = orchestrator.getSecret("admission-control-tls", "stackrox")
        // Intentionally don't assert, admission control TLS secret may not be present.

        def start = System.currentTimeSeconds()
        SensorUpgradeService.triggerCertRotation(cluster.getId())

        and:
        "Wait for cert rotation to complete"
        cluster = ClusterService.getCluster()
        def mostRecentProcess = cluster.getStatus().getUpgradeStatus().getMostRecentProcess()
        // Make sure the most recent process was just started. Subtract 60 seconds to allow for clock-skew
        assert mostRecentProcess.initiatedAt.seconds > start - 60
        assert mostRecentProcess.getType() == UpgradeProcessType.CERT_ROTATION
        def processID = mostRecentProcess.getId()
        withRetry(50, 5) {
            cluster = ClusterService.getCluster()
            mostRecentProcess = cluster.getStatus().getUpgradeStatus().getMostRecentProcess()
            assert mostRecentProcess.getId() == processID
            assert mostRecentProcess.getProgress().getUpgradeState() == UpgradeState.UPGRADE_COMPLETE
        }

        then:
        checkCurrentValueOfSecretIdenticalExceptNewCerts("SENSOR_SERVICE: ${cluster.getId()}", sensorTLSSecret,
                "sensor-tls")
        checkCurrentValueOfSecretIdenticalExceptNewCerts("COLLECTOR_SERVICE: ${cluster.getId()}", collectorTLSSecret,
                "collector-tls")
        checkCurrentValueOfSecretIdenticalExceptNewCerts("ADMISSION_CONTROL_SERVICE: ${cluster.getId()}",
                admissionControlTLSSecret, "admission-control-tls")

        // Cleanup: revert secrets to what they were before this test was run.
        cleanup:
        orchestrator.updateSecret(sensorTLSSecret)
        orchestrator.updateSecret(collectorTLSSecret)
        if (admissionControlTLSSecret != null) {
            orchestrator.updateSecret(admissionControlTLSSecret)
        }
    }

    def "Test sensor cert rotation with upgrader fails for Helm clusters"() {
        when:
        "Check that the cluster is Helm-managed"
        def cluster = ClusterService.getCluster()
        assert cluster

        Assume.assumeTrue(cluster.hasHelmConfig())

        // The following can NOT be rephrased using `thrown()`, as that also catches the
        // AssumptionViolatedException generated if the assumption fails.
        then:
        "Trigger certificate rotation should result in an exception"
        def caughtException = false
        try {
            SensorUpgradeService.triggerCertRotation(cluster.getId())
        } catch (StatusRuntimeException exc) {
            caughtException = true
            assert exc.status.description.contains("cluster is Helm-managed")
        }
        assert caughtException
    }

}
