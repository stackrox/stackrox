import util.Retry

import spock.lang.Specification

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
        assert i++ > 2, "I was $text $i"
        return true
    }
}
