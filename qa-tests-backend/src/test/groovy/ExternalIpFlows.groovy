import static io.restassured.RestAssured.given
import static util.Helpers.withRetry

import io.grpc.StatusRuntimeException
import io.restassured.response.Response
import orchestratormanager.OrchestratorTypes
import org.yaml.snakeyaml.Yaml

import common.Constants
import objects.DaemonSet
import objects.Deployment
import objects.Edge
import objects.K8sServiceAccount
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Service
import services.ClusterService
import services.DeploymentService
import services.NetworkGraphService
import services.NetworkPolicyService
import util.CollectorUtil
import util.Env
import util.Helpers
import util.NetworkGraphUtil
import util.Timer

import org.junit.Assume
import spock.lang.Ignore
import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Stepwise
import spock.lang.Tag
import spock.lang.Unroll

@Stepwise
@Tag("PZ")
class ExternalIpFlowsTest extends BaseSpecification {

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
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addPort(8080, "UDP")
                    .addLabel("app", UDPCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["socat "+SOCAT_DEBUG+" UDP-RECV:8080 STDOUT",]),
            new Deployment()
                    .setName(TCPCONNECTIONTARGET)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addPort(80)
                    .addPort(8080)
                    .addLabel("app", TCPCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:80,fork STDOUT & " +
                                      "socat "+SOCAT_DEBUG+" TCP-LISTEN:8080,fork STDOUT)" as String,]),
            new Deployment()
                    .setName(NGINXCONNECTIONTARGET)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
                    .addPort(80)
                    .addLabel("app", NGINXCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCreateLoadBalancer(!
                        (Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x"))
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
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
                    .addLabel("app", NOCONNECTIONSOURCE),
            new Deployment()
                    .setName(SHORTCONSISTENTSOURCE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
                    .addLabel("app", SHORTCONSISTENTSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS}; " +
                                      "do wget -S -T 2 http://${NGINXCONNECTIONTARGET}; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(SINGLECONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
                    .addLabel("app", SINGLECONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["wget -S -T 2 http://${NGINXCONNECTIONTARGET} && " +
                                      "while sleep 30; do echo hello; done" as String,]),
            new Deployment()
                    .setName(UDPCONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addLabel("app", UDPCONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${UDPCONNECTIONSOURCE}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN UDP:${UDPCONNECTIONTARGET}:8080; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(TCPCONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addLabel("app", TCPCONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${TCPCONNECTIONSOURCE}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(MULTIPLEPORTSCONNECTION)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
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
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
                    .addLabel("app", EXTERNALDESTINATION)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS}; " +
                                      "do wget -S -T 2 http://www.google.com; " +
                                      "done" as String,]),
            new Deployment()
                    .setName("${TCPCONNECTIONSOURCE}-qa2")
                    .setNamespace(OTHER_NAMESPACE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
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
    }

    def setupSpec() {
        CollectorUtil.enableExternalIps()
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
        CollectorUtil.disableExternalIps()
    }

    @Tag("NetworkFlowVisualization")
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify external network flow inspection"() {
        when:
        "Deployment A, where A communicates to an external target"
        String deploymentUid = deployments.find { it.name == EXTERNALDESTINATION }?.deploymentUid
        assert deploymentUid != null

        and:
        "Check for external flow"

        def extIp
        // retrying the first case, to wait for the network flows to appear,
        // then subsequent test cases are querying the same data in different ways
        withRetry(10, 30) {
            log.info "Checking for flow from ${EXTERNALDESTINATION} to ${NGINXCONNECTIONTARGET}"
            def response = NetworkGraphService.getExternalNetworkFlows(
                "Namespace:qa+DeploymentName:${EXTERNALDESTINATION}"
            )

            assert response.getFlowsList()?.size() == 1

            // retrieve the IP. We can use this to filter later
            extIp = response.getFlows(0)?.getProps()?.getDstEntity()?.getExternalSource()?.getCidr()
            assert extIp != null
        }

        // construct some subnets to use to verify filtering of
        // expected external entity
        def octets = extIp.tokenize(".")
        def widerNet = "${octets[0]}.${octets[1]}.0.0/16"
        def narrowerNet = "${octets[0]}.${octets[1]}.${octets[2]}.254/31"

        and: "Get no flows with CIDR filter"
        def cidrFilteredFlows = NetworkGraphService.getExternalNetworkFlows(
            "Namespace:qa+External Source Address:123.123.123.0/24"
        )

        and: "Get flows with wide subnet filter"
        def widerNetFlows = NetworkGraphService.getExternalNetworkFlows(
            "Namespace:qa+External Source Address:${widerNet}"
        )

        and: "Get flows with narrow subnet filter"
        def narrowerNetFlows = NetworkGraphService.getExternalNetworkFlows(
            "Namespace:qa+External Source Address:${narrowerNet}"
        )

        and: "Get flows for non-existent namespace"
        def noNamespaceFlows = NetworkGraphService.getExternalNetworkFlows(
            "Namespace:empty")

        then:
        assert cidrFilteredFlows?.getFlowsList().size() == 0

        and:
        assert widerNetFlows?.getFlowsList().size() == 1
        assert widerNetFlows.getFlows(0)?.getProps()?.getDstEntity()?.getExternalSource()?.getCidr() == extIp

        and:
        assert narrowerNetFlows?.getFlowsList().size() == 0

        and:
        assert noNamespaceFlows?.getFlowsList().size() == 0
    }
}
