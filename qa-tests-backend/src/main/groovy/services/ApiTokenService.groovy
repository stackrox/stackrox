package services

import io.stackrox.proto.api.v1.APITokenServiceGrpc
import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenRequest
import io.stackrox.proto.api.v1.Common

class ApiTokenService extends BaseService {
    static getApiTokenService() {
        return APITokenServiceGrpc.newBlockingStub(getChannel())
    }

    static generateToken(String name, String role) {
        try {
            GenerateTokenRequest.Builder request =
                    GenerateTokenRequest.newBuilder()
                            .setName(name)
                            .setRole(role)
            return getApiTokenService().generateToken(request.build())
        } catch (Exception e) {
            println "Failed to generate token: ${e}"
        }
    }

    static revokeToken(String tokenId) {
        try {
            getApiTokenService().revokeToken(Common.ResourceByID.newBuilder()
                    .setId(tokenId)
                    .build())
        } catch (Exception e) {
            println "Failed to revoke token: ${e}"
        }
    }
}
