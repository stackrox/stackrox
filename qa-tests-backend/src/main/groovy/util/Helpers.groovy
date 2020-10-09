package util

import common.Constants
import org.codehaus.groovy.runtime.powerassert.PowerAssertionError
import org.junit.AssumptionViolatedException
import org.spockframework.runtime.SpockAssertionError

// Helpers defines useful helper methods. Is mixed in to every object in order to be visible everywhere.
class Helpers {
    private static final int MAX_RETRY_ATTEMTPS = 2
    private static int retryAttempt = 0

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

    static boolean determineRetry(Throwable failure) {
        if (failure instanceof AssumptionViolatedException) {
            println "Skipping retry for: " + failure
            return false
        }

        retryAttempt++
        def willRetry = retryAttempt <= MAX_RETRY_ATTEMTPS
        if (willRetry) {
            println "An exception occurred which will cause a retry: " + failure
            println "Test Failed... Attempting Retry #${retryAttempt}"
        }
        return willRetry
    }

    static void resetRetryAttempts() {
        retryAttempt = 0
    }

    static int getAttemptCount() {
        return retryAttempt + 1
    }

    static void sleepWithRetryBackoff(int milliseconds) {
        sleep milliseconds * getAttemptCount()
    }

    static boolean containsNoWhitespace(Object ignored, String baseString, String subString) {
        return baseString.replaceAll("\\s", "").contains(subString.replaceAll("\\s", ""))
    }

    static String getStackRoxEndpoint(Object ignored) {
        return "https://" + Env.mustGetHostname() + ":" + Env.mustGetPort()
    }

    // withDo is like with, but returns a void so can safely be used in tests.
    static void withDo(Object self, Closure closure) {
        self.with(closure)
    }

    static void collectDebugForFailure(Throwable exception) {
        if (!Env.IN_CI) {
            return
        }

        if (exception && exception instanceof AssumptionViolatedException) {
            println "Won't collect logs for: ${exception.getMessage()}"
            return
        }

        if (exception) {
            println "An exception occurred in test: ${exception.getMessage()}"
        }

        try {
            def debugDir = new File(Constants.FAILURE_DEBUG_DIR)
            if (debugDir.exists() && debugDir.listFiles().size() >= Constants.FAILURE_DEBUG_LIMIT) {
                println "Debug capture limit reached. Not collecting for this failure."
                return
            }

            def collectionDir = debugDir.getAbsolutePath() + "/" + UUID.randomUUID()

            println "Will collect various stackrox logs for this failure under ${collectionDir}/"

            shellCmd("./scripts/ci/collect-service-logs.sh stackrox ${collectionDir}/stackrox-k8s-logs")
            shellCmd("./scripts/ci/collect-qa-service-logs.sh ${collectionDir}/qa-k8s-logs")
            shellCmd("./scripts/grab-data-from-central.sh ${collectionDir}/central-data")
        }
        catch (Exception e) {
            println "Could not collect logs: ${e}"
        }
    }

    static void shellCmd(String cmd, Integer timeout = 60000) {
        def sout = new StringBuilder(), serr = new StringBuilder()
        def proc = cmd.execute(null, new File(".."))
        proc.consumeProcessOutput(sout, serr)
        proc.waitForOrKill(timeout)
        println "Ran: ${cmd}"
        println "Stdout: $sout"
        println "Stderr: $serr"
    }
}
