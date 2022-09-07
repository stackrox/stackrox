package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.DebugServiceGrpc
import io.stackrox.proto.api.v1.DebugServiceOuterClass

@CompileStatic
class DebugService extends BaseService {
    static DebugServiceGrpc.DebugServiceBlockingStub getDebugService() {
        return DebugServiceGrpc.newBlockingStub(getChannel())
    }

    static setLogLevel(String level) {
        getDebugService().setLogLevel(DebugServiceOuterClass.LogLevelRequest.newBuilder().setLevel(level).build())
    }
}
