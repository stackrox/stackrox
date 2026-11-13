import java.nio.charset.Charset
import java.security.cert.X509Certificate

import com.google.gson.JsonObject
import com.google.gson.JsonPrimitive
import io.fabric8.kubernetes.api.model.Secret
import org.bouncycastle.asn1.x500.X500Name
import org.bouncycastle.asn1.x500.style.BCStyle
import org.bouncycastle.asn1.x500.style.IETFUtils
import org.yaml.snakeyaml.Yaml

import services.ClusterService
import services.DirectHTTPService
import util.Cert
import util.Env

import spock.lang.IgnoreIf
import spock.lang.Tag

@Tag("BAT")
@Tag("PZ")
// skip if executed in a test environment with just secured-cluster deployed in the test cluster
// i.e. central is deployed elsewhere
@IgnoreIf({ Env.ONLY_SECURED_CLUSTER == "true" })
@IgnoreIf({ Env.IS_BYODB })
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
        String principalShouldContain, Long timestampBeforeGeneration,
        String regeneratedSecretName = null
    ){
        def searchName = regeneratedSecretName ?: currentSecret.metadata.name
        def regeneratedSecretObj = regeneratedSecrets.
            find { it["metadata"]["name"] == searchName }
        assert regeneratedSecretObj
        Map<String, String> regeneratedSecretData =
                (Map<String, String>) regeneratedSecretObj["data"] ?:
                ((Map<String, String>) regeneratedSecretObj["stringData"])?.collectEntries { key, value ->
            [(key): Base64.encoder.encodeToString(value.getBytes(Charset.defaultCharset()))]
        }
        assert regeneratedSecretData
        if (currentSecret.data.keySet() == regeneratedSecretData.keySet()) {
            assertSameKeysAndSameValuesExcept(currentSecret.data, regeneratedSecretData,
                [certFileName, keyFileName] as Set<String>)
        } else {
            // CRS-issued secrets use different key names (cert.pem, key.pem) than
            // certgen output (sensor-cert.pem, sensor-key.pem). Compare only ca.pem.
            assert currentSecret.data.containsKey("ca.pem")
            assert regeneratedSecretData.containsKey("ca.pem")
            assert currentSecret.data["ca.pem"] == ((String)regeneratedSecretData["ca.pem"]).trim()
        }
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

    def "Test sensor cert rotation"() {
        when:
        "Fetch the current sensor-tls, collector-tls and admission-control-tls secrets, and regenerate certs"
        // With CRS-based cluster registration, TLS secrets use new naming (tls-cert-sensor, etc.).
        // The certgen API always produces secrets with legacy naming (sensor-tls, etc.).
        def sensorTLSSecret = orchestrator.getSecret("sensor-tls", "stackrox") ?:
            orchestrator.getSecret("tls-cert-sensor", "stackrox")
        assert sensorTLSSecret
        def collectorTLSSecret = orchestrator.getSecret("collector-tls", "stackrox") ?:
            orchestrator.getSecret("tls-cert-collector", "stackrox")
        assert collectorTLSSecret
        def admissionControlTLSSecret = orchestrator.getSecret("admission-control-tls", "stackrox") ?:
            orchestrator.getSecret("tls-cert-admission-control", "stackrox")
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
            "sensor-cert.pem", "sensor-key.pem", "SENSOR_SERVICE: ${cluster.getId()}", start,
            "sensor-tls")
        testMatchingSecretFoundWithExpectedProperties(regeneratedSecrets, collectorTLSSecret,
            "collector-cert.pem", "collector-key.pem", "COLLECTOR_SERVICE: ${cluster.getId()}", start,
            "collector-tls")
        if (admissionControlSecretPresent) {
            testMatchingSecretFoundWithExpectedProperties(regeneratedSecrets, admissionControlTLSSecret,
                "admission-control-cert.pem", "admission-control-key.pem",
                "ADMISSION_CONTROL_SERVICE: ${cluster.getId()}", start,
                "admission-control-tls")
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

}
