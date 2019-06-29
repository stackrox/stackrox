package services

import io.stackrox.proto.api.v1.DeploymentServiceGrpc

class DeploymentService extends BaseService {
    static getDeploymentService() {
        return DeploymentServiceGrpc.newBlockingStub(getChannel())
    }

    static listDeployments() {
        return getDeploymentService().listDeployments().getDeploymentsList()
    }
}

