import spock.lang.Specification
import spock.lang.Tag

import common.Constants
import util.Helpers

@Tag("BAT")
class HelpersTest extends Specification {

    static final private String LONG_STRING = new String(new char[50]).replace("\0", "0123456789")

    def "Verify Helpers.compareAnnotations()"() {
        expect:
        Helpers.compareAnnotations(orchestratorAnnotations, stackroxAnnotations) == same

        where:
        orchestratorAnnotations | stackroxAnnotations | same
        [ "same": "value" ] | [ "same": "value" ] | true
        [ "same": "value" ] | [ "different key": "value" ] | false
        [ "some": "value" ] | [ "same": "different value" ] | false
        [ "same": "value" ] | [ "same": "value", "extra": "extra" ] | false
        [ "same": "value", "extra": "extra" ] | [ "same": "value" ] | false
        [ "same": LONG_STRING ] | [ "same": LONG_STRING ] | true
        // long annotations are still the same when truncated by stackrox
        [ "same": LONG_STRING ] | \
            [ "same": LONG_STRING.substring(0, Constants.STACKROX_ANNOTATION_TRUNCATION_LENGTH) + "..." ] | true
        // however there is a limit
        [ "same": LONG_STRING.substring(0, Constants.STACKROX_ANNOTATION_TRUNCATION_LENGTH) ] | \
            [ "same": LONG_STRING.substring(0, 257) + "..." ] | false
    }
}
