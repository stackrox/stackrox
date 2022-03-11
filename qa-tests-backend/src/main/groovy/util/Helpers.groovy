package util

import common.Constants
import java.nio.file.Path
import java.nio.file.Paths
import java.text.SimpleDateFormat
import org.codehaus.groovy.runtime.powerassert.PowerAssertionError
import org.junit.AssumptionViolatedException
import org.spockframework.runtime.SpockAssertionError

// Helpers defines useful helper methods. Is mixed in to every object in order to be visible everywhere.
class Helpers {
    private static final int MAX_RETRY_ATTEMPTS = 0
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

    static <V> void withRetry(Object ignored, int retries, int pauseSecs, Closure<V> closure) {
        evaluateWithRetry(ignored, retries, pauseSecs, closure)
    }

    static boolean determineRetry(Throwable failure) {
        if (failure instanceof AssumptionViolatedException) {
            println "Skipping retry for: " + failure
            return false
        }

        retryAttempt++
        def willRetry = retryAttempt <= MAX_RETRY_ATTEMPTS
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
            println "Won't collect logs when not in CI"
            return
        }

        if (exception && (exception instanceof AssumptionViolatedException ||
                exception.getMessage().contains("org.junit.AssumptionViolatedException"))) {
            println "Won't collect logs for: ${exception.getMessage()}"
            return
        }

        if (exception) {
            println "An exception occurred in test: ${exception.getMessage()}"
        }

        try {
            def date = new Date()
            def sdf = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US)

            def debugDir = new File(Constants.FAILURE_DEBUG_DIR)
            if (debugDir.exists() && debugDir.listFiles().size() >= Constants.FAILURE_DEBUG_LIMIT) {
                println "${sdf.format(date)} Debug capture limit reached. Not collecting for this failure."
                return
            }

            def collectionDir = debugDir.getAbsolutePath() + "/" + UUID.randomUUID()

            println "${sdf.format(date)} Will collect various stackrox logs for this failure under ${collectionDir}/"

            shellCmd("./scripts/ci/collect-service-logs.sh stackrox ${collectionDir}/stackrox-k8s-logs")
            shellCmd("./scripts/ci/collect-qa-service-logs.sh ${collectionDir}/qa-k8s-logs")
            shellCmd("./scripts/grab-data-from-central.sh ${collectionDir}/central-data")
        }
        catch (Exception e) {
            println "Could not collect logs: ${e}"
        }
    }

    // collectImageScanForDebug(image) - a best effort debug tool to get a complete image scan.
    static void collectImageScanForDebug(String image, String saveName) {
        if (!Env.IN_CI) {
            println "Won't collect image scans when not in CI"
            return
        }

        println "Will scan ${image} to ${saveName}"

        try {
            Path imageScans = Paths.get(Constants.FAILURE_DEBUG_DIR).resolve("image-scans")
            new File(imageScans.toAbsolutePath().toString()).mkdirs()

            Process proc = "./scripts/ci/roxctl.sh image scan -i ${image}".execute(null, new File(".."))
            String output = imageScans.resolve(saveName).toAbsolutePath()
            FileWriter sout = new FileWriter(output)
            StringBuilder serr = new StringBuilder()

            proc.consumeProcessOutput(sout, serr)
            proc.waitFor()

            if (proc.exitValue() != 0) {
                println "Failed to scan the image."
                println "Stderr: $serr"
                println "Exit: ${proc.exitValue()}"
            }
        }
        catch (Exception e) {
            println "Could not collect image details: ${e}"
        }
    }

    static void shellCmd(String cmd) {
        def sout = new StringBuilder(), serr = new StringBuilder()
        def proc = cmd.execute(null, new File(".."))
        proc.consumeProcessOutput(sout, serr)
        proc.waitFor()
        println "Ran: ${cmd}"
        println "Stdout: $sout"
        println "Stderr: $serr"
        println "Exit: ${proc.exitValue()}"
    }
}
