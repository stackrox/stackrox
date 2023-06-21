package services

import com.google.protobuf.Duration

import io.stackrox.proto.api.v1.CveService
import io.stackrox.proto.api.v1.ImageCVEServiceGrpc

class CVEService extends BaseService {

    static getImageCVEClient() {
        return ImageCVEServiceGrpc.newBlockingStub(getChannel())
    }

    static suppressImageCVE(String cve) {
        return getImageCVEClient().suppressCVEs(CveService.SuppressCVERequest.newBuilder()
                .addCves(cve)
                .setDuration(Duration.newBuilder().setSeconds(1000).build())
                .build())
    }

    static unsuppressImageCVE(String cve) {
        return getImageCVEClient().unsuppressCVEs(CveService.UnsuppressCVERequest.newBuilder().addCves(cve).build())
    }
}
