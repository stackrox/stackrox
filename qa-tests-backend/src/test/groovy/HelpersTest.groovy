import spock.lang.Specification
import spock.lang.Tag

import common.Constants
import util.Helpers

@Tag("BAT")
class HelpersTest extends Specification {

    static final private String LONG_STRING = new String(new char[50]).replace("\0", "0123456789")

    def "Verify Helpers.compareAnnotations() same"() {
        when:
        Helpers.compareAnnotations(orchestratorAnnotations, stackroxAnnotations)

        then:
        notThrown(org.codehaus.groovy.runtime.powerassert.PowerAssertionError)

        where:
        orchestratorAnnotations | stackroxAnnotations
        [ "same": "value" ] | [ "same": "value" ]
        [ "same": LONG_STRING ] | [ "same": LONG_STRING ]
        // long annotations are still the same when truncated by stackrox
        [ "same": LONG_STRING ] | \
            [ "same": LONG_STRING.substring(0, Constants.STACKROX_ANNOTATION_TRUNCATION_LENGTH) + "..." ]
    }

    def "Verify Helpers.compareAnnotations() differences"() {
        when:
        Helpers.compareAnnotations(orchestratorAnnotations, stackroxAnnotations)

        then:
        thrown(org.codehaus.groovy.runtime.powerassert.PowerAssertionError)

        where:
        orchestratorAnnotations | stackroxAnnotations
        [ "same": "value" ] | [ "different key": "value" ]
        [ "some": "value" ] | [ "same": "different value" ]
        [ "same": "value" ] | [ "same": "value", "extra": "extra" ]
        [ "same": "value", "extra": "extra" ] | [ "same": "value" ]
        [ "same": LONG_STRING ] | [ "missing": "value" ]
        // There is an upper limit to long annotation values that are ignored due to truncation
        [ "same": LONG_STRING.substring(0, Constants.STACKROX_ANNOTATION_TRUNCATION_LENGTH) ] | \
            [ "same": LONG_STRING.substring(0, 257) + "..." ]
    }
}
