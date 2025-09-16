package services

import static io.stackrox.proto.api.v1.DeclarativeConfigHealthServiceGrpc.DeclarativeConfigHealthServiceBlockingStub
import static io.stackrox.proto.api.v1.DeclarativeConfigHealthServiceGrpc.newBlockingStub

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.EmptyOuterClass.Empty

@CompileStatic
class DeclarativeConfigHealthService extends BaseService {
    static DeclarativeConfigHealthServiceBlockingStub getDeclarativeConfigHealthClient() {
        return newBlockingStub(getChannel())
    }

    static getDeclarativeConfigHealthInfo() {
        return getDeclarativeConfigHealthClient().getDeclarativeConfigHealths(Empty.newBuilder().build())
    }
}
