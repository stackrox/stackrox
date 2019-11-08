package util

import org.codehaus.groovy.runtime.powerassert.PowerAssertionError
import org.spockframework.runtime.SpockAssertionError

// Helpers defines useful helper methods. Is mixed in to every object in order to be visible everywhere.
class Helpers {
    static <V> V evaluateWithRetry(Object ignored, int retries, int pauseSecs, Closure<V> closure) {
        for (int i = 0; i < retries; i++) {
            try {
                return closure()
            } catch (Exception | PowerAssertionError | SpockAssertionError t) {
                println "Caught exception: ${t}. Retrying in ${pauseSecs}s"
            }
            sleep pauseSecs * 1000
        }
        return closure()
    }

    static void withRetry(Object ignored, int retries, int pauseSecs, Closure<?> closure) {
        evaluateWithRetry(retries, pauseSecs, closure)
    }
}
