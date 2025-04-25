import groovy.transform.ASTTest
import org.codehaus.groovy.control.CompilePhase

import services.SecretService

import io.stackrox.annotations.Retry
import spock.lang.Specification

@ASTTest(
    phase = CompilePhase.CANONICALIZATION,
    value = {
        assert node.methods*.name.any { it == 'fail_with_retry' }
        assert SecretService.methods*.name.any { it == 'getSecret_with_retry' }
    }
)
class RetryTest extends Specification {
    def test() {
        expect:
        fail("called", 1)
    }

    private int i = 0

    @Retry(attempts = 3, delay = 0)
    def fail(def text, int x) {
        assert text
        assert x == 1
        assert i++ > 2, "I was $text $i times"
        return true
    }
}
