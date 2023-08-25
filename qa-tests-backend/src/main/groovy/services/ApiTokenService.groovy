package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.APITokenServiceGrpc
import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenRequest
import io.stackrox.proto.api.v1.Common

@Slf4j
class ApiTokenService extends BaseService {
    static getApiTokenService() {
        return APITokenServiceGrpc.newBlockingStub(getChannel())
    }

    static generateToken(String name, String... roles) {
        try {
            GenerateTokenRequest.Builder request =
                    GenerateTokenRequest.newBuilder()
                            .setName(name)
                            .addAllRoles(Arrays.asList(roles))
            return getApiTokenService().generateToken(request.build())
        } catch (Exception e) {
            log.error("Failed to generate token", e)
        }
    }

    static revokeToken(String tokenId) {
        try {
            getApiTokenService().revokeToken(Common.ResourceByID.newBuilder()
                    .setId(tokenId)
                    .build())
        } catch (Exception e) {
            log.error("Failed to revoke token", e)
        }
    }
}
