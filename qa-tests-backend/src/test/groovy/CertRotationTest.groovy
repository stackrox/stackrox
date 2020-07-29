import com.google.gson.JsonObject
import com.google.gson.JsonPrimitive
import groups.BAT
import io.fabric8.kubernetes.api.model.Secret
import org.junit.experimental.categories.Category
import org.yaml.snakeyaml.Yaml
import services.ClusterService
import services.DirectHTTPService
import util.Cert

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

}

