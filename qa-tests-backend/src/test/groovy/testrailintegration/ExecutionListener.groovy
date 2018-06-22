package testrailintegration

import org.junit.runner.Description
import org.junit.runner.Result
import org.junit.runner.notification.Failure
import org.junit.runner.notification.RunListener

/**
 * Created by parulshukla on 3/13/18.
 */
class ExecutionListener extends RunListener {
    /**
     * Called before any tests have been run.
     * */

    void testRunStarted(Description description) throws java.lang.Exception {
        System.out.println("Number of testcases to execute : " + description.testCount())
    }

    /**
     *  Called when all tests have finished
     * */

    void testRunFinished(Result result) throws java.lang.Exception {
        System.out.println("Number of testcases executed : " + result.getRunCount())
    }

    /**
     *  Called when an atomic test is about to be started.
     * */

    void testStarted(Description description) throws java.lang.Exception {
        System.out.println("Starting execution of test case : " + description.getMethodName())
    }

    /**
     *  Called when an atomic test has finished, whether the test succeeds or fails.
     * */

    void testFinished(Description description) throws java.lang.Exception {
        System.out.println("Finished execution of test case : " + description.getMethodName())
    }

    /**
     *  Called when an atomic test fails.
     * */

    void testFailure(Failure failure) throws java.lang.Exception {
        System.out.println("Execution of test case failed : " + failure.getMessage())
    }

    /**
     *  Called when a test will not be run, generally because a test method is annotated with Ignore.
     * */

    void testIgnored(Description description) throws java.lang.Exception {
        System.out.println("Execution of test case ignored : " + description.getMethodName())
    }
}
