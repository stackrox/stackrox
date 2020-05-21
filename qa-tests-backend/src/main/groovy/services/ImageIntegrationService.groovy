package services

import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.ImageIntegrationServiceGrpc
import io.stackrox.proto.api.v1.ImageIntegrationServiceOuterClass
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import io.stackrox.proto.storage.ImageIntegrationOuterClass.ImageIntegrationCategory
import util.Env

import util.Timer

class ImageIntegrationService extends BaseService {
    static final private String AUTO_REGISTERED_SCANNER_INTEGRATION = "Stackrox Scanner"

    static getImageIntegrationClient() {
        return ImageIntegrationServiceGrpc.newBlockingStub(getChannel())
    }

    static testImageIntegration(ImageIntegrationOuterClass.ImageIntegration integration) {
        try {
            getImageIntegrationClient().testImageIntegration(integration)
            return true
        } catch (Exception e) {
            println e.toString()
            return false
        }
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

    static getImageIntegrations() {
        return getImageIntegrationClient().getImageIntegrations(
                ImageIntegrationServiceOuterClass.GetImageIntegrationsRequest.newBuilder().build()
        ).integrationsList
    }

    static getImageIntegrationByName(String name) {
        List<ImageIntegrationOuterClass.ImageIntegration> integrations = getImageIntegrations()
        def integrationId = integrations.find { it.name == name }?.id
        return integrationId ?
                getImageIntegrationClient().getImageIntegration(getResourceByID(integrationId)) :
                null
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
        ImageIntegrationOuterClass.ImageIntegration integration =
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                        .setName(AUTO_REGISTERED_SCANNER_INTEGRATION)
                        .setType("clairify")
                        .addAllCategories([ImageIntegrationCategory.SCANNER])
                        .setClairify(ImageIntegrationOuterClass.ClairifyConfig.newBuilder()
                                .setEndpoint("https://scanner.stackrox:8080"))
                        .build()

        return createImageIntegration(integration)
    }

    static String addDockerTrustedRegistry(boolean includeScanner = true) {
        ImageIntegrationOuterClass.ImageIntegration integration =
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                        .setName("dtr")
                        .setType("dtr")
                        .addAllCategories(getIntegrationCategories(includeScanner))
                        .setDtr(ImageIntegrationOuterClass.DTRConfig.newBuilder()
                                .setUsername("qa")
                                .setPassword(Env.get("DTR_REGISTRY_PASSWORD", ""))
                                .setEndpoint("https://apollo-dtr.rox.systems/"))
                        .build()

        return createImageIntegration(integration)
    }

    static addAzureRegistry() {
        ImageIntegrationOuterClass.ImageIntegration integration =
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                        .setName("azure")
                        .setType("azure")
                        .addAllCategories([ImageIntegrationCategory.REGISTRY])
                        .setDocker(ImageIntegrationOuterClass.DockerConfig.newBuilder()
                                .setUsername("3e30919c-a552-4b1f-a67a-c68f8b32dad8")
                                .setPassword(Env.mustGet("AZURE_REGISTRY_PASSWORD"))
                                .setEndpoint("stackroxacr.azurecr.io"))
                        .build()

        return createImageIntegration(integration)
    }

    static String addGcrRegistry(boolean includeScanner = true) {
        ImageIntegrationOuterClass.ImageIntegration integration =
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                        .setName("gcr")
                        .setType("google")
                        .addAllCategories(getIntegrationCategories(includeScanner))
                        .setGoogle(ImageIntegrationOuterClass.GoogleConfig.newBuilder()
                                .setServiceAccount(Env.mustGet("GOOGLE_CREDENTIALS_GCR_SCANNER"))
                                .setEndpoint("us.gcr.io")
                                .setProject("stackrox-ci"))
                        .build()

        return createImageIntegration(integration)
    }

    static String addQuayRegistry(boolean includeScanner = true) {
        ImageIntegrationOuterClass.ImageIntegration integration =
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                        .setName("quay")
                        .setType("quay")
                        .addAllCategories(getIntegrationCategories(includeScanner))
                        .setQuay(ImageIntegrationOuterClass.QuayConfig.newBuilder()
                                .setEndpoint("quay.io")
                                .setOauthToken(Env.mustGet("QUAY_BEARER_TOKEN")))
                        .build()

        return createImageIntegration(integration)
    }

    static ImageIntegrationOuterClass.ImageIntegration getECRIntegrationConfig(
            String name,
            String registryID = Env.mustGetAWSECRRegistryID(),
            String registryRegion = Env.mustGetAWSECRRegistryRegion(),
            String endpoint = "",
            String accessKeyId = Env.mustGetAWSAccessKeyID(),
            String accessKey = Env.mustGetAWSSecretAccessKey()) {
        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(name)
                .setType("ecr")
                .addAllCategories([ImageIntegrationCategory.REGISTRY])
                .setEcr(ImageIntegrationOuterClass.ECRConfig.newBuilder()
                        .setRegistryId(registryID)
                        .setRegion(registryRegion)
                        .setEndpoint(endpoint)
                        .setAccessKeyId(accessKeyId)
                        .setSecretAccessKey(accessKey)
                )
                .build()
    }

    static getIntegrationCategories(boolean includeScanner) {
        return includeScanner ?
                [ImageIntegrationCategory.REGISTRY, ImageIntegrationCategory.SCANNER] :
                [ImageIntegrationCategory.REGISTRY]
    }
}
