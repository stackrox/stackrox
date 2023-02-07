import objects.Deployment
import objects.K8sServiceAccount
import objects.Service
import services.ClusterService
import util.Env
import util.Helpers

import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Stepwise

import services.ProcessesListeningOnPortsService

@Stepwise
@IgnoreIf({ !Env.CI_JOBNAME.contains("postgres") })
class ProcessesListeningOnPortsTest extends BaseSpecification {

    // Deployment names
    static final private String TCPCONNECTIONTARGET1 = "tcp-connection-target-1"
    static final private String TCPCONNECTIONTARGET2 = "tcp-connection-target-2"

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
            orchestrator.deleteDeployment(deployment)
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
            sleep(5000)
            createDeployments()
            sleep(5000)
            log.info ">>>> Done <<<<<"
        }
    }

    @Tag("BAT")
    def "Verify networking endpoints with processes appear in API at the deployment level"() {
        given:
        "Two deployments that listen on ports are started up"

        rebuildForRetries()
        def clusterId = ClusterService.getClusterId()

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
        assert endpoint1.podUid
        assert endpoint1.containerName == TCPCONNECTIONTARGET1
        assert endpoint1.signal.id
        assert endpoint1.signal.containerId
        assert endpoint1.signal.time
        assert endpoint1.signal.name == "socat"
        assert endpoint1.signal.execFilePath == "/usr/bin/socat"
        // assert endpoint1.signal.args == "-d -d -v TCP-LISTEN:80,fork STDOUT"
        assert endpoint1.signal.pid
        assert endpoint1.clusterId == clusterId
        assert endpoint1.namespace
        assert endpoint1.containerStartTime
        assert endpoint1.imageId

        def endpoint2 = list.find { it.endpoint.port == 8080 }

        assert endpoint2
        assert endpoint2.clusterId == clusterId
        assert endpoint2.containerName == TCPCONNECTIONTARGET1
        assert endpoint2.signal.id
        assert endpoint2.signal.containerId
        assert endpoint2.signal.time
        assert endpoint2.signal.name == "socat"
        assert endpoint2.signal.execFilePath == "/usr/bin/socat"
        assert endpoint2.signal.args == "-d -d -v TCP-LISTEN:8080,fork STDOUT"
        assert endpoint2.signal.pid

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
        assert endpoint.clusterId == clusterId
        assert endpoint.containerName == TCPCONNECTIONTARGET2
        assert endpoint.signal.id
        assert endpoint.signal.containerId
        assert endpoint.signal.time
        assert endpoint.signal.name == "socat"
        assert endpoint.signal.execFilePath == "/usr/bin/socat"
        assert endpoint.signal.args == "-d -d -v TCP-LISTEN:8081,fork STDOUT"
        assert endpoint.signal.pid

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
