class AAA extends BaseSpecification {
    @Shared
    private String runCount

    def setupSpec() {
        runCount = 0
    }

    def "Fail once to trigger secret access for defpol"() {
        when:
        runCount++

        then:
        runCount > 1
    }
}
