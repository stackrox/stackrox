package printer

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/tidwall/gjson"
)

const (
	// JUnitFailedTestCasesExpressionKey represents the key for the JSON Path expression which yields all failed test case names
	JUnitFailedTestCasesExpressionKey = "failed-testcases"
	// JUnitFailedTestCaseErrMsgExpressionKey represents the key for the JSON Path expression which yields all failed test case error messages
	JUnitFailedTestCaseErrMsgExpressionKey = "failed-testcases-error-message"
	// JUnitTestCasesExpressionKey represents the key for the JSON Path expression which yields all test case names
	JUnitTestCasesExpressionKey = "testcases"
)

type junitPrinter struct {
	suiteName           string
	jsonPathExpressions map[string]string
}

func newJUnitPrinter(suiteName string, jsonPathExpressions map[string]string) *junitPrinter {
	return &junitPrinter{suiteName: suiteName, jsonPathExpressions: jsonPathExpressions}
}

func (j *junitPrinter) Print(object interface{}, out io.Writer) error {
	jsonObjectBytes, err := json.Marshal(object)
	if err != nil {
		return errorhelpers.NewErrInvariantViolation(err.Error())
	}

	testCaseNames, failedTestCaseNames, failedTestCaseErrorMessages := retrieveJUnitSuiteData(jsonObjectBytes, j.jsonPathExpressions)

	if err := validateJUnitSuiteData(testCaseNames, failedTestCaseNames, failedTestCaseErrorMessages); err != nil {
		return err
	}

	failedTestCases, err := createFailedTestCaseMap(failedTestCaseNames, failedTestCaseErrorMessages)
	if err != nil {
		return err
	}

	suite := createJUnitTestSuite(j.suiteName, testCaseNames, failedTestCases)

	enc := xml.NewEncoder(out)
	enc.Indent("", "  ")
	return enc.Encode(suite)
}

// retrieveJUnitSuiteData retrieves all required data from the JSON object to create a JUnit test suite.
// It returns the test case names, failed test case names and the failed test case error messages.
func retrieveJUnitSuiteData(jsonObjectBytes []byte, junitJSONPathExpressions map[string]string) ([]string, []string, []string) {
	testCaseNamesResult := gjson.GetManyBytes(jsonObjectBytes, junitJSONPathExpressions[JUnitTestCasesExpressionKey])
	testCaseNames := getStringsFromGJSONResult(testCaseNamesResult)

	failedTestCaseNamesResult := gjson.GetManyBytes(jsonObjectBytes, junitJSONPathExpressions[JUnitFailedTestCasesExpressionKey])
	failedTestCaseNames := getStringsFromGJSONResult(failedTestCaseNamesResult)

	failedTestCasesErrorMessagesResult := gjson.GetManyBytes(jsonObjectBytes, junitJSONPathExpressions[JUnitFailedTestCaseErrMsgExpressionKey])
	failedTestCasesErrorMessages := getStringsFromGJSONResult(failedTestCasesErrorMessagesResult)

	return testCaseNames, failedTestCaseNames, failedTestCasesErrorMessages
}

// validateJUnitSuiteData validates the data to create a JUnit test suite for conformity. It checks whether the
// amount of failed test cases and error messages is equal and also ensures that the total amount of test cases is
// not less than the failed test cases.
func validateJUnitSuiteData(testCaseNames []string, failedTestCaseNames []string, failedTestCaseErrorMessages []string) error {
	amountTestCases := len(testCaseNames)
	amountFailedTestCases := len(failedTestCaseNames)
	amountFailedTestCaseErrorMessages := len(failedTestCaseErrorMessages)

	if amountTestCases < amountFailedTestCases {
		return errorhelpers.NewErrInvariantViolation(fmt.Sprintf("%d failed test cases are greater "+
			"than %d overall test cases", amountTestCases, amountFailedTestCases))
	}

	if len(failedTestCaseNames) != len(failedTestCaseErrorMessages) {
		return errorhelpers.NewErrInvariantViolation(fmt.Sprintf("%d failed test cases and %d error "+
			"messages are not matching", amountFailedTestCases, amountFailedTestCaseErrorMessages))
	}
	return nil
}

// createJUnitTestSuite creates a JUnit suite with the given name and test cases. The returned junitTestSuite CAN be
// passed to xml.Marshal
func createJUnitTestSuite(suiteName string, testCaseNames []string, failedTestCases map[string]string) *junitTestSuite {
	suite := &junitTestSuite{
		Name:     suiteName,
		Tests:    len(testCaseNames),
		Failures: len(failedTestCases),
		Errors:   0,
	}
	testCases := make([]junitTestCase, 0, len(testCaseNames))
	for _, testCase := range testCaseNames {
		tc := junitTestCase{
			Name:    testCase,
			Failure: nil,
		}
		if errMsg, exists := failedTestCases[testCase]; exists {
			tc.Failure = &junitFailureMessage{Message: errMsg}
		}
		testCases = append(testCases, tc)
	}
	suite.TestCases = testCases
	return suite
}

// createFailedTestCaseMap provides a helper to match failed test case names with their appropriate error message and returns
// a map, where the key is the failed test case name and the value the associated error message
// It will return an error if a duplicated failed test case name is given
func createFailedTestCaseMap(failedTestCases []string, failedTestCaseErrorMessages []string) (map[string]string, error) {
	failedTestCaseMap := make(map[string]string, len(failedTestCases))
	for i, name := range failedTestCases {
		if _, exists := failedTestCaseMap[name]; exists {
			return nil, errorhelpers.NewErrInvariantViolation(fmt.Sprintf("duplicate failed test "+
				"case %q found", name))
		}
		failedTestCaseMap[name] = failedTestCaseErrorMessages[i]
	}
	return failedTestCaseMap, nil
}

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	TestCases []junitTestCase `xml:"testcase"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
}

type junitTestCase struct {
	XMLName   xml.Name             `xml:"testcase"`
	Name      string               `xml:"name,attr"`
	ClassName string               `xml:"classname,attr"`
	Failure   *junitFailureMessage `xml:"failure,omitempty"`
}

type junitFailureMessage struct {
	Message string `xml:",chardata"`
}
