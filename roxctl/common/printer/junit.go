package printer

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/roxctl/common/printer/mapper"
)

const (
	// JUnitSkippedTestCasesExpressionKey represents the key for the JSON Path expression which yields all failed test case names
	JUnitSkippedTestCasesExpressionKey = "skipped-testcases"
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
	data, err := retrieveJUnitSuiteData(object, j.jsonPathExpressions)
	if err != nil {
		return err
	}

	testCaseNames := data[JUnitTestCasesExpressionKey]
	skippedTestCaseNames := data[JUnitSkippedTestCasesExpressionKey]
	failedTestCaseNames := data[JUnitFailedTestCasesExpressionKey]
	failedTestCaseErrorMessages := data[JUnitFailedTestCaseErrMsgExpressionKey]

	if err := validateJUnitSuiteData(testCaseNames, failedTestCaseNames, failedTestCaseErrorMessages, skippedTestCaseNames); err != nil {
		return err
	}

	failedTestCases, err := createFailedTestCaseMap(failedTestCaseNames, failedTestCaseErrorMessages)
	if err != nil {
		return err
	}

	suite := createJUnitTestSuite(j.suiteName, testCaseNames, skippedTestCaseNames, failedTestCases)

	enc := xml.NewEncoder(out)
	enc.Indent("", "  ")
	return enc.Encode(suite)
}

// retrieveJUnitSuiteData retrieves all required data from the JSON object to create a JUnit test suite.
// It returns the test case names, failed test case names and the failed test case error messages.
func retrieveJUnitSuiteData(jsonObj interface{}, junitJSONPathExpressions map[string]string) (map[string][]string, error) {
	sliceMapper, err := mapper.NewSliceMapper(jsonObj, junitJSONPathExpressions)
	if err != nil {
		return nil, err
	}

	return sliceMapper.CreateSlices(), nil
}

// validateJUnitSuiteData validates the data to create a JUnit test suite for conformity. It checks whether the
// amount of failed test cases and error messages is equal and also ensures that the total amount of test cases is
// not less than the failed test cases.
func validateJUnitSuiteData(testCaseNames, failedTestCaseNames, failedTestCaseErrorMessages, skippedTestCaseNames []string) error {
	amountTestCases := len(testCaseNames)
	amountFailedTestCases := len(failedTestCaseNames)
	amountFailedTestCaseErrorMessages := len(failedTestCaseErrorMessages)
	amountSkippedTestCases := len(skippedTestCaseNames)

	if amountTestCases < amountFailedTestCases+amountSkippedTestCases {
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
func createJUnitTestSuite(suiteName string, testCaseNames, skippedTestCasesNames []string, failedTestCases map[string]string) *junitTestSuite {
	skippedTests := set.NewStringSet(skippedTestCasesNames...)
	suite := &junitTestSuite{
		Name:     suiteName,
		Tests:    len(testCaseNames),
		Failures: len(failedTestCases),
		Skipped:  skippedTests.Cardinality(),
		Errors:   0,
	}
	testCases := make([]junitTestCase, 0, len(testCaseNames))
	for _, testCase := range testCaseNames {
		tc := junitTestCase{
			Name:    testCase,
			Failure: nil,
			Skipped: nil,
		}
		if skippedTests.Contains(testCase) {
			tc.Skipped = &struct{}{}
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
	Skipped   int             `xml:"skipped,attr"`
	Errors    int             `xml:"errors,attr"`
}

type junitTestCase struct {
	XMLName   xml.Name             `xml:"testcase"`
	Name      string               `xml:"name,attr"`
	ClassName string               `xml:"classname,attr"`
	Failure   *junitFailureMessage `xml:"failure,omitempty"`
	Skipped   *struct{}            `xml:"skipped,omitempty"`
}

type junitFailureMessage struct {
	Message string `xml:",chardata"`
}
