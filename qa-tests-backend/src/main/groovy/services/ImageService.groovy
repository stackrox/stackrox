package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.EmptyOuterClass
import io.stackrox.proto.api.v1.ImageServiceGrpc
import io.stackrox.proto.api.v1.ImageServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.ImageOuterClass

@Slf4j
class ImageService extends BaseService {
    static ImageServiceGrpc.ImageServiceBlockingStub getImageClient() {
        return ImageServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ImageOuterClass.ListImage> getImages(RawQuery request = RawQuery.newBuilder().build()) {
        return getImageClient().listImages(request).imagesList
    }

    static getImage(String digest, Boolean includeSnoozed = true) {
        if (digest == null) {
            return null
        }
        return getImageClient().getImage(
                ImageServiceOuterClass.GetImageRequest.newBuilder()
                        .setId(digest)
                        .setIncludeSnoozed(includeSnoozed)
                        .build()
        )
    }

    static clearImageCaches() {
        getImageClient().invalidateScanAndRegistryCaches(EmptyOuterClass.Empty.newBuilder().build())
    }

    static scanImage(String image, Boolean includeSnoozed = true) {
        try {
            return getImageClient().scanImage(ImageServiceOuterClass.ScanImageRequest.newBuilder()
                    .setImageName(image)
                    .setIncludeSnoozed(includeSnoozed)
                    .build())
        } catch (Exception e) {
            log.error("Image failed to scan: ${image}", e)
        }
    }

    static ImageServiceOuterClass.DeleteImagesResponse deleteImages(
            RawQuery query = RawQuery.newBuilder().build(), Boolean confirm = false
    ) {
        ImageServiceOuterClass.DeleteImagesResponse response = getImageClient()
                .deleteImages(ImageServiceOuterClass.DeleteImagesRequest.newBuilder()
                        .setQuery(query)
                        .setConfirm(confirm).build())
        log.debug "Deleted ${response.numDeleted} images based on ${query.query}"
        return response
    }

    static deleteImagesWithRetry(RawQuery query, Boolean confirm = false, Integer expectedDeletions = 1) {
        Integer deletedCount = 0
        withRetry(5, 2) {
            ImageServiceOuterClass.DeleteImagesResponse response = deleteImages(query, confirm)
            deletedCount += response.numDeleted
            if (deletedCount < expectedDeletions) {
                throw new RuntimeException("The number of images deleted has yet to reach its expected count. " +
                        deletedCount + " -v- " + expectedDeletions)
            }
        }
        log.debug "Deleted at least as many images as expected based on ${query.query}. " +
                deletedCount + " -v- " + expectedDeletions
    }
}
