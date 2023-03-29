package printers

import (
	"encoding/xml"
	"io"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/gjson"
	"github.com/stackrox/rox/pkg/set"
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

// JUnitPrinter will print a JUnit compatible output from a given JSON Object.
type JUnitPrinter struct {
	suiteName           string
	jsonPathExpressions map[string]string
}

// NewJUnitPrinter creates a JUnitPrinter from the options set.
// A JUnit printer expects a JSON Object and a map of JSON Path expressions that are compatible
// with GJSON (https://github.com/tidwall/gjson).
// When printing, the JUnitPrinter will take the given JSON object and apply the JSON Path expressions
// within the map to retrieve all required data to generate a JUnit report.
// The JSON Object itself MUST be passable to json.Marshal, so it CAN NOT be a direct JSON input.
// For the structure of the JSON object, it is preferred to have arrays of structs instead of
// array of elements, since structs will provide default values if a field is missing.
// The map of JSON Path expressions MUST provide a JSON Path expression for the keys
// JUnitFailedTestCasesExpressionKey, JUnitFailedTestCaseErrMsgExpressionKey and JUnitTestCasesExpressionKey.
// The JUnitFailedTestCasesExpressionKey is expected to yield an array of strings that represents
// the names of test cases that should be marked as failed within the JUnit report.
// The JUnitFailedTestCaseErrMsgExpressionKey is expected to yield an array of strings that represents
// the error messages of failed test cases. This is in relation with the array yielded from the expression
// JUnitFailedTestCasesExpressionKey.
// The JUnitTestCasesExpressionKey is expected to yield an array of strings that represents all the names
// of all test cases that should be added within the JUnit report.
//
// The GJSON expression syntax (https://github.com/tidwall/gjson/blob/master/SYNTAX.md) offers more complex
// and advanced scenarios, if you require them and the below example is not sufficient.
// Additionally, there are custom GJSON modifiers, which will post-process expression results. Currently,
// the gjson.ListModifier and gjson.BoolReplaceModifier are available, see their documentation on usage and
// GJSON's syntax expression to read more about modifiers.
// The following example illustrates a JSON compatible structure and an example for the map of JSON Path expressions
// JSON structure:
//
//	type data struct {
//			Policies []policy `json:"policies"`
//			FailedPolicies []failedPolicy `json:"failedPolicies"`
//	}
//
//	type policy struct {
//			name string `json:"name"`
//			severity string `json:"severity"`
//	}
//
//	type failedPolicy struct {
//			name string `json:"name"`
//			error string `json:"error"`
//	}
//
// Data:
//
//	data := &data{Policies: []policy{
//									{name: "policy1", severity: "HIGH"},
//									{name: "policy2", severity: "LOW"},
//									{name: "policy3", severity: "MEDIUM"}
//									},
//					 FailedPolicies: []failedPolicy{
//									{name: "policy1", error: "error msg1"}},
//					}
//
// Map of GJSON expressions:
//
//   - specify "#" to visit each element of an array
//
//   - the expressions for failed test cases and error messages MUST be equal and correlated
//
// Example:
//
//	expressions := map[string] {
//	JUnitFailedTestCasesExpressionKey: "data.failedPolicies.#.name",
//	JUnitFailedTestCaseErrMsgExpressionKey: "data.failedPolicies.#.error",
//	JUnitTestCasesExpressionKey: "data.policies.#.name",
//	}
//
// This would result in the following test cases and failed test cases:
// Amount of test cases: 3
// Testcases:
//   - Name: policy1, Failed: "error msg1"
//   - Name: policy2, Successful
//   - Name: policy3, Successful
func NewJUnitPrinter(suiteName string, jsonPathExpressions map[string]string) *JUnitPrinter {
	return &JUnitPrinter{suiteName: suiteName, jsonPathExpressions: jsonPathExpressions}
}

// Print will print a JUnit compatible output to the io.Writer.
// It will return an error if there is an issue with the JSON object, the JUnit report could not be generated
// or it was not possible to write to the io.Writer.
func (j *JUnitPrinter) Print(object interface{}, out io.Writer) error {
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
	sliceMapper, err := gjson.NewSliceMapper(jsonObj, junitJSONPathExpressions)
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
		return errox.InvariantViolation.CausedByf("%d failed test cases are greater "+
			"than %d overall test cases", amountTestCases, amountFailedTestCases)
	}

	if len(failedTestCaseNames) != len(failedTestCaseErrorMessages) {
		return errox.InvariantViolation.CausedByf("%d failed test cases and %d error "+
			"messages are not matching", amountFailedTestCases, amountFailedTestCaseErrorMessages)
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
			return nil, errox.InvariantViolation.CausedByf("duplicate failed test "+
				"case %q found", name)
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
