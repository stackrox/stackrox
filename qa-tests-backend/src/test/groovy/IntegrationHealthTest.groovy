import services.IntegrationHealthService

import spock.lang.Tag
import spock.lang.Unroll

class IntegrationHealthTest extends BaseSpecification {
    def setupSpec() { }

    def cleanupSpec() { }

    @Unroll
    @Tag("BAT")
    @Tag("PZ")
    def "Verify vulnerability definitions information is available"() {
        when:
        "Vulnerability definition is requested"
        def vulnDefInfo = IntegrationHealthService.getVulnDefinitionsInfo()

        then:
        "Vulnerability definitions update timestamp is not null"
        assert vulnDefInfo.hasLastUpdatedTimestamp()
    }
}
