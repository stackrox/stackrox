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

    // Deployment names
    static final private String UDPCONNECTIONTARGET = "udp-connection-target"
    static final private String TCPCONNECTIONTARGET1 = "tcp-connection-target-1"
    static final private String TCPCONNECTIONTARGET2 = "tcp-connection-target-2"
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
            //This was changed from tcp to udp. TODO: Test udp
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
            sleep(100000)
            log.info ">>>> Done <<<<<"
        }
    }

    @Category([BAT, Integration])
    def "Verify networking endpoints with processes appear in API at the namespace level"() {
        given:
        "An nginx pod is started"
        rebuildForRetries()

        String namespace = targetDeployments[0].getNamespace()
        String deploymentId = targetDeployments[0].getDeploymentUid()
        String name = targetDeployments[0].getName()
        def processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(namespace, deploymentId)
                return temp
        }
        log.info ""
        log.info ""
        log.info "*************"
        log.info "deploymentId= ${deploymentId}"
        log.info "name= ${name}"
        log.info "namespace= ${namespace}"
        log.info "${processesListeningOnPorts}"
        log.info "*************"
        log.info ""
        log.info ""

        namespace = targetDeployments[1].getNamespace()
        deploymentId = targetDeployments[1].getDeploymentUid()
        name = targetDeployments[1].getName()
        processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsResponse(namespace, deploymentId)
                return temp
        }
        log.info ""
        log.info ""
        log.info "*************"
        log.info "deploymentId= ${deploymentId}"
        log.info "name= ${name}"
        log.info "namespace= ${namespace}"
        log.info "${processesListeningOnPorts}"
        log.info "*************"
        log.info ""
        log.info ""


        processesListeningOnPorts = evaluateWithRetry(10, 10) {
                def temp = ProcessesListeningOnPortsService
                        .getProcessesListeningOnPortsWithDeploymentResponse(namespace)
                return temp
        }

        log.info ""
        log.info ""
        log.info "*************"
        log.info "${processesListeningOnPorts}"
        log.info "*************"
        log.info ""
        log.info ""

        assert processesListeningOnPorts

        def list = processesListeningOnPorts.processesListeningOnPortsWithDeploymentList

        assert list.size() == 2

	def deploymentId1 = targetDeployments[0].getDeploymentUid()
	def deploymentId2 = targetDeployments[1].getDeploymentUid()

	def processesForDeployment1 = list.find { it.deploymentId == deploymentId1 }
	def processesForDeployment2 = list.find { it.deploymentId == deploymentId2 }

        log.info "${processesForDeployment1}"
        log.info "${processesForDeployment2}"

        def list1 = processesForDeployment1.processesListeningOnPortsList

        assert list1.size() == 2

        def endpoint1_1 = list1.find { it.port == 80 }

        assert endpoint1_1
	assert endpoint1_1.process.containerName == TCPCONNECTIONTARGET1 
	assert endpoint1_1.process.processName == "socat"
	assert endpoint1_1.process.processExecFilePath == "/usr/bin/socat"
	// assert endpoint1_1.process.processArgs == "-d -d -v TCP-LISTEN:80,fork STDOUT"

	def endpoint1_2 = list1.find { it.port == 8080 }

        assert endpoint1_2
	assert endpoint1_2.process.containerName == TCPCONNECTIONTARGET1 
	assert endpoint1_2.process.processName == "socat"
	assert endpoint1_2.process.processExecFilePath == "/usr/bin/socat"
	assert endpoint1_2.process.processArgs == "-d -d -v TCP-LISTEN:8080,fork STDOUT"

        def list2 = processesForDeployment2.processesListeningOnPortsList

	assert list2.size() == 1

	def endpoint2 = list2.get(0)

        assert endpoint2.port == 8081
	assert endpoint2.process.containerName == TCPCONNECTIONTARGET2
	assert endpoint2.process.processName == "socat"
	assert endpoint2.process.processExecFilePath == "/usr/bin/socat"
	assert endpoint2.process.processArgs == "-d -d -v TCP-LISTEN:8081,fork STDOUT"
    }

    @Category([BAT, Integration])
    def "Verify networking endpoints with processes appear in API at the deployment level"() {
        given:
        "An nginx pod is started"
        rebuildForRetries()

        String namespace = targetDeployments[0].getNamespace()
        String deploymentId = targetDeployments[0].getDeploymentUid()
        String name = targetDeployments[0].getName()
        log.info ""
        log.info ""
        log.info "*************"
        log.info "deploymentId= ${deploymentId}"
        log.info "name= ${name}"
        log.info "namespace= ${namespace}"
        log.info "*************"
        log.info ""
        log.info ""
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
	// assert endpoint1.process.processArgs == "-d -d -v TCP-LISTEN:80,fork STDOUT"

	def endpoint2 = list.find { it.port == 8080 }

        assert endpoint2
	assert endpoint2.process.containerName == TCPCONNECTIONTARGET1
	assert endpoint2.process.processName == "socat"
	assert endpoint2.process.processExecFilePath == "/usr/bin/socat"
	assert endpoint2.process.processArgs == "-d -d -v TCP-LISTEN:8080,fork STDOUT"
    }
}
