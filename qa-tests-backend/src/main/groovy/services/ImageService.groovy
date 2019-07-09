package services

import io.stackrox.proto.api.v1.ImageServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery

class ImageService extends BaseService {
    static getImageClient() {
        return ImageServiceGrpc.newBlockingStub(getChannel())
    }

    static getImages(RawQuery request = RawQuery.newBuilder().build()) {
        return getImageClient().listImages(request).imagesList
    }

    static clearImageCaches() {
        getImageClient().invalidateScanAndRegistryCaches()
    }
}
