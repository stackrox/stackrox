import com.google.gson.JsonObject
import com.google.gson.JsonPrimitive
import groups.BAT
import io.fabric8.kubernetes.api.model.Secret
import io.stackrox.proto.storage.ClusterOuterClass.ClusterUpgradeStatus.UpgradeProcessStatus.UpgradeProcessType
import io.stackrox.proto.storage.ClusterOuterClass.UpgradeProgress.UpgradeState
import org.junit.experimental.categories.Category
import org.yaml.snakeyaml.Yaml
import services.ClusterService
import services.DirectHTTPService
import services.SensorUpgradeService
import util.Cert

import java.security.cert.X509Certificate

@Category(BAT)
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
        def regeneratedSecretData = (Map<String, Object>) regeneratedSecrets.
            find { it["metadata"]["name"] == currentSecret.metadata.name } ["data"]
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
        println "Admission control secret present: ${admissionControlSecretPresent}"
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
        def name = cert.subjectX500Principal.getName()
        return name[name.indexOf("CN=")..-1]
    }

    def checkCurrentValueOfSecretIdenticalExceptNewCerts(Secret previousSecret, String name) {
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
                assert extractCNFromCert(currentCert) == extractCNFromCert(previousCert)
            } else if (!k.endsWith("key.pem")) {
                assert currentSecret.data[k] == previousSecret.data[k]
            }
        }

        return true
    }

    def "Test sensor cert rotation with upgrader"() {
        when:
        "Fetch the current sensor-tls, collector-tls and admission-control-tls secrets, and trigger cert rotation"
        def sensorTLSSecret = orchestrator.getSecret("sensor-tls", "stackrox")
        assert sensorTLSSecret
        def collectorTLSSecret = orchestrator.getSecret("collector-tls", "stackrox")
        assert collectorTLSSecret
        def admissionControlTLSSecret = orchestrator.getSecret("admission-control-tls", "stackrox")
        // Intentionally don't assert, admission control TLS secret may not be present.

        def cluster = ClusterService.getCluster()
        assert cluster
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
        checkCurrentValueOfSecretIdenticalExceptNewCerts(sensorTLSSecret, "sensor-tls")
        checkCurrentValueOfSecretIdenticalExceptNewCerts(collectorTLSSecret, "collector-tls")
        checkCurrentValueOfSecretIdenticalExceptNewCerts(admissionControlTLSSecret, "admission-control-tls")
    }

}

