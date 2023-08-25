package services

import io.stackrox.proto.api.v1.DeclarativeConfigHealthServiceGrpc
import io.stackrox.proto.api.v1.EmptyOuterClass.Empty

class DeclarativeConfigHealthService extends BaseService {
    static getDeclarativeConfigHealthClient() {
        return DeclarativeConfigHealthServiceGrpc.newBlockingStub(getChannel())
    }

    static getDeclarativeConfigHealthInfo() {
        return getDeclarativeConfigHealthClient().getDeclarativeConfigHealths(Empty.newBuilder().build())
    }
}
