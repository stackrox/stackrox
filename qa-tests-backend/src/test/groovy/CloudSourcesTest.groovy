import static util.Helpers.withRetry

import io.stackrox.proto.api.v1.CloudSourceService
import services.CloudSourcesService
import services.DiscoveredClustersService
import util.Env

import spock.lang.Tag

class CloudSourcesTest extends BaseSpecification {

    static final private String CLOUD_SOURCE_NAME = "testing OCM"

    @Tag("BAT")
    def "Create OCM cloud source and verify discovered clusters exist"() {
        when:
        "OCM cloud source is created and tested"
        def cloudSourceId = CloudSourcesService.createCloudSource(CloudSourceService.CloudSource.newBuilder().
                setName(CLOUD_SOURCE_NAME).
                setOcm(CloudSourceService.OCMConfig.newBuilder().
                        setEndpoint("https://api.openshift.com").build()).
                setCredentials(CloudSourceService.CloudSource.Credentials.newBuilder().
                        setSecret(Env.mustGetOcmOfflineToken()).build())
                .build())
        assert cloudSourceId

        then:
        "verify we have discovered clusters"
        // The initial sync may take a bit since we are connecting to the "Red Hat1" OCM organization which hosts
        // a lot of clusters.
        withRetry(10, 10) {
            def count = DiscoveredClustersService.countDiscoveredClusters()
            assert count > 0
        }

        cleanup:
        if (cloudSourceId) {
            CloudSourcesService.deleteCloudSource(cloudSourceId)
        }
    }
}
