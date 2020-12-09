import groups.BAT
import org.junit.experimental.categories.Category
import services.IntegrationHealthService
import spock.lang.Unroll

class IntegrationHealthTest extends BaseSpecification {
    def setupSpec() { }

    def cleanupSpec() { }

    @Unroll
    @Category([BAT])
    def "Verify vulnerability definitions information is available"() {
        when:
        "Vulnerability definition is requested"
        def vulnDefInfo = IntegrationHealthService.getVulnDefinitionsInfo()

        then:
        "Vulnerability definitions update timestamp is not null"
        assert vulnDefInfo.hasLastUpdatedTimestamp()
    }
}
