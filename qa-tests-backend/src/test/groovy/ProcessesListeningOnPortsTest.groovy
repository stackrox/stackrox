import static util.Helpers.evaluateWithRetry

import objects.Deployment
import objects.K8sServiceAccount
import objects.Service
import util.Env
import util.Helpers

import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Stepwise

import services.ProcessesListeningOnPortsService

@Stepwise
@IgnoreIf({ !Env.get("ROX_POSTGRES_DATASTORE", null) })
class ProcessesListeningOnPortsTest extends BaseSpecification {

    // Deployment names
    static final private String TCPCONNECTIONTARGET1 = "tcp-connection-target-1"
    static final private String TCPCONNECTIONTARGET2 = "tcp-connection-target-2"
    static final private String TCPCONNECTIONTARGET3 = "tcp-connection-target-3"

    // Other namespace
    static final private String OTHER_NAMESPACE = "qa2"

    static final private String SOCAT_DEBUG = "-d -d -v"

    // Target deployments
    @Shared
    private List<Deployment> targetDeployments

    def buildTargetDeployments() {
        return [
            new Deployment()
                    .setName(TCPCONNECTIONTARGET1)
                    .setImage("quay.io/rhacs-eng/qa:socat")
                    .addPort(80)
                    .addPort(8080)
                    .addLabel("app", TCPCONNECTIONTARGET1)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:80,fork STDOUT & " +
                                      "socat "+SOCAT_DEBUG+" TCP-LISTEN:8080,fork STDOUT)" as String,]),
            new Deployment()
                    .setName(TCPCONNECTIONTARGET2)
                    .setImage("quay.io/rhacs-eng/qa:socat")
                    .addPort(8081, "TCP")
                    .addLabel("app", TCPCONNECTIONTARGET2)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:8081,fork STDOUT)" as String,]),
            new Deployment()
                    .setName(TCPCONNECTIONTARGET3)
                    .setImage("quay.io/rhacs-eng/qa:socat")
                    .addPort(8082, "TCP")
                    .addLabel("app", TCPCONNECTIONTARGET3)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:8082,fork STDOUT & " +
                            "sleep 90 && pkill socat && sleep 3600)" as String,]),
                    // The 8082 port is opened. 90 seconds later the process is killed. After that we sleep forever
        ]
    }

    def createDeployments() {
        targetDeployments = buildTargetDeployments()
        orchestrator.batchCreateDeployments(targetDeployments)
        for (Deployment d : targetDeployments) {
            assert Services.waitForDeployment(d)
        }
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
        for (Deployment deployment : targetDeployments) {
            orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
        }
        for (Deployment deployment : targetDeployments) {
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
            createDeployments()
            log.info ">>>> Done <<<<<"
        }
    }

    @Tag("BAT")
    def "Verify networking endpoints with processes appear in API at the deployment level"() {
        given:
        "Two deployments that listen on ports are started up"

        setupSpec()
        rebuildForRetries()

        String deploymentId1 = targetDeployments[0].getDeploymentUid()
        String deploymentId2 = targetDeployments[1].getDeploymentUid()

        def gotCorrectNumElements = waitForResponseToHaveNumElements(2, deploymentId1, 240)

        assert gotCorrectNumElements

        def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(deploymentId1)
                return temp
        }

        assert processesListeningOnPorts

        def list = processesListeningOnPorts.listeningEndpointsList
        assert list.size() == 2

        def endpoint1 = list.find { it.endpoint.port == 80 }

        assert endpoint1
        assert endpoint1.deploymentId
        assert endpoint1.podId
        assert endpoint1.containerName == TCPCONNECTIONTARGET1
        assert endpoint1.signal.name == "socat"
        assert endpoint1.signal.execFilePath == "/usr/bin/socat"
        assert endpoint1.signal.args == "-d -d -v TCP-LISTEN:80,fork STDOUT"

        def endpoint2 = list.find { it.endpoint.port == 8080 }

        assert endpoint2
        assert endpoint2.deploymentId
        assert endpoint2.podId
        assert endpoint2.containerName == TCPCONNECTIONTARGET1
        assert endpoint2.signal.name == "socat"
        assert endpoint2.signal.execFilePath == "/usr/bin/socat"
        assert endpoint2.signal.args == "-d -d -v TCP-LISTEN:8080,fork STDOUT"

        gotCorrectNumElements = waitForResponseToHaveNumElements(1, deploymentId2, 240)

        assert gotCorrectNumElements

        processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(deploymentId2)
                return temp
        }

        assert processesListeningOnPorts

        list = processesListeningOnPorts.listeningEndpointsList
        assert list.size() == 1

        def endpoint = list.find { it.endpoint.port == 8081 }

        assert endpoint
        assert endpoint.deploymentId
        assert endpoint.podId
        assert endpoint.containerName == TCPCONNECTIONTARGET2
        assert endpoint.signal.name == "socat"
        assert endpoint.signal.execFilePath == "/usr/bin/socat"
        assert endpoint.signal.args == "-d -d -v TCP-LISTEN:8081,fork STDOUT"

        destroyDeployments()

        gotCorrectNumElements = waitForResponseToHaveNumElements(0, deploymentId1, 240)

        assert gotCorrectNumElements

        processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(deploymentId1)
                return temp
        }

        assert processesListeningOnPorts

        def list2 = processesListeningOnPorts.listeningEndpointsList
        assert list2.size() == 0

        gotCorrectNumElements = waitForResponseToHaveNumElements(0, deploymentId2, 240)

        assert gotCorrectNumElements

        processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(deploymentId2)
                return temp
        }

        assert processesListeningOnPorts

        def list3 = processesListeningOnPorts.listeningEndpointsList
        assert list3.size() == 0

        destroyDeployments()
    }

    @Tag("BAT")
    def "Verify networking endpoints disappear when process is terminated"() {
        given:
        "When a deployment listening on a port is created and then the process is terminated"

        setupSpec()
        rebuildForRetries()

        String deploymentId3 = targetDeployments[2].getDeploymentUid()

        def gotCorrectNumElements = waitForResponseToHaveNumElements(1, deploymentId3, 240)

        // First check that the listening endpoint appears in the API
        assert gotCorrectNumElements

        def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(deploymentId3)
                return temp
        }

        assert processesListeningOnPorts

        def list = processesListeningOnPorts.listeningEndpointsList
        assert list.size() == 1

        def endpoint = list.find { it.endpoint.port == 8082 }

        assert endpoint
        assert endpoint.containerName == TCPCONNECTIONTARGET3
        assert endpoint.signal.id
        assert endpoint.signal.containerId
        assert endpoint.signal.time
        assert endpoint.signal.name == "socat"
        assert endpoint.signal.execFilePath == "/usr/bin/socat"
        assert endpoint.signal.args == "-d -d -v TCP-LISTEN:8082,fork STDOUT"
        assert endpoint.signal.pid

        gotCorrectNumElements = waitForResponseToHaveNumElements(0, deploymentId3, 180)

        // Allow enough time for the process and port to close and check that it is not in the API response
        assert gotCorrectNumElements

        destroyDeployments()
    }

    @Tag("BAT")
    def "Verify networking endpoints doesn't disappear when port stays open"() {
        given:
        "A deployment listening on a port is brought up and it is checked twice that the port is found"

        setupSpec()
        rebuildForRetries()

        String deploymentId2 = targetDeployments[1].getDeploymentUid()

        def gotCorrectNumElements = waitForResponseToHaveNumElements(1, deploymentId2, 240)

        // First check that the listening endpoint appears in the API
        assert gotCorrectNumElements

        def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(deploymentId2)
                return temp
        }

        assert processesListeningOnPorts

        def list = processesListeningOnPorts.listeningEndpointsList
        assert list.size() == 1

        def endpoint = list.find { it.endpoint.port == 8081 }

        assert endpoint
        assert endpoint.containerName == TCPCONNECTIONTARGET2
        assert endpoint.signal.id
        assert endpoint.signal.containerId
        assert endpoint.signal.time
        assert endpoint.signal.name == "socat"
        assert endpoint.signal.execFilePath == "/usr/bin/socat"
        assert endpoint.signal.args == "-d -d -v TCP-LISTEN:8081,fork STDOUT"
        assert endpoint.signal.pid

        gotCorrectNumElements = waitForResponseToHaveNumElements(0, deploymentId2, 65)

        // Confirm that the listening endpoint still appears in the API 65 seconds later
        assert !gotCorrectNumElements

        destroyDeployments()
    }

    private waitForResponseToHaveNumElements(int numElements,
        String deploymentId, int timeoutSeconds = 240) {

        int intervalSeconds = 1
        int waitTime
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                    def temp = ProcessesListeningOnPortsService
                            .getProcessesListeningOnPortsResponse(deploymentId)
                    return temp
            }

            def list = processesListeningOnPorts.listeningEndpointsList

            if (list.size() == numElements) {
                return true
            }
            sleep intervalSeconds * 1000
        }
        log.info "Timedout waiting for response to have {$numElements} elements"
        return false
    }
}
