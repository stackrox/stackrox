package services

import io.stackrox.proto.api.v1.AuthServiceGrpc
import io.stackrox.proto.api.v1.EmptyOuterClass

class AuthService extends BaseService {
    static getAuthService() {
        return AuthServiceGrpc.newBlockingStub(getChannel())
    }
    static getAuthStatus() {
        return getAuthService().getAuthStatus(EmptyOuterClass.Empty.newBuilder().build())
    }
}
