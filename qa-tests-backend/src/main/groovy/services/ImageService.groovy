package services

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

    static getImage(String digest, Boolean includeSnoozed = true) {
        if (digest == null) {
            ImageOuterClass.Image nullImage
            return nullImage
        }
        return getImageClient().getImage(
                ImageServiceOuterClass.GetImageRequest.newBuilder()
                        .setId(digest)
                        .setIncludeSnoozed(includeSnoozed)
                        .build()
        )
    }

    static clearImageCaches() {
        getImageClient().invalidateScanAndRegistryCaches()
    }

    static scanImage(String image, Boolean includeSnoozed = true) {
        try {
            return getImageClient().scanImage(ImageServiceOuterClass.ScanImageRequest.newBuilder()
                    .setImageName(image)
                    .setIncludeSnoozed(includeSnoozed)
                    .build())
        } catch (Exception e) {
            println "Image failed to scan: ${image} - ${e.toString()}"
            return ""
        }
    }

    static deleteImages(RawQuery query = RawQuery.newBuilder().build(), Boolean confirm = false) {
        ImageServiceOuterClass.DeleteImagesResponse response = getImageClient()
                .deleteImages(ImageServiceOuterClass.DeleteImagesRequest.newBuilder()
                        .setQuery(query)
                        .setConfirm(confirm).build())
        println "Deleted ${response.numDeleted} images based on ${query.query}"
    }
}
