package services

import io.stackrox.proto.api.v1.DeploymentServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery

class DeploymentService extends BaseService {
    static getDeploymentService() {
        return DeploymentServiceGrpc.newBlockingStub(getChannel())
    }

    static listDeployments() {
        return getDeploymentService().listDeployments().getDeploymentsList()
    }

    static listDeploymentsSearch(RawQuery query = RawQuery.newBuilder().build()) {
        return getDeploymentService().listDeployments(query)
    }

    static listDeploymentsWithProcessInfo(RawQuery query = RawQuery.newBuilder().build()) {
        return getDeploymentService().listDeploymentsWithProcessInfo(query)
    }

    static getDeployment(String id) {
        return getDeploymentService().getDeployment(getResourceByID(id))
    }

    static getDeploymentCount(RawQuery query = RawQuery.newBuilder().build()) {
        return getDeploymentService().countDeployments(query).count
    }
}
