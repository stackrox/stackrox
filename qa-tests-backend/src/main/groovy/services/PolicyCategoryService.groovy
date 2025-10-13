package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v1.PolicyCategoryServiceGrpc
import io.stackrox.proto.api.v1.PolicyCategoryServiceOuterClass

@Slf4j
@CompileStatic
class PolicyCategoryService extends BaseService {
    static PolicyCategoryServiceGrpc.PolicyCategoryServiceBlockingStub getPolicyCategoryClient() {
        return PolicyCategoryServiceGrpc.newBlockingStub(getChannel())
    }

    static String createNewPolicyCategory(String name) {
        String categoryID = ""
        try {
            categoryID = getPolicyCategoryClient().postPolicyCategory(
                    PolicyCategoryServiceOuterClass.PostPolicyCategoryRequest.newBuilder().
                        setPolicyCategory(
                                PolicyCategoryServiceOuterClass.PolicyCategory.newBuilder().setName(name).build()).
                            build()
            ).getId()
        } catch (Exception e) {
            log.error("Error creating policy category", e)
        }

        return categoryID
    }

    static deletePolicyCategory(String id) {
        try {
            getPolicyCategoryClient().deletePolicyCategory(
                    PolicyCategoryServiceOuterClass.DeletePolicyCategoryRequest.
                            newBuilder().setId(id).build())
        } catch (Exception e) {
            log.error("Error deleting policy category", e)
        }
    }
}
