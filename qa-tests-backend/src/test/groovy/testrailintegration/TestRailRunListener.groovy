package testrailintegration

import org.spockframework.runtime.AbstractRunListener
import org.spockframework.runtime.model.ErrorInfo
import org.spockframework.runtime.model.FeatureInfo
import org.spockframework.runtime.model.SpecInfo

class TestRailRunListener extends AbstractRunListener {
    def errorCases = [:]
    def skippedCases = []
    def passedCases = []

    @Override
    void beforeSpec(SpecInfo spec) {
        TestRailconfig.createTestRailInstance()
        TestRailconfig.setProjectSectionId("Functional Verification Tests-UI", "Container-Detection(1.4)")
        TestRailconfig.createRun()

        println "Starting Spec - (${spec.name})"
    }

    @Override
    void afterSpec(SpecInfo spec) {
        println "Total Failed in spec: ${errorCases}"
        println "Total Skipped in spec: ${skippedCases}"
        println "Total Passed in spec: ${passedCases}"

        //TODO: upload results and close run
    }

    @Override
    void beforeFeature(FeatureInfo feature) {
    }

    @Override
    void afterFeature(FeatureInfo feature) {
        if (!errorCases.keySet().contains(feature.name) &&
                !skippedCases.contains(feature.name)) {
            println "${feature.name} PASSED!!!"
            passedCases.add(feature.name)

            //TODO: Add Passed test result using feature.name
        }
    }

    @Override
    void error(ErrorInfo error) {
        def methodName = error.getMethod().getFeature().name
        def methodErrorDetails = error.exception.stackTrace.toString()

        if (errorCases.get(methodName) == null) {
            errorCases.put(methodName, Arrays.asList(methodErrorDetails))
        }
        else {
            errorCases.put(methodName, errorCases.get(methodName) << methodErrorDetails)
        }
        //TODO: add Failed test result using methodName
    }

    @Override
    void featureSkipped(FeatureInfo feature) {
        println "${feature.name} SKIPPED!!!!!"
        skippedCases.add(feature.name)

        //TODO: Add Skipped test result using feature.name
    }
}
