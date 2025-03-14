import util.Retry

import spock.lang.Specification

class RetryTest extends Specification {
    def test() {
        expect:
        fail("called", 1)
    }

    int i = 0

    @Retry(attempts = 3, delay = 0)
    def fail(def text, int x) {
        assert text
        assert x == 1
        println "I was $text $i"
        i++
        assert i > 1
        return true
    }
}