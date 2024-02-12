package sampleScripts

import util.TestMetrics

@SuppressWarnings(["SystemErrPrint", "SystemOutPrint", "SystemExit"])
def main() {
    if (args.length == 0) {
        System.err.println("Required args are missing.")
        System.err.println("Usage: gradlew runSampleScript -PrunScript=checkTestSkip " +
                           "--args='<test suite/spec'> [<test case/feature>]")
        System.exit(1)
    }

    def testMetrics = new TestMetrics()

    testMetrics.loadStableSuiteHistory("gke-qa-e2e-tests")
    testMetrics.loadStableTestHistory("gke-qa-e2e-tests")

    String suiteName = args[0]

    if (args.length == 1) {
        System.out.printf("Checking %s\n", suiteName)
        System.out.printf("Will run? %s", !testMetrics.isSuiteStable(suiteName))
    }
    else {
        String caseName = args[1]
        System.out.printf("Checking %s / %s\n", suiteName, caseName)
        System.out.printf("Will run? %s", !testMetrics.isTestStable(suiteName, caseName))
    }
}

main()
