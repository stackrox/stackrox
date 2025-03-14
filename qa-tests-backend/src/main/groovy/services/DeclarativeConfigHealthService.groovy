package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.DeclarativeConfigHealthServiceGrpc
import io.stackrox.proto.api.v1.EmptyOuterClass.Empty

@CompileStatic
class DeclarativeConfigHealthService extends BaseService {
    static DeclarativeConfigHealthServiceGrpc.DeclarativeConfigHealthServiceBlockingStub getDeclarativeConfigHealthClient() {
        return DeclarativeConfigHealthServiceGrpc.newBlockingStub(getChannel())
    }

    static getDeclarativeConfigHealthInfo() {
        return getDeclarativeConfigHealthClient().getDeclarativeConfigHealths(Empty.newBuilder().build())
    }
}
