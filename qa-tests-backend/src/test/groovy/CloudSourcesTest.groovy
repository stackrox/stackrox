import static util.Helpers.withRetry

import io.stackrox.proto.api.v1.CloudSourceService
import services.CloudSourcesService
import services.DiscoveredClustersService
import util.Env

import spock.lang.Tag

class CloudSourcesTest extends BaseSpecification {

    static final private String CLOUD_SOURCE_NAME = "testing OCM"

    @Tag("BAT")
    def "Create OCM cloud source and verify discovered clusters exist: #authMethod"() {
        when:
        "OCM cloud source is created and tested"
        // On create we call "api.openshift.com" to ensure that configuration works. Internal timeout is ~8sec.
        // In case API service is not temporally available, we should add longer retry times to give API service
        // enough time to recover. Our options are limited here, because we depend on 3rd party service availability.
        def cloudSourceId
        withRetry(10, 30) {
            cloudSourceId = CloudSourcesService.createCloudSource(CloudSourceService.CloudSource.newBuilder().
                setName(CLOUD_SOURCE_NAME).
                setType(CloudSourceService.CloudSource.Type.TYPE_OCM).
                setOcm(CloudSourceService.OCMConfig.newBuilder().
                    setEndpoint("https://api.openshift.com").build()).
                setCredentials(CloudSourceService.CloudSource.Credentials.newBuilder().
                    setSecret(token).
                    setClientId(clientId).
                    setClientSecret(clientSecret).build())
                .build())
            assert cloudSourceId
        }

        then:
        "verify we have discovered clusters"
        // The initial sync may take a bit since we are connecting to the "Red Hat1" OCM organization which hosts
        // a lot of clusters.
        withRetry(10, 30) {
            def count = DiscoveredClustersService.countDiscoveredClusters()
            assert count > 0
        }

        cleanup:
        if (cloudSourceId) {
            CloudSourcesService.deleteCloudSource(cloudSourceId)
        }

        where:
        authMethod            | token                        | clientId                 | clientSecret
        "OCM offline token"   | Env.mustGetOcmOfflineToken() | ""                       | ""
        "OCM service account" | ""                           | Env.mustGetOcmClientId() | Env.mustGetOcmClientSecret()
    }
}
