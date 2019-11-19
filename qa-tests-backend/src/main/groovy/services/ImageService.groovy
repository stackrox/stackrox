package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.ImageServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery

class ImageService extends BaseService {
    static getImageClient() {
        return ImageServiceGrpc.newBlockingStub(getChannel())
    }

    static getImages(RawQuery request = RawQuery.newBuilder().build()) {
        return getImageClient().listImages(request).imagesList
    }

    static getImage(String digest) {
        return getImageClient().getImage(Common.ResourceByID.newBuilder().setId(digest).build())
    }

    static clearImageCaches() {
        getImageClient().invalidateScanAndRegistryCaches()
    }
}
