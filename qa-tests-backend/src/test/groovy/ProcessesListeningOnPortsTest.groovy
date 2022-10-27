import static com.jayway.restassured.RestAssured.given

import orchestratormanager.OrchestratorTypes

import groups.BAT
import groups.Integration
import objects.Deployment
import objects.K8sServiceAccount
import objects.Service
import util.Env
import util.Helpers
import util.NetworkGraphUtil

import org.junit.experimental.categories.Category
import spock.lang.Shared
import spock.lang.Stepwise

import services.ProcessesListeningOnPortsService

@Stepwise
class ProcessesListeningOnPortsTest extends BaseSpecification {

//    // Deployment names
//    static final private String UDPCONNECTIONTARGET = "udp-connection-target"
//    static final private String TCPCONNECTIONTARGET = "tcp-connection-target"
//    static final private String NGINXCONNECTIONTARGET = "nginx-connection-target"
//    static final private String UDPCONNECTIONSOURCE = "udp-connection-source"
//    static final private String TCPCONNECTIONSOURCE = "tcp-connection-source"
//    //static final private String ICMPCONNECTIONSOURCE = "icmp-connection-source"
//    static final private String NOCONNECTIONSOURCE = "no-connection-source"
//    static final private String SHORTCONSISTENTSOURCE = "short-consistent-source"
//    static final private String SINGLECONNECTIONSOURCE = "single-connection-source"
//    static final private String MULTIPLEPORTSCONNECTION = "two-ports-connect-source"
//    static final private String EXTERNALDESTINATION = "external-destination-source"
//
//    // Other namespace
//    static final private String OTHER_NAMESPACE = "qa2"
//
//    static final private String SOCAT_DEBUG = "-d -d -v"
//
//    // Target deployments
//    @Shared
//    private List<Deployment> targetDeployments
//
//    def buildTargetDeployments() {
//        return [
//            new Deployment()
//                    .setName(UDPCONNECTIONTARGET)
//                    .setImage("quay.io/rhacs-eng/qa:socat")
//                    .addPort(8080, "UDP")
//                    .addLabel("app", UDPCONNECTIONTARGET)
//                    .setExposeAsService(true)
//                    .setCommand(["/bin/sh", "-c",])
//                    .setArgs(["socat "+SOCAT_DEBUG+" UDP-RECV:8080 STDOUT",]),
//            new Deployment()
//                    .setName(TCPCONNECTIONTARGET)
//                    .setImage("quay.io/rhacs-eng/qa:socat")
//                    .addPort(80)
//                    .addPort(8080)
//                    .addLabel("app", TCPCONNECTIONTARGET)
//                    .setExposeAsService(true)
//                    .setCommand(["/bin/sh", "-c",])
//                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:80,fork STDOUT & " +
//                                      "socat "+SOCAT_DEBUG+" TCP-LISTEN:8080,fork STDOUT)" as String,]),
//            new Deployment()
//                    .setName(NGINXCONNECTIONTARGET)
//                    .setImage("quay.io/rhacs-eng/qa:nginx")
//                    .addPort(80)
//                    .addLabel("app", NGINXCONNECTIONTARGET)
//                    .setExposeAsService(true)
//                    .setCreateLoadBalancer(true)
//                    .setCreateRoute(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT),
//        ]
//    }
//
//    // Source deployments
//    @Shared
//    private List<Deployment> sourceDeployments
//
//    def buildSourceDeployments() {
//        return [
//            new Deployment()
//                    .setName(NOCONNECTIONSOURCE)
//                    .setImage("quay.io/rhacs-eng/qa:nginx")
//                    .addLabel("app", NOCONNECTIONSOURCE),
//            new Deployment()
//                    .setName(SHORTCONSISTENTSOURCE)
//                    .setImage("quay.io/rhacs-eng/qa:nginx-1.15.4-alpine")
//                    .addLabel("app", SHORTCONSISTENTSOURCE)
//                    .setCommand(["/bin/sh", "-c",])
//                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS}; " +
//                                      "do wget -S -T 2 http://${NGINXCONNECTIONTARGET}; " +
//                                      "done" as String,]),
//            new Deployment()
//                    .setName(SINGLECONNECTIONSOURCE)
//                    .setImage("quay.io/rhacs-eng/qa:nginx-1.15.4-alpine")
//                    .addLabel("app", SINGLECONNECTIONSOURCE)
//                    .setCommand(["/bin/sh", "-c",])
//                    .setArgs(["wget -S -T 2 http://${NGINXCONNECTIONTARGET} && " +
//                                      "while sleep 30; do echo hello; done" as String,]),
//            new Deployment()
//                    .setName(UDPCONNECTIONSOURCE)
//                    .setImage("quay.io/rhacs-eng/qa:socat")
//                    .addLabel("app", UDPCONNECTIONSOURCE)
//                    .setCommand(["/bin/sh", "-c",])
//                    .setArgs(["while sleep 5; " +
//                                      "do echo \"Hello from ${UDPCONNECTIONSOURCE}\" | " +
//                                      "socat "+SOCAT_DEBUG+" -s STDIN UDP:${UDPCONNECTIONTARGET}:8080; " +
//                                      "done" as String,]),
//            new Deployment()
//                    .setName(TCPCONNECTIONSOURCE)
//                    .setImage("quay.io/rhacs-eng/qa:socat")
//                    .addLabel("app", TCPCONNECTIONSOURCE)
//                    .setCommand(["/bin/sh", "-c",])
//                    .setArgs(["while sleep 5; " +
//                                      "do echo \"Hello from ${TCPCONNECTIONSOURCE}\" | " +
//                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
//                                      "done" as String,]),
//            new Deployment()
//                    .setName(MULTIPLEPORTSCONNECTION)
//                    .setImage("quay.io/rhacs-eng/qa:socat")
//                    .addLabel("app", MULTIPLEPORTSCONNECTION)
//                    .setCommand(["/bin/sh", "-c",])
//                    .setArgs(["while sleep 5; " +
//                                      "do echo \"Hello from ${MULTIPLEPORTSCONNECTION}\" | " +
//                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
//                                      "echo \"Hello from ${MULTIPLEPORTSCONNECTION}\" | " +
//                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:8080; " +
//                                      "done" as String,]),
//            new Deployment()
//                    .setName(EXTERNALDESTINATION)
//                    .setImage("quay.io/rhacs-eng/qa:nginx-1.15.4-alpine")
//                    .addLabel("app", EXTERNALDESTINATION)
//                    .setCommand(["/bin/sh", "-c",])
//                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS}; " +
//                                      "do wget -S -T 2 http://www.google.com; " +
//                                      "done" as String,]),
//            new Deployment()
//                    .setName("${TCPCONNECTIONSOURCE}-qa2")
//                    .setNamespace(OTHER_NAMESPACE)
//                    .setImage("quay.io/rhacs-eng/qa:socat")
//                    .addLabel("app", "${TCPCONNECTIONSOURCE}-qa2")
//                    .setCommand(["/bin/sh", "-c",])
//                    .setArgs(["while sleep 5; " +
//                                      "do echo \"Hello from ${TCPCONNECTIONSOURCE}-qa2\" | " +
//                                      "socat "+SOCAT_DEBUG+" -s STDIN "+
//                                         "TCP:${TCPCONNECTIONTARGET}.qa.svc.cluster.local:80; " +
//                                      "done" as String,]),
//        ]
//    }
//
//    @Shared
//    private List<Deployment> deployments
//
//    def createDeployments() {
//        targetDeployments = buildTargetDeployments()
//        orchestrator.batchCreateDeployments(targetDeployments)
//        for (Deployment d : targetDeployments) {
//            assert Services.waitForDeployment(d)
//        }
//        sourceDeployments = buildSourceDeployments()
//        orchestrator.batchCreateDeployments(sourceDeployments)
//        for (Deployment d : sourceDeployments) {
//            assert Services.waitForDeployment(d)
//        }
//        deployments = sourceDeployments + targetDeployments
//        //
//        // Commenting out ICMP test setup for now
//        // See ROX-635
//        //
//        /*
//        def nginxIp = DEPLOYMENTS.find { it.name == NGINXCONNECTIONTARGET }?.pods?.get(0)?.podIP
//        Deployment icmp = new Deployment()
//                .setName(ICMPCONNECTIONSOURCE)
//                .setImage("ubuntu")
//                .addLabel("app", ICMPCONNECTIONSOURCE)
//                .setCommand(["/bin/sh", "-c",])
//                .setArgs(["apt-get update && " +
//                                  "apt-get install iputils-ping -y && " +
//                                  "ping ${nginxIp}" as String,])
//        orchestrator.createDeployment(icmp)
//        DEPLOYMENTS.add(icmp)
//        */
//    }
//
//    def setupSpec() {
//        orchestrator.createNamespace(OTHER_NAMESPACE)
//        orchestrator.createImagePullSecret(
//                "quay",
//                Env.mustGet("REGISTRY_USERNAME"),
//                Env.mustGet("REGISTRY_PASSWORD"),
//                OTHER_NAMESPACE,
//                "https://quay.io"
//        )
//        orchestrator.createServiceAccount(
//                new K8sServiceAccount(
//                        name: "default",
//                        namespace: OTHER_NAMESPACE,
//                        imagePullSecrets: ["quay"]
//                )
//        )
//
//        createDeployments()
//    }
//
//    def destroyDeployments() {
//        for (Deployment deployment : deployments) {
//            orchestrator.deleteDeployment(deployment)
//        }
//        for (Deployment deployment : deployments) {
//            if (deployment.exposeAsService) {
//                orchestrator.waitForServiceDeletion(new Service(deployment.name, deployment.namespace))
//            }
//        }
//        orchestrator.deleteNamespace(OTHER_NAMESPACE)
//        orchestrator.waitForNamespaceDeletion(OTHER_NAMESPACE)
//    }
//
//    def cleanupSpec() {
//        destroyDeployments()
//    }
//
//    def rebuildForRetries() {
//        if (Helpers.getAttemptCount() > 1) {
//            log.info ">>>> Recreating test deployments prior to retest <<<<<"
//            destroyDeployments()
//            sleep(5000)
//            createDeployments()
//            sleep(5000)
//            log.info ">>>> Done <<<<<"
//        }
//    }

    // Deployment names
    static final private String UDPCONNECTIONTARGET = "udp-connection-target"
    static final private String TCPCONNECTIONTARGET = "tcp-connection-target"
    static final private String NGINXCONNECTIONTARGET = "nginx-connection-target"
    static final private String UDPCONNECTIONSOURCE = "udp-connection-source"
    static final private String TCPCONNECTIONSOURCE = "tcp-connection-source"
    //static final private String ICMPCONNECTIONSOURCE = "icmp-connection-source"
    static final private String NOCONNECTIONSOURCE = "no-connection-source"
    static final private String SHORTCONSISTENTSOURCE = "short-consistent-source"
    static final private String SINGLECONNECTIONSOURCE = "single-connection-source"
    static final private String MULTIPLEPORTSCONNECTION = "two-ports-connect-source"
    static final private String EXTERNALDESTINATION = "external-destination-source"

    // Other namespace
    static final private String OTHER_NAMESPACE = "qa2"

    static final private String SOCAT_DEBUG = "-d -d -v"

    // Target deployments
    @Shared
    private List<Deployment> targetDeployments

    def buildTargetDeployments() {
        return [
            new Deployment()
                    .setName(UDPCONNECTIONTARGET)
                    .setImage("quay.io/rhacs-eng/qa:socat")
                    .addPort(8080, "UDP")
                    .addLabel("app", UDPCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["socat "+SOCAT_DEBUG+" UDP-RECV:8080 STDOUT",]),
            new Deployment()
                    .setName(TCPCONNECTIONTARGET)
                    .setImage("quay.io/rhacs-eng/qa:socat")
                    .addPort(80)
                    .addPort(8080)
                    .addLabel("app", TCPCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:80,fork STDOUT & " +
                                      "socat "+SOCAT_DEBUG+" TCP-LISTEN:8080,fork STDOUT)" as String,]),
            new Deployment()
                    .setName(NGINXCONNECTIONTARGET)
                    .setImage("quay.io/rhacs-eng/qa:nginx")
                    .addPort(80)
                    .addLabel("app", NGINXCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCreateLoadBalancer(true)
                    .setCreateRoute(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT),
        ]
    }

    // Source deployments
    @Shared
    private List<Deployment> sourceDeployments

    def buildSourceDeployments() {
        return [
            new Deployment()
                    .setName(NOCONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa:nginx")
                    .addLabel("app", NOCONNECTIONSOURCE),
            new Deployment()
                    .setName(SHORTCONSISTENTSOURCE)
                    .setImage("quay.io/rhacs-eng/qa:nginx-1.15.4-alpine")
                    .addLabel("app", SHORTCONSISTENTSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS}; " +
                                      "do wget -S -T 2 http://${NGINXCONNECTIONTARGET}; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(SINGLECONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa:nginx-1.15.4-alpine")
                    .addLabel("app", SINGLECONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["wget -S -T 2 http://${NGINXCONNECTIONTARGET} && " +
                                      "while sleep 30; do echo hello; done" as String,]),
            new Deployment()
                    .setName(UDPCONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa:socat")
                    .addLabel("app", UDPCONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${UDPCONNECTIONSOURCE}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN UDP:${UDPCONNECTIONTARGET}:8080; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(TCPCONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa:socat")
                    .addLabel("app", TCPCONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${TCPCONNECTIONSOURCE}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(MULTIPLEPORTSCONNECTION)
                    .setImage("quay.io/rhacs-eng/qa:socat")
                    .addLabel("app", MULTIPLEPORTSCONNECTION)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${MULTIPLEPORTSCONNECTION}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
                                      "echo \"Hello from ${MULTIPLEPORTSCONNECTION}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:8080; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(EXTERNALDESTINATION)
                    .setImage("quay.io/rhacs-eng/qa:nginx-1.15.4-alpine")
                    .addLabel("app", EXTERNALDESTINATION)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS}; " +
                                      "do wget -S -T 2 http://www.google.com; " +
                                      "done" as String,]),
            new Deployment()
                    .setName("${TCPCONNECTIONSOURCE}-qa2")
                    .setNamespace(OTHER_NAMESPACE)
                    .setImage("quay.io/rhacs-eng/qa:socat")
                    .addLabel("app", "${TCPCONNECTIONSOURCE}-qa2")
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${TCPCONNECTIONSOURCE}-qa2\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN "+
                                         "TCP:${TCPCONNECTIONTARGET}.qa.svc.cluster.local:80; " +
                                      "done" as String,]),
        ]
    }

    @Shared
    private List<Deployment> deployments

    def createDeployments() {
        targetDeployments = buildTargetDeployments()
        orchestrator.batchCreateDeployments(targetDeployments)
        for (Deployment d : targetDeployments) {
            assert Services.waitForDeployment(d)
        }
        sourceDeployments = buildSourceDeployments()
        orchestrator.batchCreateDeployments(sourceDeployments)
        for (Deployment d : sourceDeployments) {
            assert Services.waitForDeployment(d)
        }
        deployments = sourceDeployments + targetDeployments
        //
        // Commenting out ICMP test setup for now
        // See ROX-635
        //
        /*
        def nginxIp = DEPLOYMENTS.find { it.name == NGINXCONNECTIONTARGET }?.pods?.get(0)?.podIP
        Deployment icmp = new Deployment()
                .setName(ICMPCONNECTIONSOURCE)
                .setImage("ubuntu")
                .addLabel("app", ICMPCONNECTIONSOURCE)
                .setCommand(["/bin/sh", "-c",])
                .setArgs(["apt-get update && " +
                                  "apt-get install iputils-ping -y && " +
                                  "ping ${nginxIp}" as String,])
        orchestrator.createDeployment(icmp)
        DEPLOYMENTS.add(icmp)
        */
    }

    def setupSpec() {
        orchestrator.createNamespace(OTHER_NAMESPACE)
        orchestrator.createImagePullSecret(
                "quay",
                Env.mustGet("REGISTRY_USERNAME"),
                Env.mustGet("REGISTRY_PASSWORD"),
                OTHER_NAMESPACE,
                "https://quay.io"
        )
        orchestrator.createServiceAccount(
                new K8sServiceAccount(
                        name: "default",
                        namespace: OTHER_NAMESPACE,
                        imagePullSecrets: ["quay"]
                )
        )

        createDeployments()
    }

    def destroyDeployments() {
        for (Deployment deployment : deployments) {
            orchestrator.deleteDeployment(deployment)
        }
        for (Deployment deployment : deployments) {
            if (deployment.exposeAsService) {
                orchestrator.waitForServiceDeletion(new Service(deployment.name, deployment.namespace))
            }
        }
        orchestrator.deleteNamespace(OTHER_NAMESPACE)
        orchestrator.waitForNamespaceDeletion(OTHER_NAMESPACE)
    }

    def cleanupSpec() {
        destroyDeployments()
    }

    def rebuildForRetries() {
        if (Helpers.getAttemptCount() > 1) {
            log.info ">>>> Recreating test deployments prior to retest <<<<<"
            destroyDeployments()
            sleep(5000)
            createDeployments()
            sleep(5000)
            log.info ">>>> Done <<<<<"
        }
    }

    @Category([BAT, Integration])
    def "Verify networking endpoints with processes appear in API at the namespace level"() {
        given:
        "An nginx pod is started"
        //  rebuildForRetries()

        //  String namespace = deployments[0].getNamespace()
        String namespace = "qa"

        def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsWithDeploymentResponse(namespace)
                return temp
        }

        assert processesListeningOnPorts

        def list = processesListeningOnPorts.processesListeningOnPortsWithDeploymentList

        assert list.size() == 1
        assert list.find { it.deploymentId == "nginx" } != null
        assert list.get(0).processesListeningOnPortsList.port == [80]
        assert list.get(0).processesListeningOnPortsList.process.processName == ["nginx"]
        assert list.get(0).processesListeningOnPortsList.process.processExecFilePath == ["/usr/bin/nginx"]
        assert list.get(0).processesListeningOnPortsList.process.processArgs == ["fake args"]
    }

    @Category([BAT, Integration])
    def "Verify networking endpoints with processes appear in API at the deployment level"() {
        given:
        "An nginx pod is started"
        rebuildForRetries()

        //  String namespace = deployments[0].getNamespace()
        String namespace = "qa"
        String deploymentId = "nginx"

        def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(namespace, deploymentId)
                return temp
        }

        assert processesListeningOnPorts

        def list = processesListeningOnPorts.processesListeningOnPortsList

        log.info "${Env.mustGet("REGISTRY_USERNAME")}"

        assert list.size() == 2

        assert list.get(0).port == 80
        assert list.get(0).process.podId == "nginx-5da7f5-fdsaf"
        assert list.get(0).process.containerName == "nginx"
        assert list.get(0).process.processName == "nginx"
        assert list.get(0).process.processExecFilePath == "/usr/bin/nginx"
        assert list.get(0).process.processArgs == "fake args"

        assert list.get(1).port == 8080
        assert list.get(1).process.podId == "visa-5da7f5-fdsaf"
        assert list.get(1).process.containerName == "visa"
        assert list.get(1).process.processName == "visa"
        assert list.get(1).process.processExecFilePath == "/usr/bin/visa"
        assert list.get(1).process.processArgs == "fake args for visa"
    }
}
