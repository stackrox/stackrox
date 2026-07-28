package printers

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type junitSuite struct {
	XMLName  xml.Name      `xml:"testsuite"`
	Name     string        `xml:"name,attr"`
	Tests    int           `xml:"tests,attr"`
	Failures int           `xml:"failures,attr"`
	Skipped  int           `xml:"skipped,attr"`
	Errors   int           `xml:"errors,attr"`
	Cases    []xmlTestCase `xml:"testcase"`
}

type xmlTestCase struct {
	Name    string        `xml:"name,attr"`
	Failure *junitFailure `xml:"failure"`
	Skipped *struct{}     `xml:"skipped"`
}

type junitFailure struct {
	Message string `xml:",chardata"`
}

func parseJUnitXML(data []byte) (junitSuite, error) {
	var s junitSuite
	return s, xml.Unmarshal(data, &s)
}

type junitTestData struct {
	Data junitTestStructure `json:"data"`
}

type jaggedJunitTestData struct {
	Data []junitTestStructure `json:"data"`
}

type junitTestStructure struct {
	Tests       []test       `json:"tests"`
	FailedTests []failedTest `json:"failedTests"`
	SkippedTest []test       `json:"skippedTests"`
}

type failedTest struct {
	Name       string `json:"name"`
	ErrMessage string `json:"error"`
}

type test struct {
	Name string `json:"name"`
}

func TestJunitPrinter_Print_JaggedArray(t *testing.T) {
	expectedOutput := `<testsuite name="testsuite" tests="4" failures="2" skipped="0" errors="0">
  <testcase name="test1" classname=""></testcase>
  <testcase name="test2" classname="">
    <failure>err msg 2</failure>
  </testcase>
  <testcase name="test3" classname=""></testcase>
  <testcase name="test4" classname="">
    <failure>err msg 4</failure>
  </testcase>
</testsuite>`
	jsonExpr := map[string]string{
		JUnitTestCasesExpressionKey:            "data.#.tests.#.name",
		JUnitFailedTestCasesExpressionKey:      "data.#.failedTests.#.name",
		JUnitFailedTestCaseErrMsgExpressionKey: "data.#.failedTests.#.error",
		JUnitSkippedTestCasesExpressionKey:     "data.#.skippedTests.#.name",
	}
	p := NewJUnitPrinter("testsuite", jsonExpr)
	testObj := &jaggedJunitTestData{
		Data: []junitTestStructure{{
			Tests: []test{
				{Name: "test1"},
				{Name: "test2"},
			},
			FailedTests: []failedTest{
				{Name: "test2", ErrMessage: "err msg 2"},
			},
		}, {
			Tests: []test{
				{Name: "test3"},
				{Name: "test4"},
			},
			FailedTests: []failedTest{
				{Name: "test4", ErrMessage: "err msg 4"},
			},
		},
		}}
	out := bytes.Buffer{}
	err := p.Print(testObj, &out)
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, out.String())

	// check that we can ingest the JUnit report and evaluate its content
	suite, err := parseJUnitXML(out.Bytes())
	require.NoError(t, err)

	assert.Equal(t, 4, suite.Tests)
	assert.Equal(t, 2, suite.Failures)
	assert.Equal(t, 0, suite.Skipped)
	assert.Equal(t, 0, suite.Errors)
	assert.Equal(t, "testsuite", suite.Name)
	for i, tc := range suite.Cases {
		testData := testObj.Data[i/len(testObj.Data)]
		assert.Equal(t, testData.Tests[i%len(testData.Tests)].Name, tc.Name)
		for _, failedTest := range testData.FailedTests {
			if tc.Name == failedTest.Name {
				require.NotNil(t, tc.Failure)
				assert.Equal(t, failedTest.ErrMessage, tc.Failure.Message)
			}
		}
	}
}

func TestJunitPrinter_Print(t *testing.T) {
	expectedOutput := `<testsuite name="testsuite" tests="6" failures="2" skipped="2" errors="0">
  <testcase name="test1" classname=""></testcase>
  <testcase name="test2" classname="">
    <failure>err msg 2</failure>
  </testcase>
  <testcase name="test3" classname=""></testcase>
  <testcase name="test4" classname="">
    <failure>err msg 4</failure>
  </testcase>
  <testcase name="test5" classname="">
    <skipped></skipped>
  </testcase>
  <testcase name="test6" classname="">
    <skipped></skipped>
  </testcase>
</testsuite>`
	jsonExpr := map[string]string{
		JUnitTestCasesExpressionKey:            "data.tests.#.name",
		JUnitFailedTestCasesExpressionKey:      "data.failedTests.#.name",
		JUnitFailedTestCaseErrMsgExpressionKey: "data.failedTests.#.error",
		JUnitSkippedTestCasesExpressionKey:     "data.skippedTests.#.name",
	}
	p := NewJUnitPrinter("testsuite", jsonExpr)
	testObj := &junitTestData{
		Data: junitTestStructure{
			Tests: []test{
				{Name: "test1"},
				{Name: "test2"},
				{Name: "test3"},
				{Name: "test4"},
				{Name: "test5"},
				{Name: "test6"},
			},
			FailedTests: []failedTest{
				{Name: "test2", ErrMessage: "err msg 2"},
				{Name: "test4", ErrMessage: "err msg 4"},
			},
			SkippedTest: []test{
				{Name: "test5"},
				{Name: "test6"},
			},
		},
	}
	out := bytes.Buffer{}
	err := p.Print(testObj, &out)
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, out.String())

	// check that we can ingest the JUnit report and evaluate its content
	suite, err := parseJUnitXML(out.Bytes())
	require.NoError(t, err)

	assert.Equal(t, 6, suite.Tests)
	assert.Equal(t, 2, suite.Failures)
	assert.Equal(t, 2, suite.Skipped)
	assert.Equal(t, 0, suite.Errors)
	assert.Equal(t, "testsuite", suite.Name)
	for i, tc := range suite.Cases {
		assert.Equal(t, testObj.Data.Tests[i].Name, tc.Name)
		for _, failedTest := range testObj.Data.FailedTests {
			if tc.Name == failedTest.Name {
				require.NotNil(t, tc.Failure)
				assert.Equal(t, failedTest.ErrMessage, tc.Failure.Message)
			}
		}
		for _, skippedTest := range testObj.Data.SkippedTest {
			if tc.Name == skippedTest.Name {
				assert.NotNil(t, tc.Skipped)
			}
		}
	}
}

func TestValidateJUnitSuiteData(t *testing.T) {
	cases := map[string]struct {
		tcNames        []string
		failedTcNames  []string
		failedTcErrMsg []string
		skippedTcNames []string
		shouldFail     bool
		error          error
	}{
		"should not fail if: overall test cases >= failed test cases && failed test cases == err messages": {
			tcNames:        []string{"a", "b", "c"},
			failedTcNames:  []string{"a"},
			failedTcErrMsg: []string{"a"},
			skippedTcNames: []string{"b"},
		},
		"should not fail if no skipped test cases and no failed test cases and error messages are given": {
			tcNames:        []string{"a", "b", "c"},
			failedTcNames:  nil,
			failedTcErrMsg: nil,
		},
		"should fail if overall test cases < failed test cases": {
			tcNames:        []string{"a"},
			failedTcNames:  []string{"a", "b"},
			failedTcErrMsg: []string{"a", "b"},
			shouldFail:     true,
			error:          errox.InvariantViolation,
		},
		"should fail if overall test cases < skipped test cases": {
			tcNames:        []string{"a"},
			skippedTcNames: []string{"a", "b"},
			shouldFail:     true,
			error:          errox.InvariantViolation,
		},
		"should fail if failed test cases != error messages": {
			tcNames:        []string{"a", "b", "c"},
			failedTcNames:  []string{"a", "b"},
			failedTcErrMsg: []string{"a", "b", "c"},
			shouldFail:     true,
			error:          errox.InvariantViolation,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateJUnitSuiteData(c.tcNames, c.failedTcNames, c.failedTcErrMsg, c.skippedTcNames)
			if c.shouldFail {
				require.Error(t, err)
				assert.ErrorIs(t, err, c.error)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateFailedTestCaseMap(t *testing.T) {
	cases := map[string]struct {
		failedTc       []string
		failedTcErrMsg []string
		shouldFail     bool
		error          error
		expectedOutput map[string]string
	}{
		"should not fail with unique test case names": {
			failedTc:       []string{"a", "b", "c"},
			failedTcErrMsg: []string{"aa", "bb", "cc"},
			expectedOutput: map[string]string{
				"a": "aa",
				"b": "bb",
				"c": "cc",
			},
		},
		"should fail with non-unique test case names": {
			failedTc:       []string{"a", "b", "b", "c"},
			failedTcErrMsg: []string{"aa", "bb", "cc", "dd"},
			shouldFail:     true,
			error:          errox.InvariantViolation,
			expectedOutput: nil,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := createFailedTestCaseMap(c.failedTc, c.failedTcErrMsg)
			if c.shouldFail {
				require.Error(t, err)
				assert.ErrorIs(t, err, c.error)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, c.expectedOutput, res)
		})
	}
}
