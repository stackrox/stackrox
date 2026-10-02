import objects.StackroxScannerIntegration
import services.IntegrationHealthService

import org.junit.Assume
import spock.lang.Tag
import spock.lang.Unroll

class IntegrationHealthTest extends BaseSpecification {
    @Unroll
    @Tag("BAT")
    @Tag("PZ")
    def "Verify vulnerability definitions information is available"() {
        when:
        "Vulnerability definition is requested"
        Assume.assumeTrue("Clairify integration is present", StackroxScannerIntegration.isTestable())
        def vulnDefInfo = IntegrationHealthService.getVulnDefinitionsInfo()

        then:
        "Vulnerability definitions update timestamp is not null"
        assert vulnDefInfo.hasLastUpdatedTimestamp()
    }
}
