package printer

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/printers"
)

// JUnitPrinterFactory holds all configuration options for a JUnit printer.
// It is an implementation of CustomPrinterFactory and acts as a factory for a JUnitPrinter
type JUnitPrinterFactory struct {
	suiteName string
	// jsonPathExpressions hold all required expressions to build a JUnit test suite.
	// The data is currently NOT expected to be given by the user. The map itself MUST contain the keys
	// JUnitFailedTestCasesExpressionKey, JUnitFailedTestCaseErrMsgExpressionKey, JUnitTestCasesExpressionKey
	jsonPathExpressions map[string]string
}

// NewJUnitPrinterFactory creates a new JUnitPrinterFactory with the injected default values
func NewJUnitPrinterFactory(defaultTestSuiteName string, jsonPathExpressions map[string]string) *JUnitPrinterFactory {
	return &JUnitPrinterFactory{suiteName: defaultTestSuiteName, jsonPathExpressions: jsonPathExpressions}
}

// AddFlags will add all JUnit printer specific flags to the cobra.Command
func (j *JUnitPrinterFactory) AddFlags(cmd *cobra.Command) {
	// TODO: Check whether this is actually to be set by the user or whether this should be set by the command, i.e. image name
	cmd.PersistentFlags().StringVar(&j.suiteName, "junit-suite-name", j.suiteName, "set the name of the JUnit test suite")
}

// SupportedFormats returns the supported printer format that can be created by JUnitPrinterFactory
func (j *JUnitPrinterFactory) SupportedFormats() []string {
	return []string{"junit"}
}

// CreatePrinter creates a JUnitPrinter from the options set. If the format is unsupported, or it is not
// possible to create an ObjectPrinter with the current configuration it will return an error
// A JUnit printer expects a JSON Object and a map of JSON Path expressions that are compatible
// with GJSON (https://github.com/tidwall/gjson).
// When printing, the JUnit printer will take the given JSON object and apply the JSON Path expressions
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
// the mapper.ListModifier and mapper.BoolReplaceModifier are available, see their documentation on usage and
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
//	expressions := map[string]{
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
func (j *JUnitPrinterFactory) CreatePrinter(format string) (ObjectPrinter, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "junit":
		return printers.NewJUnitPrinter(j.suiteName, j.jsonPathExpressions), nil
	default:
		return nil, errox.InvalidArgs.Newf("invalid output format used for JUnit Printer %q", format)
	}
}

// validate verifies whether the current configuration can be used to create an ObjectPrinter. It will return an error
// if it is not possible
func (j *JUnitPrinterFactory) validate() error {
	// ensure that the suite name is not empty
	if j.suiteName == "" {
		return errox.InvalidArgs.New("empty JUnit test suite name given, " +
			"please provide a meaningful name")
	}

	for _, key := range []string{printers.JUnitTestCasesExpressionKey, printers.JUnitFailedTestCasesExpressionKey, printers.JUnitFailedTestCaseErrMsgExpressionKey} {
		if _, exists := j.jsonPathExpressions[key]; !exists {
			// since the jsonPathExpression map is NOT expected to be set by the user, return an ErrInvariantViolation
			// instead
			return errox.InvariantViolation.Newf("missing required JSON Path expression for key %q", key)
		}
	}

	return nil
}
