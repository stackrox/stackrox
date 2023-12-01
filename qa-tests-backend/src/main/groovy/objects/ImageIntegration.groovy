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
                endpoint: "https://scanner.stackrox.svc:8080",
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

class ClairV4ScannerIntegration implements ImageIntegration {

    static String name() { "Clair v4 Scanner" }

    static Boolean isTestable() {
        return Env.get("CLAIR_V4_ENDPOINT") != null
    }

    static ImageIntegrationOuterClass.ImageIntegration.Builder getCustomBuilder(Map customArgs = [:]) {
        Map defaultArgs = [
                name: "clairv4",
                endpoint: Env.get("CLAIR_V4_ENDPOINT", ""),
                insecure: true,
        ]
        Map args = defaultArgs + customArgs

        ImageIntegrationOuterClass.ClairV4Config.Builder config =
                ImageIntegrationOuterClass.ClairV4Config.newBuilder()
                        .setEndpoint(args.endpoint as String)
                        .setInsecure(args.insecure as boolean)

        return ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(args.name as String)
                .setType("clairV4")
                .clearCategories()
                .addAllCategories([ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER])
                .setClairV4(config)
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
                useAssumeRole: false,
                useAssumeRoleExternalId: false,
                assumeRoleAccessKey: Env.mustGetAWSAssumeRoleAccessKeyID(),
                assumeRoleSecretKey: Env.mustGetAWSAssumeRoleSecretKeyID(),
                assumeRoleRoleId: Env.mustGetAWSAssumeRoleRoleID(),
                assumeRoleExternalId: Env.mustGetAWSAssumeRoleExternalID(),
                assumeRoleTestConditionId: Env.mustGetAWSAssumeRoleTestConditionID(),
        ]
        Map args = defaultArgs + customArgs

        if (args.useIam) {
            args.accessKeyId = ""
            args.secretAccessKey = ""
        }

        if (args.useAssumeRole) {
            args.accessKeyId = args.assumeRoleAccessKey
            args.secretAccessKey = args.assumeRoleSecretKey
        } else if (args.useAssumeRoleExternalId) {
            args.useAssumeRole = true
            args.accessKeyId = args.assumeRoleAccessKey
            args.secretAccessKey = args.assumeRoleSecretKey
            args.assumeRoleRoleId = args.assumeRoleExternalId
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
                oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN"),
        ]
        Map args = defaultArgs + customArgs

        ImageIntegrationOuterClass.QuayConfig.Builder config =
                ImageIntegrationOuterClass.QuayConfig.newBuilder()
                        .setEndpoint(args.endpoint as String)
                        .setOauthToken(args.oauthToken as String)
                        .setInsecure(args.insecure as Boolean)

        if (args.useRobotCreds) {
            config.setRegistryRobotCredentials(
                    ImageIntegrationOuterClass.QuayConfig.RobotAccount.newBuilder()
                        .setUsername(Env.mustGet("QUAY_RHACS_ENG_RO_USERNAME"))
                        .setPassword(Env.mustGet("QUAY_RHACS_ENG_RO_PASSWORD"))
            )
        }

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
