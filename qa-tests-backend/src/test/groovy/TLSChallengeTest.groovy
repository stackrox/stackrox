import static io.stackrox.proto.storage.ClusterOuterClass.ClusterHealthStatus.HealthStatusLabel
import io.fabric8.kubernetes.api.model.EnvVar
import orchestratormanager.OrchestratorManagerException
import spock.lang.Shared
import groups.SensorBounceNext
import io.stackrox.proto.storage.ClusterOuterClass
import objects.ConfigMap
import objects.Deployment
import objects.Secret
import services.ClusterService
import org.junit.experimental.categories.Category
import spock.lang.Retry
import util.ApplicationHealth
import util.Timer
import java.nio.file.Files
import java.nio.file.Paths

@Retry(count = 1)
class TLSChallengeTest extends BaseSpecification {
    @Shared
    private EnvVar originalCentralEndpoint = new EnvVar()
    private final static String PROXY_NAMESPACE = "qa-tls-challenge"
    private final static String CENTRAL_PROXY_ENDPOINT = "nginx-loadbalancer.${PROXY_NAMESPACE}:443"
    private final static String ASSETS_DIR = Paths.get(
            System.getProperty("user.dir"), "artifacts", "tls-challenge-test")

    private final static LEAF_KEY_CONTENT = Files.readAllBytes(
            Paths.get(ASSETS_DIR, "nginx-lb-certs", "leaf-key.pem"))
    private final static LEAF_CERT_CONTENT = Files.readAllBytes(
            Paths.get(ASSETS_DIR, "nginx-lb-certs", "leaf-cert.pem"))
    private final static CA_CERT_CONTENT = Files.readAllBytes(
            Paths.get(ASSETS_DIR, "nginx-lb-certs", "ca.pem"))

    def setupSpec() {
        originalCentralEndpoint = orchestrator.getDeploymentEnv("stackrox", "sensor", "ROX_CENTRAL_ENDPOINT")
        orchestrator.ensureNamespaceExists(PROXY_NAMESPACE)
        addStackroxImagePullSecret(PROXY_NAMESPACE)

        ByteArrayOutputStream out = new ByteArrayOutputStream()
        out.write(LEAF_CERT_CONTENT)
        out.write(CA_CERT_CONTENT)
        def certChain = out.toByteArray()

        deployNGINXProxy(certChain, LEAF_KEY_CONTENT)
    }

    def cleanupSpec() {
        orchestrator.deleteNamespace(PROXY_NAMESPACE)
        orchestrator.waitForNamespaceDeletion(PROXY_NAMESPACE)

        orchestrator.deleteSecret("additional-ca", "stackrox")
        orchestrator.restartPodByLabelWithExecKill("stackrox", [app: "central"])
        orchestrator.waitForPodsReady("stackrox", [app: "central"], 1, 50, 3)

        // Restart sensor to reset the gRPC connection to central.
        // Scale to 0 and back to 1 so that the check for sensor healthiness is based on the restarted sensor pod.
        orchestrator.scaleDeployment("stackrox", "sensor", 0)
        orchestrator.waitForAllPodsToBeRemoved("stackrox", ["app": "sensor"], 30, 5)
        orchestrator.updateDeploymentEnv("stackrox", "sensor",
                originalCentralEndpoint.name, originalCentralEndpoint.value)
        orchestrator.scaleDeployment("stackrox", "sensor", 1)
        ApplicationHealth ah = new ApplicationHealth(orchestrator, 600)
        ah.waitForSensorHealthiness()
        if (ClusterService.isOpenShift3()) {
            // OpenShift 3.11 networking needs a kick to ensure sensor is reachable (ROX-7869)
            sleep(5000)
            orchestrator.addOrUpdateServiceLabel("sensor", "stackrox", "kick",
                    System.currentTimeSeconds().toString())
        }

        orchestrator.deleteAllPodsAndWait("stackrox", [app: "collector"])
        ah.waitForCollectorHealthiness()

        withRetry(30, 1) { Services.getMetadataClient().getMetadata() }
        waitUntilCentralSensorConnectionIs(HealthStatusLabel.HEALTHY)
    }

    @Category(SensorBounceNext)
    def "Verify sensor can communicate with central behind an untrusted load balancer"() {
        when:
        "Deploying Sensor without root CA certs can't connect to load balancer"

        printlnDated "Setting sensor ROX_CENTRAL_ENDPOINT to ${CENTRAL_PROXY_ENDPOINT}"
        orchestrator.updateDeploymentEnv("stackrox", "sensor", "ROX_CENTRAL_ENDPOINT", CENTRAL_PROXY_ENDPOINT)
        printlnDated "Wait for sensor to be restarted"
        orchestrator.waitForPodsReady("stackrox", [app: "sensor"], 1, 10, 5)

        then:
        "Central connection to Sensor becomes unhealthy because root CAs are missing"
        printlnDated "Wait until Sensor connection is marked as UNHEALTHY or DEGRADED in Centrals clusters health"
        assert waitUntilCentralSensorConnectionIs(HealthStatusLabel.UNHEALTHY, HealthStatusLabel.DEGRADED)

        when:
        "Central receives additional CA configurations after restart"

        printlnDated "Create additional-ca secret"
        Secret additionalCASecret = new Secret(
                name: "additional-ca",
                namespace: "stackrox",
                type: "Opaque",
                data: [ "ca.crt": Base64.getEncoder().encodeToString(CA_CERT_CONTENT) ]
        )
        orchestrator.createSecret(additionalCASecret)

        // restart with "kill 1" to prevent deletion of PVs on local machines
        assert orchestrator.restartPodByLabelWithExecKill("stackrox", [app: "central"])
        printlnDated "Wait for central pod being ready again"
        orchestrator.waitForPodsReady("stackrox", [app: "central"], 1, 50, 3)

        // restart nginx load balancer
        orchestrator.restartPodByLabels(PROXY_NAMESPACE, [app: "nginx"], 30, 5)

        then:
        "Sensor receives root CAs from central after restart and is connected to central"

        // delete sensor to force reconnect
        printlnDated "Restart Sensor, should connect to ${CENTRAL_PROXY_ENDPOINT}"
        orchestrator.restartPodByLabels("stackrox", [app: "sensor"], 30, 5)

        printlnDated "Wait until Sensor is ready again"
        assert Services.waitForDeployment(new Deployment(name: "sensor", namespace: "stackrox"))

        // Check connection details Sensor <> Central
        assert checkSensorLogs()
        assert waitUntilCentralSensorConnectionIs(HealthStatusLabel.HEALTHY)
    }

    boolean checkSensorLogs() {
        def log = ""
        Timer t = new Timer(40, 5)
        while (t.IsValid()) {
            def pod = orchestrator.getPods("stackrox", "sensor").get(0)
            log = orchestrator.getPodLog("stackrox", pod.metadata.name)

            // Check if sensor logs contain connection information
            if (log.contains("Connecting to Central server ${CENTRAL_PROXY_ENDPOINT}")
                && log.contains("Communication with central started")) {
                printlnDated "Found successful connection logs in sensor pod"
                return true
            }
        }

        printlnDated "Could not establish connection to central ${CENTRAL_PROXY_ENDPOINT}"
        println log
        return false
    }

    boolean waitUntilCentralSensorConnectionIs(HealthStatusLabel... healthStatusLabels) {
        Timer t = new Timer(60, 5)
        while (t.IsValid()) {
            List<ClusterOuterClass.Cluster> list = Services.getClusterClient().getClusters().getClustersList()
            if (list.empty) {
                throw new OrchestratorManagerException(
                        "Did not found any cluster, maybe redeploy StackRox or register a new cluster.")
            }

            printlnDated "Receiving cluster status from central, checking sensor connection"
            HealthStatusLabel healthStatusLabel = list.get(0).getHealthStatus().getSensorHealthStatus()
            def found = healthStatusLabels.find { it == healthStatusLabel }
            printlnDated("Status is: ${healthStatusLabel}")
            if (found) {
                return true
            }
        }
        return false
    }

    def deployNGINXProxy(byte[] certChain, byte[] leafKeyContent) {
        def nginxConfig = new String(Files.readAllBytes(Paths.get(ASSETS_DIR, "nginx-proxy.conf")))
        ConfigMap nginxConfigMap = new ConfigMap(
                name: "nginx-proxy-conf",
                data: ["nginx-proxy-grpc-tls.conf": nginxConfig],
                namespace: PROXY_NAMESPACE
        )
        orchestrator.createConfigMap(nginxConfigMap)

        Secret tlsConfSecret = new Secret()
        tlsConfSecret.name = "nginx-tls-conf"
        tlsConfSecret.type = "tls"
        tlsConfSecret.namespace = PROXY_NAMESPACE
        tlsConfSecret.data = [
                "tls.crt": Base64.getEncoder().encodeToString(certChain),
                "tls.key": Base64.getEncoder().encodeToString(leafKeyContent),
        ]
        orchestrator.createSecret(tlsConfSecret)

        Deployment loadBalancerDeployment = new Deployment()
        loadBalancerDeployment.setNamespace(PROXY_NAMESPACE)
                .setName("nginx-loadbalancer")
                .setExposeAsService(true)
                .setImage("quay.io/rhacs-eng/qa:nginx-1-17-1")
                .addVolumeFromConfigMap(nginxConfigMap, "/etc/nginx/conf.d/")
                .addVolumeFromSecret(tlsConfSecret, "/run/secrets/tls/")
                .setTargetPort(8443)
                .setPorts([443: "TCP"])
        loadBalancerDeployment.setLabels([app: "nginx"])
        orchestrator.createDeployment(loadBalancerDeployment)
    }
}
