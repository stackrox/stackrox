import static com.jayway.restassured.RestAssured.given

import groups.BAT
import groups.Integration
import objects.Deployment
import objects.K8sServiceAccount
import objects.Service
import util.Env
import util.Helpers

import org.junit.experimental.categories.Category
import spock.lang.Shared
import spock.lang.Stepwise

import services.ProcessesListeningOnPortsService

@Stepwise
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

    @Category([BAT, Integration])
    def "Verify networking endpoints with processes appear in API at the namespace level"() {
        given:
        "Two deployments that listen on ports are started up"
        rebuildForRetries()

        String namespace = targetDeployments[0].getNamespace()
        String deploymentId = targetDeployments[0].getDeploymentUid()

        def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(namespace, deploymentId)
                return temp
        }

        namespace = targetDeployments[1].getNamespace()
        deploymentId = targetDeployments[1].getDeploymentUid()

        processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(namespace, deploymentId)
                return temp
        }

        def gotCorrectNumElements = waitForNamespaceResponseToHaveNumElements(2, namespace, 240)

        assert gotCorrectNumElements

        processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsWithDeploymentResponse(namespace)
                return temp
        }

        assert processesListeningOnPorts

        def list = processesListeningOnPorts.processesListeningOnPortsWithDeploymentList

        assert list.size() == 2

        def deploymentId1 = targetDeployments[0].getDeploymentUid()
        def deploymentId2 = targetDeployments[1].getDeploymentUid()

        def processesForDeployment1 = list.find { it.deploymentId == deploymentId1 }
        def processesForDeployment2 = list.find { it.deploymentId == deploymentId2 }

        def list1 = processesForDeployment1.processesListeningOnPortsList

        assert list1.size() == 2

        def endpoint1a = list1.find { it.port == 80 }

        assert endpoint1a
        assert endpoint1a.process.containerName == TCPCONNECTIONTARGET1
        assert endpoint1a.process.processName == "socat"
        assert endpoint1a.process.processExecFilePath == "/usr/bin/socat"
        // TODO This assert should be uncommented when the fix for it is merged into master
        // assert endpoint1a.process.processArgs == "-d -d -v TCP-LISTEN:80,fork STDOUT"

        def endpoint1b = list1.find { it.port == 8080 }

        assert endpoint1b
        assert endpoint1b.process.containerName == TCPCONNECTIONTARGET1
        assert endpoint1b.process.processName == "socat"
        assert endpoint1b.process.processExecFilePath == "/usr/bin/socat"
        assert endpoint1b.process.processArgs == "-d -d -v TCP-LISTEN:8080,fork STDOUT"

        def list2 = processesForDeployment2.processesListeningOnPortsList

        assert list2.size() == 1

        def endpoint2 = list2.get(0)

        assert endpoint2.port == 8081
        assert endpoint2.process.containerName == TCPCONNECTIONTARGET2
        assert endpoint2.process.processName == "socat"
        assert endpoint2.process.processExecFilePath == "/usr/bin/socat"
        assert endpoint2.process.processArgs == "-d -d -v TCP-LISTEN:8081,fork STDOUT"

        destroyDeployments()

        gotCorrectNumElements = waitForNamespaceResponseToHaveNumElements(0, namespace, 240)

        assert gotCorrectNumElements

        log.info "Destroyed deployment"

        processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsWithDeploymentResponse(namespace)
                return temp
        }

        def list3 = processesListeningOnPorts.processesListeningOnPortsWithDeploymentList

        assert list3.size() == 0
    }

    @Category([BAT, Integration])
    def "Verify networking endpoints with processes appear in API at the deployment level"() {
        given:
        "Two deployments that listen on ports are started up"
        rebuildForRetries()

        String namespace = targetDeployments[0].getNamespace()
        String deploymentId = targetDeployments[0].getDeploymentUid()

        def gotCorrectNumElements = waitForDeploymentResponseToHaveNumElements(2, namespace, deploymentId, 240)

        assert gotCorrectNumElements

        def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(namespace, deploymentId)
                return temp
        }

        assert processesListeningOnPorts

        def list = processesListeningOnPorts.processesListeningOnPortsList
        assert list.size() == 2

        def endpoint1 = list.find { it.port == 80 }

        assert endpoint1
        assert endpoint1.process.containerName == TCPCONNECTIONTARGET1
        assert endpoint1.process.processName == "socat"
        assert endpoint1.process.processExecFilePath == "/usr/bin/socat"
        // TODO This assert should be uncommented when the fix for it is merged into master
        // assert endpoint1.process.processArgs == "-d -d -v TCP-LISTEN:80,fork STDOUT"

        def endpoint2 = list.find { it.port == 8080 }

        assert endpoint2
        assert endpoint2.process.containerName == TCPCONNECTIONTARGET1
        assert endpoint2.process.processName == "socat"
        assert endpoint2.process.processExecFilePath == "/usr/bin/socat"
        assert endpoint2.process.processArgs == "-d -d -v TCP-LISTEN:8080,fork STDOUT"

        destroyDeployments()

        gotCorrectNumElements = waitForDeploymentResponseToHaveNumElements(0, namespace, deploymentId, 240)

        assert gotCorrectNumElements

        processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(namespace, deploymentId)
                return temp
        }

        assert processesListeningOnPorts

        def list2 = processesListeningOnPorts.processesListeningOnPortsList
        assert list2.size() == 0
    }

    private waitForNamespaceResponseToHaveNumElements(int numElements, String namespace, int timeoutSeconds = 240) {
        int intervalSeconds = 1
        int waitTime
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                    def temp = ProcessesListeningOnPortsService
                            .getProcessesListeningOnPortsWithDeploymentResponse(namespace)
                    return temp
            }

            def list = processesListeningOnPorts.processesListeningOnPortsWithDeploymentList

            if (list.size() == numElements) {
                return true
            }
            sleep intervalSeconds * 1000
        }
        log.info "Timedout waiting for response to have {$numElements} elements"
        return false
    }

    private waitForDeploymentResponseToHaveNumElements(int numElements, String namespace,
        String deploymentId, int timeoutSeconds = 240) {

        int intervalSeconds = 1
        int waitTime
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                    def temp = ProcessesListeningOnPortsService
                            .getProcessesListeningOnPortsResponse(namespace, deploymentId)
                    return temp
            }

            def list = processesListeningOnPorts.processesListeningOnPortsList

            if (list.size() == numElements) {
                return true
            }
            sleep intervalSeconds * 1000
        }
        log.info "Timedout waiting for response to have {$numElements} elements"
        return false
    }
}
