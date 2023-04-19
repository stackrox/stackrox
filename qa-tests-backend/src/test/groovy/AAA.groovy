import spock.lang.Shared
import spock.lang.Tag

class AAA extends BaseSpecification {
    @Shared
    private Integer runCount

    def setupSpec() {
        runCount = 0
        log.info("setupSpec runCount: ${runCount}")
    }

    @Tag("BAT")
    def "Fail once to trigger secret access for defpol"() {
        when:
        runCount++
        log.info("when runCount: ${runCount}")

        then:
        log.info("test runCount: ${runCount > 1}")
        runCount > 1
    }
}
