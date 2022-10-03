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
                .addCves(cve)
                .setDuration(Duration.newBuilder().setSeconds(1000).build())
                .build())
    }

    static unsuppressCVE(String cve) {
        return getCVEClient().unsuppressCVEs(CveService.UnsuppressCVERequest.newBuilder().addCves(cve).build())
    }

    static suppressImageCVE(String cve) {
        if (! Env.CI_JOBNAME.contains("postgres")) {
            return suppressCVE(cve)
        }
        return getImageCVEClient().suppressCVEs(CveService.SuppressCVERequest.newBuilder()
                .addCves(cve)
                .setDuration(Duration.newBuilder().setSeconds(1000).build())
                .build())
    }

    static unsuppressImageCVE(String cve) {
        if (! Env.CI_JOBNAME.contains("postgres")) {
            return unsuppressCVE(cve)
        }
        return getImageCVEClient().unsuppressCVEs(CveService.UnsuppressCVERequest.newBuilder().addCves(cve).build())
    }
}
