package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.AuthServiceGrpc
import io.stackrox.proto.api.v1.AuthServiceOuterClass

@CompileStatic
class AuthService extends BaseService {
    static AuthServiceGrpc.AuthServiceBlockingStub getAuthService() {
        return AuthServiceGrpc.newBlockingStub(getChannel())
    }
    static AuthServiceOuterClass.AuthStatus getAuthStatus() {
        return getAuthService().getAuthStatus(EMPTY)
    }
}
