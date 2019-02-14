package services

import io.stackrox.proto.api.v1.ImageServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass

class ImageService extends BaseService {
    static getImageClient() {
        return ImageServiceGrpc.newBlockingStub(getChannel())
    }

    static getImages() {
        return getImageClient().listImages(SearchServiceOuterClass.RawQuery.newBuilder().build()).imagesList
    }

    static clearImageCaches() {
        getImageClient().invalidateScanAndRegistryCaches()
    }
}
