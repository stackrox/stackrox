package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.DeploymentServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery

@CompileStatic
class DeploymentService extends BaseService {
    static DeploymentServiceGrpc.DeploymentServiceBlockingStub getDeploymentService() {
        return DeploymentServiceGrpc.newBlockingStub(getChannel())
    }

    static listDeployments() {
        return getDeploymentService().listDeployments(null).getDeploymentsList()
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

    static getDeploymentWithRisk(String id) {
        return getDeploymentService().getDeploymentWithRisk(getResourceByID(id))
    }

    static getDeploymentCount(RawQuery query = RawQuery.newBuilder().build()) {
        return getDeploymentService().countDeployments(query).count
    }
}
