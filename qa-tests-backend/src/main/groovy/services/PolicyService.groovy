package services

import io.stackrox.proto.api.v1.EmptyOuterClass
import io.stackrox.proto.api.v1.PolicyServiceGrpc

class PolicyService extends BaseService {
    static getPolicyClient() {
        return PolicyServiceGrpc.newBlockingStub(getChannel())
    }

    static reassessPolicies() {
        getPolicyClient().reassessPolicies(EmptyOuterClass.Empty.newBuilder().build())
    }
}
