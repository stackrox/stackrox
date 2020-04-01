package services

import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.ImageIntegrationServiceGrpc
import io.stackrox.proto.api.v1.ImageIntegrationServiceOuterClass
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import io.stackrox.proto.storage.ImageIntegrationOuterClass.ImageIntegrationCategory
import util.Timer

class ImageIntegrationService extends BaseService {
    static final private String AUTO_REGISTERED_SCANNER_INTEGRATION = "Stackrox Scanner"

    static getImageIntegrationClient() {
        return ImageIntegrationServiceGrpc.newBlockingStub(getChannel())
    }

    static createImageIntegration(ImageIntegrationOuterClass.ImageIntegration integration) {
        try {
            getImageIntegrationClient().testImageIntegration(integration)
        } catch (Exception e) {
            println "Integration test failed: ${integration.name}: ${e.message}"
            return ""
        }

        Timer t = new Timer(15, 3)
        while (t.IsValid()) {
            try {
                ImageIntegrationOuterClass.ImageIntegration createdIntegration =
                        getImageIntegrationClient().postImageIntegration(integration)
                return createdIntegration.getId()
            } catch (Exception e) {
                println "Unable to create image integration ${integration.name}: ${e.message}"
            }
        }
        println "Unable to create image integration"
        return ""
    }

    static deleteImageIntegration(String integrationId) {
        try {
            getImageIntegrationClient().deleteImageIntegration(getResourceByID(integrationId))
        } catch (Exception e) {
            println "Failed to delete integration: ${e.toString()}"
            return false
        }
        try {
            ImageIntegrationOuterClass.ImageIntegration integration =
                    getImageIntegrationClient().getImageIntegration(getResourceByID(integrationId))
            while (integration) {
                sleep 2000
                integration = getImageIntegrationClient().getImageIntegration(getResourceByID(integrationId))
            }
        } catch (StatusRuntimeException e) {
            return e.status.code == Status.Code.NOT_FOUND
        }
    }

    /*
        Helper functions to simplify creating known integrations
    */

    // For now, we delete the auto registered StackRox Scanner integration
    // since the QA tests are not stable when we run with them.
    // This function returns whether or not the integration was deleted.
    static boolean deleteAutoRegisteredStackRoxScannerIntegrationIfExists() {
        try {
            // The Stackrox Scanner integration is auto-added by the product,
            // so we first check whether it already exists.
            def scannerIntegrations = getImageIntegrationClient().getImageIntegrations(
                    ImageIntegrationServiceOuterClass.GetImageIntegrationsRequest.
                            newBuilder().
                            setName(AUTO_REGISTERED_SCANNER_INTEGRATION).
                            build()
            )
            if (scannerIntegrations.integrationsCount > 1) {
                throw new RuntimeException("UNEXPECTED: Got more than one scanner integration: ${scannerIntegrations}")
            }
            if (scannerIntegrations.integrationsCount == 0) {
                return false
            }
            def id = scannerIntegrations.getIntegrations(0).id
            // Delete
            getImageIntegrationClient().deleteImageIntegration(getResourceByID(id))
            return true
        } catch (Exception e) {
            println "Unable to delete existing Stackrox scanner integration: ${e.message}"
            // return false since we are not sure if the integration exists and failed to delete, or did not exist
            return false
        }
    }

    // This function adds the Stackrox scanner integration.
    // Currently, we do this if it was deleted by the test on setup,
    // so that the test leaves things in the same state after cleanup.
    static String addStackroxScannerIntegration() {
        List<ImageIntegrationCategory> categories = [ImageIntegrationCategory.SCANNER]
        String endpoint = "https://scanner.stackrox:8080"
        String type = "clairify"

        ImageIntegrationOuterClass.ImageIntegration integration =
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                        .setName(AUTO_REGISTERED_SCANNER_INTEGRATION)
                        .setType(type)
                        .addAllCategories(categories)
                        .setClairify(ImageIntegrationOuterClass.ClairifyConfig.newBuilder()
                                .setEndpoint(endpoint))
                        .build()

        return createImageIntegration(integration)
    }

    static String addDockerTrustedRegistry(boolean includeScanner = true) {
        String name = "dtr"
        String type = "dtr"
        List<ImageIntegrationCategory> categories = getIntegrationCategories(includeScanner)
        String username = "qa"
        String password = System.getenv("DTR_REGISTRY_PASSWORD")
        String endpoint = "https://apollo-dtr.rox.systems/"

        ImageIntegrationOuterClass.ImageIntegration integration =
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                        .setName(name)
                        .setType(type)
                        .addAllCategories(categories)
                        .setDtr(ImageIntegrationOuterClass.DTRConfig.newBuilder()
                                .setUsername(username)
                                .setPassword(password)
                                .setEndpoint(endpoint))
                        .build()

        return createImageIntegration(integration)
    }

    static addAzureRegistry() {
        String name = "azure"
        String type = "azure"
        List<ImageIntegrationCategory> categories = [ImageIntegrationCategory.REGISTRY]
        String azurePassword = System.getenv("AZURE_REGISTRY_PASSWORD")
        String azureUsername = "3e30919c-a552-4b1f-a67a-c68f8b32dad8"
        String azureEndpoint = "stackroxacr.azurecr.io"

        ImageIntegrationOuterClass.ImageIntegration integration =
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                        .setName(name)
                        .setType(type)
                        .addAllCategories(categories)
                        .setDocker(ImageIntegrationOuterClass.DockerConfig.newBuilder()
                                .setUsername(azureUsername)
                                .setPassword(azurePassword)
                                .setEndpoint(azureEndpoint))
                        .build()

        return createImageIntegration(integration)
    }

    static String addGcrRegistry(boolean includeScanner = true) {
        String name = "gcr"
        String type = "google"
        List<ImageIntegrationCategory> categories = getIntegrationCategories(includeScanner)
        String serviceAccount = System.getenv("GOOGLE_CREDENTIALS_GCR_SCANNER")
        String endpoint = "us.gcr.io"
        String project = "stackrox-ci"

        ImageIntegrationOuterClass.ImageIntegration integration =
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                        .setName(name)
                        .setType(type)
                        .addAllCategories(categories)
                        .setGoogle(ImageIntegrationOuterClass.GoogleConfig.newBuilder()
                                .setServiceAccount(serviceAccount)
                                .setEndpoint(endpoint)
                                .setProject(project))
                        .build()

        return createImageIntegration(integration)
    }

    static getIntegrationCategories(boolean includeScanner) {
        return includeScanner ?
                [ImageIntegrationCategory.REGISTRY, ImageIntegrationCategory.SCANNER] :
                [ImageIntegrationCategory.REGISTRY]
    }
}
