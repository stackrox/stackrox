import static util.Helpers.evaluateWithRetry

import objects.Deployment
import util.Env

import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Stepwise

import services.ProcessesListeningOnPortsService

@Stepwise
@Tag("BAT")
@Tag("Parallel")
@Tag("PZ")
@IgnoreIf({ Env.get("ROX_PROCESSES_LISTENING_ON_PORT", "true") != "true" })
class ProcessesListeningOnPortsTest extends BaseSpecification {

    // Deployment names
    static final private String TCPCONNECTIONTARGET1 = "tcp-connection-target-1"
    static final private String TCPCONNECTIONTARGET2 = "tcp-connection-target-2"
    static final private String TCPCONNECTIONTARGET3 = "tcp-connection-target-3"

    // Other namespace
    static final private String TEST_NAMESPACE = "qa-plop"

    static final private String SOCAT_DEBUG = "-d -d -v"

    // Target deployments
    @Shared
    final private List<Deployment> targetDeployments = [
            new Deployment()
                    .setName(TCPCONNECTIONTARGET1)
                    .setNamespace(TEST_NAMESPACE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addPort(80)
                    .addPort(8080)
                    .addLabel("app", TCPCONNECTIONTARGET1)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:80,fork STDOUT & " +
                                      "socat "+SOCAT_DEBUG+" TCP-LISTEN:8080,fork STDOUT)" as String,]),
            new Deployment()
                    .setName(TCPCONNECTIONTARGET2)
                    .setNamespace(TEST_NAMESPACE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addPort(8081, "TCP")
                    .addLabel("app", TCPCONNECTIONTARGET2)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:8081,fork STDOUT)" as String,]),
            new Deployment()
                    .setName(TCPCONNECTIONTARGET3)
                    .setNamespace(TEST_NAMESPACE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addPort(8082, "TCP")
                    .addLabel("app", TCPCONNECTIONTARGET3)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:8082,fork STDOUT & " +
                            "sleep 90 && pkill socat && sleep 3600)" as String,]),
                    // The 8082 port is opened. 90 seconds later the process is killed. After that we sleep forever
        ]

    def setupSpec() {
        // cleanup after a prior incomplete run
        destroyDeployments()
        destroyNamespace()
    }

    def cleanupSpec() {
        destroyDeployments()
        destroyNamespace()
    }

    def createDeployments() {
        // batchCreateDeployments() provisions the namespace as needed (see
        // ensureNamespaceExsists())
        orchestrator.batchCreateDeployments(targetDeployments)
        for (Deployment d : targetDeployments) {
            assert Services.waitForDeployment(d)
        }
    }

    def destroyDeployments() {
        for (Deployment deployment : targetDeployments) {
            if (orchestrator.getDeploymentId(deployment) != null) {
                orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
            }
        }
    }

    def destroyNamespace() {
        orchestrator.deleteNamespace(TEST_NAMESPACE)
        orchestrator.waitForNamespaceDeletion(TEST_NAMESPACE)
    }

    def "Verify networking endpoints with processes appear in API at the deployment level"() {
        given:
        // implicitly creates the namespace as needed
        createDeployments()

        String deploymentId1 = targetDeployments.find { it.name == TCPCONNECTIONTARGET1 }?.deploymentUid
        String deploymentId2 = targetDeployments.find { it.name == TCPCONNECTIONTARGET2 }?.deploymentUid

        def processesListeningOnPorts = waitForResponseToHaveNumElements(2, deploymentId1, 240)

        assert processesListeningOnPorts

        def list = processesListeningOnPorts.listeningEndpointsList
        assert list.size() == 2

        def endpoint1 = list[0]

        assert endpoint1
        assert endpoint1.deploymentId
        assert endpoint1.podId
        assert endpoint1.podUid
        assert endpoint1.containerName == TCPCONNECTIONTARGET1
        assert endpoint1.signal.name == "socat"
        assert endpoint1.signal.execFilePath == "/usr/bin/socat"
        assert endpoint1.signal.args == "-d -d -v TCP-LISTEN:80,fork STDOUT"

        def endpoint2 = list[1]

        assert endpoint2
        assert endpoint2.deploymentId
        assert endpoint2.podId
        assert endpoint2.podUid
        assert endpoint2.containerName == TCPCONNECTIONTARGET1
        assert endpoint2.signal.name == "socat"
        assert endpoint2.signal.execFilePath == "/usr/bin/socat"
        assert endpoint2.signal.args == "-d -d -v TCP-LISTEN:8080,fork STDOUT"

        processesListeningOnPorts = waitForResponseToHaveNumElements(1, deploymentId2, 240)

        assert processesListeningOnPorts

        list = processesListeningOnPorts.listeningEndpointsList
        assert list.size() == 1

        def endpoint = list.find { it.endpoint.port == 8081 }

        assert endpoint
        assert endpoint.deploymentId
        assert endpoint.podId
        assert endpoint.podUid
        assert endpoint.containerName == TCPCONNECTIONTARGET2
        assert endpoint.signal.name == "socat"
        assert endpoint.signal.execFilePath == "/usr/bin/socat"
        assert endpoint.signal.args == "-d -d -v TCP-LISTEN:8081,fork STDOUT"
    }

    def "Networking endpoints are no longer in the API when deployments are deleted"() {
        given:
        String deploymentId1 = targetDeployments.find { it.name == TCPCONNECTIONTARGET1 }?.deploymentUid
        String deploymentId2 = targetDeployments.find { it.name == TCPCONNECTIONTARGET2 }?.deploymentUid

        destroyDeployments()

        def processesListeningOnPorts = waitForResponseToHaveNumElements(0, deploymentId1, 240)

        assert processesListeningOnPorts

        def list2 = processesListeningOnPorts.listeningEndpointsList
        assert list2.size() == 0

        processesListeningOnPorts = waitForResponseToHaveNumElements(0, deploymentId2, 240)

        assert processesListeningOnPorts

        def list3 = processesListeningOnPorts.listeningEndpointsList
        assert list3.size() == 0
    }

    def "Verify networking endpoints disappear when process is terminated"() {
        given:
        createDeployments()

        String deploymentId3 = targetDeployments.find { it.name == TCPCONNECTIONTARGET3 }?.deploymentUid

        def processesListeningOnPorts = waitForResponseToHaveNumElements(1, deploymentId3, 240)

        // First check that the listening endpoint appears in the API
        assert processesListeningOnPorts

        def list = processesListeningOnPorts.listeningEndpointsList
        assert list.size() == 1

        def endpoint = list.find { it.endpoint.port == 8082 }

        assert endpoint
        assert endpoint.deploymentId
        assert endpoint.podId
        assert endpoint.podUid
        assert endpoint.containerName == TCPCONNECTIONTARGET3
        assert endpoint.signal.name == "socat"
        assert endpoint.signal.execFilePath == "/usr/bin/socat"
        assert endpoint.signal.args == "-d -d -v TCP-LISTEN:8082,fork STDOUT"

        // Allow enough time for the process and port to close and check that it is not in the API response
        processesListeningOnPorts = waitForResponseToHaveNumElements(0, deploymentId3, 180)

        assert processesListeningOnPorts

        destroyDeployments()
    }

    def "Verify networking endpoint doesn't disappear when port stays open"() {
        given:
        createDeployments()

        String deploymentId2 = targetDeployments.find { it.name == TCPCONNECTIONTARGET2 }?.deploymentUid

        def processesListeningOnPorts = waitForResponseToHaveNumElements(1, deploymentId2, 240)

        // First check that the listening endpoint appears in the API
        assert processesListeningOnPorts

        def list = processesListeningOnPorts.listeningEndpointsList
        assert list.size() == 1

        def endpoint = list.find { it.endpoint.port == 8081 }

        assert endpoint
        assert endpoint.deploymentId
        assert endpoint.podId
        assert endpoint.podUid
        assert endpoint.containerName == TCPCONNECTIONTARGET2
        assert endpoint.signal.name == "socat"
        assert endpoint.signal.execFilePath == "/usr/bin/socat"
        assert endpoint.signal.args == "-d -d -v TCP-LISTEN:8081,fork STDOUT"

        sleep 65000 // Sleep for 65 seconds
        processesListeningOnPorts = evaluateWithRetry(10, 10) {
            ProcessesListeningOnPortsService.getProcessesListeningOnPortsResponse(deploymentId2)
        }

        // Confirm that the listening endpoint still appears in the API 65 seconds later
        list = processesListeningOnPorts.listeningEndpointsList
        assert list.size() == 1

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
                return processesListeningOnPorts
            }
            sleep intervalSeconds * 1000
        }
        log.info "Timedout waiting for response to have {$numElements} elements"
        return null
    }
}
