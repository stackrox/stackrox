package services

import io.stackrox.proto.api.v1.AuthProviderServiceGrpc
import io.stackrox.proto.api.v1.AuthproviderService

class AuthProviderService extends BaseService {
    static getAuthProviderService() {
        return AuthProviderServiceGrpc.newBlockingStub(getChannel())
    }

    static getAuthProviders() {
        return getAuthProviderService().getAuthProviders(
                AuthproviderService.GetAuthProvidersRequest.newBuilder().build()
        )
    }
}
