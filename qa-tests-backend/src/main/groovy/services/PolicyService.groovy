package services

import io.stackrox.proto.api.v1.PolicyServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.PolicyOuterClass

class PolicyService extends BaseService {
    static getPolicyClient() {
        return PolicyServiceGrpc.newBlockingStub(getChannel())
    }

    static reassessPolicies() {
        getPolicyClient().reassessPolicies(EMPTY)
    }

    static List<PolicyOuterClass.ListPolicy> getPolicies(RawQuery query = RawQuery.newBuilder().build()) {
        return getPolicyClient().listPolicies(query).policiesList
    }

}
