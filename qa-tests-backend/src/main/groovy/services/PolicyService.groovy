package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.PolicyServiceGrpc
import io.stackrox.proto.api.v1.PolicyServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.PolicyOuterClass

@Slf4j
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

    static String createNewPolicy(PolicyOuterClass.Policy policy) {
        String policyID = ""

        try {
            policyID = getPolicyClient().postPolicy(
                    PolicyServiceOuterClass.PostPolicyRequest.newBuilder().
                            setPolicy(policy).
                            setEnableStrictValidation(true).
                            build()
            ).getId()
        } catch (Exception e) {
            log.error("error creating new policy", e)
        }

        return policyID
    }

    static PolicyOuterClass.Policy createAndFetchPolicy(PolicyOuterClass.Policy policy) {
        return getPolicyClient().postPolicy(
                PolicyServiceOuterClass.PostPolicyRequest.newBuilder().
                        setPolicy(policy).
                        setEnableStrictValidation(true).
                        build()
        )
    }

    static deletePolicy(String policyID) {
        try {
            getPolicyClient().deletePolicy(
                    Common.ResourceByID.newBuilder()
                            .setId(policyID)
                            .build()
            )
        } catch (Exception e) {
            log.error("error deleting policy", e)
        }
    }

    static PolicyServiceOuterClass.DryRunResponse dryRunPolicy(PolicyOuterClass.Policy policy) {
        return getPolicyClient().dryRunPolicy(policy)
    }

    static patchPolicy(PolicyServiceOuterClass.PatchPolicyRequest pr) {
        try {
            getPolicyClient().patchPolicy(pr).newBuilder().build()
        }
        catch (Exception e) {
            log.error("error patching policy", e)
        }
    }
}
