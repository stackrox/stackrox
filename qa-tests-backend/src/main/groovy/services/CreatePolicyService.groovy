package services

import stackrox.generated.Common
import stackrox.generated.PolicyServiceGrpc
import stackrox.generated.PolicyServiceOuterClass.Policy

class CreatePolicyService extends BaseService {

    static getPolicyClient() {
        return PolicyServiceGrpc.newBlockingStub(getChannel())
    }

    static String createNewPolicy(Policy policy) {
        String policyID = ""

        try {
            policyID = getPolicyClient().postPolicy(policy).getId()
        } catch (Exception e) {
            println e.toString()
        }

        return policyID
    }

    static deletePolicy(String policyID) {
        try {
            getPolicyClient().deletePolicy(
                    Common.ResourceByID.newBuilder()
                            .setId(policyID)
                            .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }
}
