import io.grpc.netty.GrpcSslContexts
import io.grpc.netty.NegotiationType
import io.grpc.netty.NettyChannelBuilder

import io.netty.handler.ssl.SslContext
import io.netty.handler.ssl.util.InsecureTrustManagerFactory

import stackrox.generated.AlertServiceGrpc
import stackrox.generated.AlertServiceOuterClass.ListAlert
import stackrox.generated.DeploymentServiceGrpc
import stackrox.generated.PolicyServiceGrpc
import stackrox.generated.PolicyServiceOuterClass.ListPolicy
import stackrox.generated.PolicyServiceOuterClass.Policy
import stackrox.generated.SearchServiceOuterClass.RawQuery
import stackrox.generated.AlertServiceOuterClass.Alert
import stackrox.generated.AlertServiceOuterClass.ListAlertsRequest
import stackrox.generated.DeploymentServiceOuterClass.ListDeployment
import stackrox.generated.DeploymentServiceOuterClass.Deployment
import stackrox.generated.Common.ResourceByID

class Services {

    static getChannel() {
        SslContext sslContext = GrpcSslContexts
                .forClient()
                .trustManager(InsecureTrustManagerFactory.INSTANCE)
                .build()

        Integer port = Integer.parseInt(System.getenv("PORT"))

        def channel = NettyChannelBuilder
                .forAddress(System.getenv("HOSTNAME"), port)
                .negotiationType(NegotiationType.TLS)
                .sslContext(sslContext)
                .build()
        return channel
    }

    static ResourceByID getResourceByID(String id) {
        return ResourceByID.newBuilder().setId(id).build()
    }

    static getPolicyClient() {
        return PolicyServiceGrpc.newBlockingStub(getChannel())
    }

    static getAlertClient() {
        return AlertServiceGrpc.newBlockingStub(getChannel())
    }

    static getDeploymentClient() {
        return DeploymentServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ListPolicy> getPolicies(RawQuery query = RawQuery.newBuilder().build()) {
        return getPolicyClient().listPolicies(query).policiesList
    }

    static Policy getPolicy(String id) {
        return getPolicyClient().getPolicy(getResourceByID(id))
    }

    static List<ListAlert> getViolations(ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().listAlerts(request).alertsList
    }

    static Alert getViolaton(String id) {
        return getAlertClient().getAlert(getResourceByID(id))
    }

    static List<ListDeployment> getDeployments(RawQuery query = RawQuery.newBuilder().build()) {
        return getDeploymentClient().listDeployments(query).deploymentsList
    }

    static Deployment getDeployment(String id) {
        return getDeploymentClient().getDeployment(getResourceByID(id))
    }

    static waitForViolation(String deploymentName, String policyName, Integer timeoutSeconds) {
        int intervalSeconds = 1
        for (int i = 0; i < timeoutSeconds / intervalSeconds; i++) {
            try {
                def violations = getViolations(ListAlertsRequest.newBuilder()
                        .setQuery("Deployment Name:${deploymentName}+Policy Name:${policyName}").build())
                if (violations.size() == 1) {
                    return true
                }
            } catch (Exception e) {
                println e
            } finally {
                sleep(intervalSeconds * 1000)
            }
        }
        return false
    }

    static boolean waitForDeployment(String name, Integer timeoutSeconds = 5) {
        int intervalSeconds = 1
        for (int i = 0; i < timeoutSeconds / intervalSeconds; i++) {
            try {
                def deployments = getDeployments(RawQuery.newBuilder().setQuery("Deployment Name:${name}").build())
                if (deployments.size() == 1) {
                    return true
                }
            } finally {
                sleep(intervalSeconds * 1000)
            }
        }
        return false
    }

}
