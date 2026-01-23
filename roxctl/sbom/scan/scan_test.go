package scan

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	cliIO "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/common/scan"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	testSBOMScanResponse = &v1.SBOMScanResponse{
		Scan: &v1.SBOMScanResponse_SBOMScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "test-component",
					Version: "1.0.0",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:      "CVE-2024-1234",
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							Cvss:     9.0,
							Link:     "https://nvd.nist.gov/vuln/detail/CVE-2024-1234",
							Advisory: &storage.Advisory{
								Name: "ADVISORY-2024-1234",
								Link: "https://example.com/advisory/2024-1234",
							},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.0.1",
							},
						},
					},
				},
			},
		},
	}
)

func TestSBOMScanCommand(t *testing.T) {
	suite.Run(t, new(sbomScanTestSuite))
}

type sbomScanTestSuite struct {
	suite.Suite
	tempDir string
}

func (s *sbomScanTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "sbom-scan-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *sbomScanTestSuite) TearDownTest() {
	if s.tempDir != "" {
		_ = os.RemoveAll(s.tempDir)
	}
}

func (s *sbomScanTestSuite) createTempFile(name string, content string) string {
	filePath := filepath.Join(s.tempDir, name)
	err := os.WriteFile(filePath, []byte(content), 0644)
	s.Require().NoError(err)
	return filePath
}

func (s *sbomScanTestSuite) TestConstruct() {
	jsonFactory := printer.NewJSONPrinterFactory(false, false)
	jsonPrinter, err := jsonFactory.CreatePrinter("json")
	s.Require().NoError(err)

	validObjPrinterFactory, err := printer.NewObjectPrinterFactory("json", jsonFactory)
	s.Require().NoError(err)

	emptyOutputFormatPrinterFactory, err := printer.NewObjectPrinterFactory("json", jsonFactory)
	s.Require().NoError(err)
	emptyOutputFormatPrinterFactory.OutputFormat = ""

	cases := map[string]struct {
		printerFactory     *printer.ObjectPrinterFactory
		standardizedFormat bool
		noOutputFormat     bool
		expectedPrinter    printer.ObjectPrinter
		httpClientErr      error
		shouldFail         bool
	}{
		"valid printer factory with output format": {
			printerFactory:     validObjPrinterFactory,
			standardizedFormat: true,
			expectedPrinter:    jsonPrinter,
			noOutputFormat:     false,
		},
		"empty output format should set noOutputFormat flag": {
			printerFactory: emptyOutputFormatPrinterFactory,
			noOutputFormat: true,
		},
		"HTTP client error should propagate": {
			printerFactory: validObjPrinterFactory,
			httpClientErr:  errors.New("http client creation failed"),
			shouldFail:     true,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()

			mockEnv := mocks.NewMockEnvironment(ctrl)
			mockClient := &mockHTTPClient{}

			// Mock the HTTPClient call.
			if c.httpClientErr != nil {
				mockEnv.EXPECT().HTTPClient(gomock.Any(), gomock.Any()).Return(nil, c.httpClientErr)
			} else {
				mockEnv.EXPECT().HTTPClient(gomock.Any(), gomock.Any()).Return(mockClient, nil)
			}

			// Create command with timeout flag to avoid panic.
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().Duration("timeout", 0, "")

			sbomScanCmd := &sbomScanCommand{env: mockEnv}

			err := sbomScanCmd.Construct(nil, cmd, c.printerFactory)

			if c.shouldFail {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Assert().Equal(c.expectedPrinter, sbomScanCmd.printer)
				s.Assert().Equal(c.standardizedFormat, sbomScanCmd.standardizedFormat)
				s.Assert().Equal(c.noOutputFormat, sbomScanCmd.noOutputFormat)
				s.Assert().NotNil(sbomScanCmd.client)
			}
		})
	}
}
func (s *sbomScanTestSuite) TestValidate() {
	cases := map[string]struct {
		setupFunc  func() string
		severities []string
		shouldFail bool
		errorType  error
	}{
		"valid SBOM file exists": {
			setupFunc: func() string {
				return s.createTempFile("valid.json", `{"spdxVersion":"SPDX-2.3"}`)
			},
			severities: scan.AllSeverities(),
			shouldFail: false,
		},
		"SBOM file does not exist": {
			setupFunc: func() string {
				return filepath.Join(s.tempDir, "nonexistent.json")
			},
			severities: scan.AllSeverities(),
			shouldFail: true,
			errorType:  errox.InvalidArgs,
		},
		"valid severities with mixed case": {
			setupFunc: func() string {
				return s.createTempFile("valid.json", `{"spdxVersion":"SPDX-2.3"}`)
			},
			severities: []string{"critical", "IMPORTANT", "low"},
			shouldFail: false,
		},
		"invalid severity": {
			setupFunc: func() string {
				return s.createTempFile("valid.json", `{"spdxVersion":"SPDX-2.3"}`)
			},
			severities: []string{"CRITICAL", "INVALID_SEVERITY"},
			shouldFail: true,
			errorType:  errox.InvalidArgs,
		},
		"single valid severity": {
			setupFunc: func() string {
				return s.createTempFile("valid.json", `{"spdxVersion":"SPDX-2.3"}`)
			},
			severities: []string{"CRITICAL"},
			shouldFail: false,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			sbomPath := c.setupFunc()
			cmd := &sbomScanCommand{
				sbomFilePath: sbomPath,
				severities:   c.severities,
			}

			err := cmd.Validate()
			if c.shouldFail {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, c.errorType)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *sbomScanTestSuite) TestScanSBOM_Success() {
	// Create a valid SBOM file.
	sbomContent := `{"spdxVersion": "SPDX-2.3", "name": "test-sbom"}`
	sbomPath := s.createTempFile("sbom.json", sbomContent)

	responseData, err := protojson.Marshal(testSBOMScanResponse)
	s.Require().NoError(err)

	mockClient := &mockHTTPClient{
		newReqFunc: func(method string, path string, body io.Reader) (*http.Request, error) {
			s.Assert().Equal(http.MethodPost, method)
			s.Assert().Equal(sbomScanAPIPath, path)
			return http.NewRequest(method, "http://localhost"+path, body)
		},
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Verify the request has the correct content type.
			s.Assert().Equal("text/spdx+json", req.Header.Get("Content-Type"))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(responseData)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			}, nil
		},
	}

	// Create command with raw output (no formatting).
	testIO, _, out, _ := cliIO.TestIO()
	env := environment.NewTestCLIEnvironment(s.T(), testIO, printer.DefaultColorPrinter())

	cmd := &sbomScanCommand{
		sbomFilePath:   sbomPath,
		env:            env,
		client:         mockClient,
		noOutputFormat: true,
		severities:     scan.AllSeverities(),
	}

	err = cmd.ScanSBOM()
	s.Require().NoError(err)

	// Verify raw output was written.
	s.Assert().Contains(out.String(), "test-component")
}

func (s *sbomScanTestSuite) TestScanSBOM_FileNotFound() {
	sbomPath := filepath.Join(s.tempDir, "nonexistent.json")

	io, _, _, _ := cliIO.TestIO()
	env := environment.NewTestCLIEnvironment(s.T(), io, printer.DefaultColorPrinter())

	cmd := &sbomScanCommand{
		sbomFilePath: sbomPath,
		env:          env,
	}

	err := cmd.ScanSBOM()
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "opening SBOM file")
}

func (s *sbomScanTestSuite) TestScanSBOM_NonOKStatusCode() {
	sbomContent := `{"spdxVersion": "SPDX-2.3", "name": "test"}`
	sbomPath := s.createTempFile("sbom.json", sbomContent)

	mockClient := &mockHTTPClient{
		newReqFunc: func(method string, path string, body io.Reader) (*http.Request, error) {
			return http.NewRequest(method, "http://localhost"+path, body)
		},
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("Bad request")),
			}, nil
		},
	}

	io, _, _, _ := cliIO.TestIO()
	env := environment.NewTestCLIEnvironment(s.T(), io, printer.DefaultColorPrinter())

	cmd := &sbomScanCommand{
		sbomFilePath: sbomPath,
		env:          env,
		client:       mockClient,
	}

	err := cmd.ScanSBOM()
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "400")
	s.Assert().Contains(err.Error(), "Bad request")
}

func (s *sbomScanTestSuite) TestScanSBOM_HTMLContentType() {
	sbomContent := `{"spdxVersion": "SPDX-2.3", "name": "test"}`
	sbomPath := s.createTempFile("sbom.json", sbomContent)

	mockClient := &mockHTTPClient{
		newReqFunc: func(method string, path string, body io.Reader) (*http.Request, error) {
			return http.NewRequest(method, "http://localhost"+path, body)
		},
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("<html>Not Found</html>")),
				Header: http.Header{
					"Content-Type": []string{"text/html"},
				},
			}, nil
		},
	}

	io, _, _, _ := cliIO.TestIO()
	env := environment.NewTestCLIEnvironment(s.T(), io, printer.DefaultColorPrinter())

	cmd := &sbomScanCommand{
		sbomFilePath: sbomPath,
		env:          env,
		client:       mockClient,
	}

	err := cmd.ScanSBOM()
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "text/html")
	s.Assert().Contains(err.Error(), "confirm Central version supports SBOM scanning")
}

func (s *sbomScanTestSuite) TestScanSBOM_WithContentTypeFlag() {
	sbomContent := `{"spdxVersion": "SPDX-2.3", "name": "test"}`
	sbomPath := s.createTempFile("sbom.json", sbomContent)

	mockClient := &mockHTTPClient{
		newReqFunc: func(method string, path string, body io.Reader) (*http.Request, error) {
			return http.NewRequest(method, "http://localhost"+path, body)
		},
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Verify custom content type is used.
			s.Assert().Equal("application/custom+json", req.Header.Get("Content-Type"))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			}, nil
		},
	}

	io, _, _, _ := cliIO.TestIO()
	env := environment.NewTestCLIEnvironment(s.T(), io, printer.DefaultColorPrinter())

	cmd := &sbomScanCommand{
		sbomFilePath:   sbomPath,
		contentType:    "application/custom+json",
		env:            env,
		client:         mockClient,
		noOutputFormat: true,
	}

	err := cmd.ScanSBOM()
	s.Require().NoError(err)
}

func (s *sbomScanTestSuite) TestScanSBOM_FailOnFinding() {
	sbomContent := `{"spdxVersion": "SPDX-2.3", "name": "test"}`
	sbomPath := s.createTempFile("sbom.json", sbomContent)

	responseData, err := protojson.Marshal(testSBOMScanResponse)
	s.Require().NoError(err)

	mockClient := &mockHTTPClient{
		newReqFunc: func(method string, path string, body io.Reader) (*http.Request, error) {
			return http.NewRequest(method, "http://localhost"+path, body)
		},
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(responseData)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			}, nil
		},
	}

	jsonFactory := printer.NewJSONPrinterFactory(false, false)
	jsonPrinter, err := jsonFactory.CreatePrinter("json")
	s.Require().NoError(err)

	io, _, _, _ := cliIO.TestIO()
	env := environment.NewTestCLIEnvironment(s.T(), io, printer.DefaultColorPrinter())

	cmd := &sbomScanCommand{
		sbomFilePath:       sbomPath,
		env:                env,
		client:             mockClient,
		failOnFinding:      true,
		printer:            jsonPrinter,
		standardizedFormat: true,
		severities:         scan.AllSeverities(),
	}

	err = cmd.ScanSBOM()
	s.Require().Error(err)
	s.Assert().ErrorIs(err, scan.ErrVulnerabilityFound)
}

func (s *sbomScanTestSuite) TestGuessMediaType() {
	cases := map[string]struct {
		content       string
		expectedType  string
		shouldFail    bool
		errorContains string
	}{
		"valid SPDX 2.3 JSON": {
			content:      `{"spdxVersion": "SPDX-2.3", "name": "test"}`,
			expectedType: "text/spdx+json",
		},
		"valid SPDX 2.3 JSON with whitespace": {
			content:      `  {  "spdxVersion"  :  "SPDX-2.3"  }  `,
			expectedType: "text/spdx+json",
		},
		"valid SPDX 2.3 JSON with newlines": {
			content: `{
				"spdxVersion": "SPDX-2.3",
				"name": "test"
			}`,
			expectedType: "text/spdx+json",
		},
		"valid SPDX 2.3 JSON with UTF-8 BOM": {
			content:      "\xEF\xBB\xBF{\"spdxVersion\": \"SPDX-2.3\"}",
			expectedType: "text/spdx+json",
		},
		"SPDX 2.2 should fail": {
			content:       `{"spdxVersion": "SPDX-2.2"}`,
			shouldFail:    true,
			errorContains: "unsupported SBOM format",
		},
		"not JSON format": {
			content:       `not json content`,
			shouldFail:    true,
			errorContains: "does not appear to be valid JSON",
		},
		"JSON array instead of object": {
			content:       `["item1", "item2"]`,
			shouldFail:    true,
			errorContains: "unsupported SBOM format",
		},
		"missing spdxVersion field": {
			content:       `{"name": "test", "version": "1.0"}`,
			shouldFail:    true,
			errorContains: "unsupported SBOM format",
		},
		"empty file": {
			content:       ``,
			shouldFail:    true,
			errorContains: "does not appear to be valid JSON",
		},
		"spdxVersion with tabs": {
			content:      "{\t\"spdxVersion\"\t:\t\"SPDX-2.3\"\t}",
			expectedType: "text/spdx+json",
		},
		"spdxVersion with mixed whitespace": {
			content:      "{ \n\t\"spdxVersion\" \n\t: \n\t\"SPDX-2.3\" }",
			expectedType: "text/spdx+json",
		},
		"just opening brace": {
			content:       "{",
			shouldFail:    true,
			errorContains: "unsupported SBOM format",
		},
		"only whitespace": {
			content:       "   \n\t  ",
			shouldFail:    true,
			errorContains: "does not appear to be valid JSON",
		},
		"spdxVersion inside string field": {
			content:       `{"description": "this does NOT conform to \"spdxVersion\": \"SPDX-2.3\" spec"}`,
			shouldFail:    true,
			errorContains: "unsupported SBOM format",
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			filePath := s.createTempFile("sbom.json", c.content)
			file, err := os.Open(filePath)
			s.Require().NoError(err)
			defer utils.IgnoreError(file.Close)

			mediaType, err := guessMediaType(file)

			if c.shouldFail {
				s.Require().Error(err)
				if c.errorContains != "" {
					s.Assert().Contains(err.Error(), c.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Assert().Equal(c.expectedType, mediaType)

				// Verify file pointer is reset to beginning.
				pos, err := file.Seek(0, io.SeekCurrent)
				s.Require().NoError(err)
				s.Assert().Equal(int64(0), pos, "file pointer should be reset to beginning")
			}
		})
	}
}

func (s *sbomScanTestSuite) TestGuessMediaType_LargeFile() {
	// Create content with spdxVersion field after 4KB but starting with valid JSON opening.
	largeContent := `{` + strings.Repeat(" ", 5000) + `"spdxVersion": "SPDX-2.3"}`
	filePath := s.createTempFile("large.json", largeContent)

	file, err := os.Open(filePath)
	s.Require().NoError(err)
	defer utils.IgnoreError(file.Close)

	// Should fail because spdxVersion is beyond the 4KB read buffer.
	_, err = guessMediaType(file)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "unsupported SBOM format")
}

// TestSBOMScanResponseCompatibilityWithStorageImage verifies that v1.SBOMScanResponse
// can be marshaled to JSON and then unmarshaled into storage.Image while preserving
// all fields required by the printing/formatting logic (see roxctl/common/scan/cve.go).
//
// This contract verification is critical because printSBOMScanResults unmarshals the
// API response into storage.Image to reuse existing formatting logic.
//
// The following vulnerability fields are used by the printer:
//   - vulnerability.GetCve()                  → CveID
//   - vulnerability.GetSeverity()             → CveSeverity
//   - vulnerability.GetCvss()                 → CveCVSS
//   - vulnerability.GetLink()                 → CveInfo
//   - vulnerability.GetAdvisory().GetName()   → AdvisoryID
//   - vulnerability.GetAdvisory().GetLink()   → AdvisoryInfo
//   - comp.GetName()                          → ComponentName
//   - comp.GetVersion()                       → ComponentVersion
//   - vulnerability.GetFixedBy()              → ComponentFixedVersion
//
// If any of these fields fail to marshal/unmarshal correctly, that is a breaking change
// which will trigger this test to fail.
func (s *sbomScanTestSuite) TestSBOMScanResponseCompatibilityWithStorageImage() {
	// Marshal the SBOMScanResponse to JSON
	sbomJSON, err := protojson.Marshal(testSBOMScanResponse)
	s.Require().NoError(err, "failed to marshal SBOMScanResponse to JSON")

	// Unmarshal into storage.Image (this is what printSBOMScanResults does).
	var image storage.Image
	err = protojson.Unmarshal(sbomJSON, &image)
	s.Require().NoError(err, "failed to unmarshal SBOMScanResponse JSON into storage.Image - this indicates a breaking compatibility issue")

	// Verify the critical fields are preserved.
	s.Require().NotNil(image.GetScan(), "Image.Scan should not be nil")
	s.Require().NotNil(image.GetScan().GetComponents(), "Image.Scan.Components should not be nil")

	// Verify component data is intact (used by printer).
	s.Require().Len(image.GetScan().GetComponents(), 1, "should have 1 component")
	component := image.GetScan().GetComponents()[0]
	s.Assert().Equal("test-component", component.GetName(), "component name should be preserved (used by printer)")
	s.Assert().Equal("1.0.0", component.GetVersion(), "component version should be preserved (used by printer)")

	// Verify vulnerability data is intact (used by printer).
	s.Require().Len(component.GetVulns(), 1, "should have 1 vulnerability")
	vuln := component.GetVulns()[0]
	s.Assert().Equal("CVE-2024-1234", vuln.GetCve(), "CVE ID should be preserved (used by printer)")
	s.Assert().Equal(storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, vuln.GetSeverity(), "severity should be preserved (used by printer)")
	s.Assert().Equal(float32(9.0), vuln.GetCvss(), "CVSS score should be preserved (used by printer)")
	s.Assert().Equal("https://nvd.nist.gov/vuln/detail/CVE-2024-1234", vuln.GetLink(), "link should be preserved (used by printer)")

	// Verify Advisory data is intact (used by printer).
	s.Require().NotNil(vuln.GetAdvisory(), "Advisory should not be nil (used by printer)")
	s.Assert().Equal("ADVISORY-2024-1234", vuln.GetAdvisory().GetName(), "Advisory name should be preserved (used by printer)")
	s.Assert().Equal("https://example.com/advisory/2024-1234", vuln.GetAdvisory().GetLink(), "Advisory link should be preserved (used by printer)")

	// Verify FixedBy data (used by printer).
	s.Require().NotNil(vuln.GetSetFixedBy(), "SetFixedBy should not be nil (used by printer)")
	fixedBy, ok := vuln.GetSetFixedBy().(*storage.EmbeddedVulnerability_FixedBy)
	s.Require().True(ok, "SetFixedBy should be of type *storage.EmbeddedVulnerability_FixedBy")
	s.Assert().Equal("1.0.1", fixedBy.FixedBy, "FixedBy version should be preserved (used by printer)")

	// Test that the formatted output works correctly.
	cveSummary := scan.NewCVESummaryForPrinting(image.GetScan(), scan.AllSeverities())
	s.Assert().Equal(1, cveSummary.CountVulnerabilities(), "should count 1 vulnerability")
	s.Assert().Equal(1, cveSummary.CountComponents(), "should count 1 component")
}

func (s *sbomScanTestSuite) TestPrintSBOMScanResults_FormattedOutput() {
	responseData, err := protojson.Marshal(testSBOMScanResponse)
	s.Require().NoError(err)

	jsonFactory := printer.NewJSONPrinterFactory(false, false)
	jsonPrinter, err := jsonFactory.CreatePrinter("json")
	s.Require().NoError(err)

	testIO, _, out, _ := cliIO.TestIO()
	env := environment.NewTestCLIEnvironment(s.T(), testIO, printer.DefaultColorPrinter())

	cmd := &sbomScanCommand{
		env:                env,
		printer:            jsonPrinter,
		standardizedFormat: true,
		severities:         scan.AllSeverities(),
		sbomFilePath:       "test.json",
	}

	err = cmd.printSBOMScanResults(bytes.NewReader(responseData))
	s.Require().NoError(err)
	s.Assert().Contains(out.String(), "CVE-2024-1234")
	s.Assert().Contains(out.String(), "test-component")
}

func (s *sbomScanTestSuite) TestPrintSBOMScanResults_InvalidJSON() {
	jsonFactory := printer.NewJSONPrinterFactory(false, false)
	jsonPrinter, err := jsonFactory.CreatePrinter("json")
	s.Require().NoError(err)

	io, _, _, _ := cliIO.TestIO()
	env := environment.NewTestCLIEnvironment(s.T(), io, printer.DefaultColorPrinter())

	cmd := &sbomScanCommand{
		env:                env,
		printer:            jsonPrinter,
		standardizedFormat: true,
	}

	err = cmd.printSBOMScanResults(strings.NewReader("invalid json"))
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "unmarshalling response")
}

// mockHTTPClient is a mock implementation of RoxctlHTTPClient interface.
type mockHTTPClient struct {
	doFunc             func(req *http.Request) (*http.Response, error)
	newReqFunc         func(method string, path string, body io.Reader) (*http.Request, error)
	doReqAndVerifyFunc func(path string, method string, code int, body io.Reader) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return nil, errors.New("Do not implemented")
}

func (m *mockHTTPClient) NewReq(method string, path string, body io.Reader) (*http.Request, error) {
	if m.newReqFunc != nil {
		return m.newReqFunc(method, path, body)
	}
	return http.NewRequest(method, "http://localhost"+path, body)
}

func (m *mockHTTPClient) DoReqAndVerifyStatusCode(path string, method string, code int, body io.Reader) (*http.Response, error) {
	if m.doReqAndVerifyFunc != nil {
		return m.doReqAndVerifyFunc(path, method, code, body)
	}
	return nil, errors.New("DoReqAndVerifyStatusCode not implemented")
}
