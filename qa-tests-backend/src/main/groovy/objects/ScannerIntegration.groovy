package objects

import common.Constants
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import util.Env

interface ScannerIntegration {
    String name()
    Boolean isTestable()
    Object getScannerConfig()
    ImageIntegrationOuterClass.ImageIntegration.Builder getBuilder(Object config)
}

class StackroxScannerIntegration implements ScannerIntegration {

    String name() { Constants.AUTO_REGISTERED_STACKROX_SCANNER_INTEGRATION }

    Boolean isTestable() {
        return true
    }

    Object getScannerConfig() {
        ImageIntegrationOuterClass.ClairifyConfig.newBuilder()
                .setEndpoint("https://scanner.stackrox:8080")
    }

    ImageIntegrationOuterClass.ImageIntegration.Builder getBuilder(Object config) {
        ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName(Constants.AUTO_REGISTERED_STACKROX_SCANNER_INTEGRATION)
                .setType("clairify")
                .addAllCategories([ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER])
                .setClairify(config as ImageIntegrationOuterClass.ClairifyConfig.Builder)
    }
}

class AnchoreScannerIntegration implements ScannerIntegration {

    String name() { "Anchore Scanner" }

    Boolean isTestable() {
        return Env.get("ANCHORE_ENDPOINT") != null
    }

    Object getScannerConfig() {
        ImageIntegrationOuterClass.AnchoreConfig.newBuilder()
                .setPassword(Env.get("ANCHORE_PASSWORD", ""))
                .setUsername(Env.get("ANCHORE_USERNAME", ""))
                .setEndpoint(Env.get("ANCHORE_ENDPOINT", ""))
    }

    ImageIntegrationOuterClass.ImageIntegration.Builder getBuilder(Object config) {
        ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName("anchore")
                .setType("anchore")
                .addAllCategories([ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER])
                .setAnchore(config as ImageIntegrationOuterClass.AnchoreConfig.Builder)
    }
}

class ClairScannerIntegration implements ScannerIntegration {

    String name() { "Clair Scanner" }

    Boolean isTestable() {
        return Env.get("CLAIR_ENDPOINT") != null
    }

    Object getScannerConfig() {
        ImageIntegrationOuterClass.ClairConfig.newBuilder()
                .setEndpoint(Env.get("CLAIR_ENDPOINT", ""))
    }

    ImageIntegrationOuterClass.ImageIntegration.Builder getBuilder(Object config) {
        ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                .setName("clair")
                .setType("clair")
                .addAllCategories([ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER])
                .setClair(config as ImageIntegrationOuterClass.ClairConfig.Builder)
    }
}
