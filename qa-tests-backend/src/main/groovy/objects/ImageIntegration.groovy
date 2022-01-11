package objects

import common.Constants
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import services.ImageIntegrationService
import util.Env

trait ImageIntegration {
    abstract static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs)

    static ImageIntegrationOuterClass.ImageIntegration.Builder getDefaultBuilder() {
        getCustomBuilder()
    }

    static String createDefaultIntegration() {
        ImageIntegrationService.createImageIntegration(
                getDefaultBuilder().build()
        )
    }

    static String createCustomIntegration(Map customArgs = [:]) {
        ImageIntegrationService.createImageIntegration(
                getCustomBuilder(customArgs).build(),
                customArgs.containsKey("skipTestIntegration") && customArgs.skipTestIntegration
        )
    }
}

class StackroxScannerIntegration implements ImageIntegration {

    static String name() { Constants.AUTO_REGISTERED_STACKROX_SCANNER_INTEGRATION }

    static Boolean isTestable() {
        return true
    }

    static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs = [:]) {
        Map defaultArgs = [
                name: Constants.AUTO_REGISTERED_STACKROX_SCANNER_INTEGRATION,
                endpoint: "https://scanner.stackrox:8080",
        ]
        Map args = defaultArgs + customArgs

        ImageIntegrationOuterClass.ClairifyConfig.Builder config =
                ImageIntegrationOuterClass.ClairifyConfig.newBuilder()
                        .setEndpoint(args.endpoint as String)

        def categories = [
            ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER,
            ImageIntegrationOuterClass.ImageIntegrationCategory.NODE_SCANNER,
        ]

        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(args.name as String)
                .setType("clairify")
                .clearCategories()
                .addAllCategories(categories)
                .setClairify(config)
    }
}

class AnchoreScannerIntegration implements ImageIntegration {

    static String name() { "Anchore Scanner" }

    static Boolean isTestable() {
        return Env.get("ANCHORE_ENDPOINT") != null
    }

    static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs = [:]) {
        Map defaultArgs = [
                name: "anchore",
                endpoint: Env.get("ANCHORE_ENDPOINT", ""),
                username: Env.get("ANCHORE_USERNAME", ""),
                password: Env.get("ANCHORE_PASSWORD", ""),
        ]
        Map args = defaultArgs + customArgs

        ImageIntegrationOuterClass.AnchoreConfig.Builder config =
                ImageIntegrationOuterClass.AnchoreConfig.newBuilder()
                        .setUsername(args.username as String)
                        .setPassword(args.password as String)
                        .setEndpoint(args.endpoint as String)

        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(args.name as String)
                .setType("anchore")
                .clearCategories()
                .addAllCategories([ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER])
                .setAnchore(config)
    }
}

class ClairScannerIntegration implements ImageIntegration {

    static String name() { "Clair Scanner" }

    static Boolean isTestable() {
        return Env.get("CLAIR_ENDPOINT") != null
    }

    static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs = [:]) {
        Map defaultArgs = [
                name: "clair",
                endpoint: Env.get("CLAIR_ENDPOINT", ""),
        ]
        Map args = defaultArgs + customArgs

        ImageIntegrationOuterClass.ClairConfig.Builder config =
                ImageIntegrationOuterClass.ClairConfig.newBuilder()
                        .setEndpoint(args.endpoint as String)

        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(args.name as String)
                .setType("clair")
                .clearCategories()
                .addAllCategories([ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER])
                .setClair(config)
    }
}

class ECRRegistryIntegration implements ImageIntegration {

    static String name() { "ECR Registry" }

    static Boolean isTestable() {
        return true
    }

    static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs = [:]) {
        Map defaultArgs = [
                name: "ecr",
                endpoint: "ecr.${Env.mustGetAWSECRRegistryRegion()}.amazonaws.com",
                registryId: Env.mustGetAWSECRRegistryID(),
                region: Env.mustGetAWSECRRegistryRegion(),
                accessKeyId: Env.mustGetAWSAccessKeyID(),
                secretAccessKey: Env.mustGetAWSSecretAccessKey(),
                useIam: false,
                assumeRoleRoleId: Env.mustGetAWSAssumeRoleRoleID(),
                assumeRoleTestConditionId: Env.mustGetAWSAssumeRoleTestConditionID(),
                useAssumeRole: false,
                useAssumeRoleExternalId: false,
        ]
        Map args = defaultArgs + customArgs

        if (args.useIam) {
            args.accessKeyId = ""
            args.secretAccessKey = ""
        }

        if (args.useAssumeRole) {
            args.endpoint = ""
            args.accessKeyId = Env.mustGetAWSAssumeRoleAccessKeyID()
            args.secretAccessKey = Env.mustGetAWSAssumeRoleSecretKeyID()
        }

        if (args.useAssumeRoleExternalId) {
            args.useAssumeRole = true
            args.endpoint = ""
            args.accessKeyId = Env.mustGetAWSAssumeRoleAccessKeyID()
            args.secretAccessKey = Env.mustGetAWSAssumeRoleSecretKeyID()
            args.assumeRoleRoleId = Env.mustGetAWSAssumeRoleExternalID()
        }

        ImageIntegrationOuterClass.ECRConfig.Builder config =
                ImageIntegrationOuterClass.ECRConfig.newBuilder()
                        .setEndpoint(args.endpoint as String)
                        .setRegistryId(args.registryId as String)
                        .setRegion(args.region as String)
                        .setAccessKeyId(args.accessKeyId as String)
                        .setSecretAccessKey(args.secretAccessKey as String)
                        .setUseIam(args.useIam as Boolean)
                        .setUseAssumeRole(args.useAssumeRole as Boolean)
                        .setAssumeRoleId(args.assumeRoleRoleId as String)
                        .setAssumeRoleExternalId(args.assumeRoleTestConditionId as String)

        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(args.name as String)
                .setType("ecr")
                .clearCategories()
                .addAllCategories([ImageIntegrationOuterClass.ImageIntegrationCategory.REGISTRY])
                .setEcr(config)
    }
}

class AzureRegistryIntegration implements ImageIntegration {

    static String name() { "Azure Registry" }

    static Boolean isTestable() {
        return true
    }

    static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs = [:]) {
        Map defaultArgs = [
                name: "acr",
                endpoint: "stackroxci.azurecr.io",
                username: "stackroxci",
                password: Env.mustGet("AZURE_REGISTRY_PASSWORD"),
        ]
        Map args = defaultArgs + customArgs

        ImageIntegrationOuterClass.DockerConfig.Builder config =
                ImageIntegrationOuterClass.DockerConfig.newBuilder()
                        .setEndpoint(args.endpoint as String)
                        .setUsername(args.username as String)
                        .setPassword(args.password as String)

        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(args.name as String)
                .setType("azure")
                .clearCategories()
                .addAllCategories([ImageIntegrationOuterClass.ImageIntegrationCategory.REGISTRY])
                .setDocker(config)
    }
}

class QuayImageIntegration implements ImageIntegration {

    static String name() { "Quay Registry+Scanner" }

    static Boolean isTestable() {
        return true
    }

    static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs = [:]) {
        Map defaultArgs = [
                name: "quay",
                endpoint: "quay.io",
                includeScanner: true,
                insecure: false,
                oauthToken: Env.mustGet("QUAY_BEARER_TOKEN"),
        ]
        Map args = defaultArgs + customArgs

        ImageIntegrationOuterClass.QuayConfig.Builder config =
                ImageIntegrationOuterClass.QuayConfig.newBuilder()
                        .setEndpoint(args.endpoint as String)
                        .setOauthToken(args.oauthToken as String)
                        .setInsecure(args.insecure as Boolean)

        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(args.name as String)
                .setType("quay")
                .clearCategories()
                .addAllCategories(ImageIntegrationService.getIntegrationCategories(args.includeScanner as Boolean))
                .setQuay(config)
    }
}

class GoogleArtifactRegistry implements ImageIntegration {

    static String name() { "Google Artifact Registry" }

    static Boolean isTestable() {
        return true
    }

    static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs = [:]) {
        Map defaultArgs = [
                name: "google-artifact-registry",
                project: "stackrox-ci",
                endpoint: "us-west1-docker.pkg.dev",
                serviceAccount: Env.mustGet("GOOGLE_ARTIFACT_REGISTRY_SERVICE_ACCOUNT"),
                skipTestIntegration: false,
        ]
        Map args = defaultArgs + customArgs

        ImageIntegrationOuterClass.GoogleConfig.Builder config =
                ImageIntegrationOuterClass.GoogleConfig.newBuilder()
                        .setProject(args.project as String)
                        .setServiceAccount(args.serviceAccount as String)
                        .setEndpoint(args.endpoint as String)

        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(args.name as String)
                .setType("artifactregistry")
                .clearCategories()
                .addAllCategories(ImageIntegrationService.getIntegrationCategories(false))
                .setGoogle(config)
                .setSkipTestIntegration(args.skipTestIntegration as Boolean)
    }
}

class GCRImageIntegration implements ImageIntegration {

    static String name() { "GCR Registry+Scanner" }

    static Boolean isTestable() {
        return true
    }

    static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs = [:]) {
        Map defaultArgs = [
                name: "gcr",
                project: "stackrox-ci",
                endpoint: "us.gcr.io",
                includeScanner: true,
                serviceAccount: Env.mustGet("GOOGLE_CREDENTIALS_GCR_SCANNER"),
                skipTestIntegration: false,
        ]
        Map args = defaultArgs + customArgs

        ImageIntegrationOuterClass.GoogleConfig.Builder config =
                ImageIntegrationOuterClass.GoogleConfig.newBuilder()
                        .setProject(args.project as String)
                        .setServiceAccount(args.serviceAccount as String)
                        .setEndpoint(args.endpoint as String)

        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(args.name as String)
                .setType("google")
                .clearCategories()
                .addAllCategories(ImageIntegrationService.getIntegrationCategories(args.includeScanner as Boolean))
                .setGoogle(config)
                .setSkipTestIntegration(args.skipTestIntegration as Boolean)
    }
}
