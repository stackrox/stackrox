package services

import com.google.protobuf.Duration
import io.stackrox.proto.api.v1.CVEServiceGrpc
import io.stackrox.proto.api.v1.CveService
import io.stackrox.proto.api.v1.ImageCVEServiceGrpc

import util.Env

class CVEService extends BaseService {
    static getCVEClient() {
        return CVEServiceGrpc.newBlockingStub(getChannel())
    }

    static getImageCVEClient() {
        return ImageCVEServiceGrpc.newBlockingStub(getChannel())
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

    static suppressImageCVE(String cve) {
        if (Env.CI_JOBNAME.contains("postgres")) {
            return getImageCVEClient().suppressCVEs(CveService.SuppressCVERequest.newBuilder()
                    .addIds(cve)
                    .setDuration(Duration.newBuilder().setSeconds(1000).build())
                    .build())
        }
        suppressCVE(cve)
    }

    static unsuppressImageCVE(String cve) {
        if (Env.CI_JOBNAME.contains("postgres")) {
            return getImageCVEClient().unsuppressCVEs(CveService.UnsuppressCVERequest.newBuilder().addIds(cve).build())
        }
        unsuppressCVE(cve)
    }
}
