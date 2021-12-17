import objects.ECRRegistryIntegration
import services.BaseService
import services.MetadataService

// Some sample code to test the ECR registry integration API
// usage ./gradlew testCreateECRIntegration

BaseService.useBasicAuth()
BaseService.setUseClientCert(false)

println "======================================================="
println MetadataService.getMetadataServiceClient().getMetadata()
println "======================================================="

println "Creating an ECR image integration:"
println ECRRegistryIntegration.createCustomIntegration(useAssumeRole: true)
