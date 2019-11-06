package services

import io.stackrox.proto.api.v1.AuthServiceGrpc

class AuthService extends BaseService {
    static getAuthService() {
        return AuthServiceGrpc.newBlockingStub(getChannel())
    }
    static getAuthStatus() {
        return getAuthService().getAuthStatus(EMPTY)
    }
}
