package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.ImageServiceGrpc
import io.stackrox.proto.api.v1.ImageServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.ImageOuterClass

class ImageService extends BaseService {
    static getImageClient() {
        return ImageServiceGrpc.newBlockingStub(getChannel())
    }

    static getImages(RawQuery request = RawQuery.newBuilder().build()) {
        return getImageClient().listImages(request).imagesList
    }

    static getImage(String digest) {
        if (digest == null) {
            ImageOuterClass.Image nullImage
            return nullImage
        }
        return getImageClient().getImage(Common.ResourceByID.newBuilder().setId(digest).build())
    }

    static clearImageCaches() {
        getImageClient().invalidateScanAndRegistryCaches()
    }

    static scanImage(String image) {
        try {
            return getImageClient().scanImage(ImageServiceOuterClass.ScanImageRequest.newBuilder()
                    .setImageName(image)
                    .build())
        } catch (Exception e) {
            println "Image failed to scan: ${image} - ${e.toString()}"
            return ""
        }
    }
}
