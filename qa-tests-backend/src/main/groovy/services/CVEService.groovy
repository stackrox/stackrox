package services

import com.google.protobuf.Duration
import io.stackrox.proto.api.v1.CVEServiceGrpc
import io.stackrox.proto.api.v1.CveService

class CVEService extends BaseService {
    static getCVEClient() {
        return CVEServiceGrpc.newBlockingStub(getChannel())
    }

    static suppressCVE(String cve) {
        return getCVEClient().suppressCVEs(CveService.SuppressCVERequest.newBuilder()
                .addIds(cve)
                .setDuration(Duration.newBuilder().setSeconds(1000).build())
                .build())
    }

    static unsuppressCVE(String cve) {
        return getCVEClient().unsuppressCVEs(CveService.UnsuppressCVERequest.newBuilder().addIds(cve).build())
    }
}
