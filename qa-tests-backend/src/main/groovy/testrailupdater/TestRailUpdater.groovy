package testrailupdater

import groovy.io.FileType
import org.jsoup.Jsoup
import org.jsoup.nodes.Document
import org.jsoup.nodes.Element
import org.jsoup.select.Elements
import util.Env

class TestRailUpdater {

    public static Map<String, List<TestResult>> results = [:]

    static class TestResult {
        def suite
        def testName
        def passed
        def reason
        def elapsed
        def id
    }

    static enum TestResultStatus {
        PASSED("passed"),
        FAILED("failed"),
        IGNORED("ignored")

        private final String value

        TestResultStatus(String value) {
            this.value = value
        }

        @Override
        String toString() {
            return this.value
        }
    }

    static void main(String[] args) {
        compileTestResults()
        if (TestRailManager.setupTestRailInstance()) {
            TestRailManager.updateTestResults(results)
        } else {
            println "Could not complete TestRail initialization before timeout... Aborting update."
        }
    }

    static void compileTestResults() {
        def resultsFiles = []
        def resultsDir = new File(Env.mustGetResultsFilePath())
        resultsDir.eachFileRecurse (FileType.FILES) { file ->
            resultsFiles << file
        }

        for (File f : resultsFiles) {
            println "Parsing results file: ${f.toString()}"
            def specResults = parseHTML(f.toString())
            def specName = f.toString()[f.toString().lastIndexOf("/")+1..f.toString().lastIndexOf(".")-1]
            if (results.containsKey(specName)) {
                specResults.addAll(results.get(specName))
            }
            results.put(specName, specResults.find { it.testName == "classMethod" } ? [] : specResults)
        }
    }

    static List<TestResult> parseHTML(String filePath) {
        List<TestResult> tests = []

        File html = new File(filePath)
        Document document = Jsoup.parse(html, "UTF-8")
        Elements failedTests = document.getElementsByClass("tab").get(0).getElementsByClass("test")
        Element allTestsTable = document.select("table").get(2)
        Elements allTests = allTestsTable.select("tr")

        for (Element e : allTests) {
            if (e.select("td").size() > 0) {
                def name = e.select("td").get(0).text()
                def elapsed = e.select("td").get(1).text()
                        .replaceAll("[^\\d\\.](?=.)")  { it[0] + " " }
                        .replaceAll("-", "")
                def res = e.select("td").get(2).text()
                def reason = ""

                if (res == "failed") {
                    Element b = failedTests.find {
                        it.select("h3").get(0).text() == name
                    }
                    reason = b?.select("span")?.get(0)?.text()
                }

                tests.add(new TestResult(
                        suite: filePath.toString()[
                                filePath.toString().lastIndexOf("/")+1..
                                        filePath.toString().lastIndexOf(".")-1],
                        testName: name,
                        passed: TestResultStatus.valueOf(res.toUpperCase()),
                        reason: reason,
                        elapsed: elapsed)
                )
            }
        }
        return tests
    }
}
