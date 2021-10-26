package scan

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

var (
	lowSeverityCVEs = [3]*storage.EmbeddedVulnerability{
		{
			Cve:        "CVE-123-LOW",
			Cvss:       2.0,
			Summary:    "This is a low CVE 1",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1"},
			Severity:   storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		{
			Cve:        "CVE-456-LOW",
			Cvss:       2.9,
			Summary:    "This is a low CVE 2",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.2"},
			Severity:   storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		{
			Cve:        "CVE-789-LOW",
			Cvss:       2.5,
			Summary:    "This is a low CVE 3",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.3"},
			Severity:   storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
	}
	moderateSeverityCVEs = [3]*storage.EmbeddedVulnerability{
		{
			Cve:        "CVE-123-MED",
			Cvss:       4.5,
			Summary:    "This is a mod CVE 1",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1"},
			Severity:   storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		},
		{
			Cve:        "CVE-456-MED",
			Cvss:       4.9,
			Summary:    "This is a mod CVE 2",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.2"},
			Severity:   storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		},
		{
			Cve:        "CVE-789-MED",
			Cvss:       5.2,
			Summary:    "This is a mod CVE 3",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.3"},
			Severity:   storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		},
	}
	importantSeverityCVEs = [3]*storage.EmbeddedVulnerability{
		{
			Cve:        "CVE-123-IMP",
			Cvss:       7.0,
			Summary:    "This is a imp CVE 1",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1"},
			Severity:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		},
		{
			Cve:        "CVE-456-IMP",
			Cvss:       6.8,
			Summary:    "This is a imp CVE 2",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.2"},
			Severity:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		},
		{
			Cve:        "CVE-789-IMP",
			Cvss:       7.0,
			Summary:    "This is a imp CVE 3",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.3"},
			Severity:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		},
	}
	criticalSeverityCVEs = [3]*storage.EmbeddedVulnerability{
		{
			Cve:        "CVE-123-CRIT",
			Cvss:       8.5,
			Summary:    "This is a crit CVE 1",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1"},
			Severity:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		},
		{
			Cve:        "CVE-456-CRIT",
			Cvss:       9.0,
			Summary:    "This is a crit CVE 2",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.2"},
			Severity:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		},
		{
			Cve:        "CVE-789-CRIT",
			Cvss:       9.5,
			Summary:    "This is a crit CVE 3",
			Link:       "<some-link-to-nvd>",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.3"},
			Severity:   storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		},
	}
	testComponents = []*storage.EmbeddedImageScanComponent{
		{
			Name:    "apt",
			Version: "1.0",
			FixedBy: "1.4",
			Vulns: []*storage.EmbeddedVulnerability{
				lowSeverityCVEs[0],
				lowSeverityCVEs[2],
				criticalSeverityCVEs[2],
			},
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0},
		},
		{
			Name:          "systemd",
			Version:       "1.3-debu49",
			FixedBy:       "1.3-debu102",
			Vulns:         moderateSeverityCVEs[:],
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1},
		},
		{
			Name:          "curl",
			Version:       "7.0-rc1",
			FixedBy:       "7.1-rc2",
			Vulns:         importantSeverityCVEs[:],
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1},
		},
		{
			Name:          "bash",
			Version:       "4.2",
			FixedBy:       "4.3",
			Vulns:         criticalSeverityCVEs[:],
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2},
		},
		{
			Name:    "openssl",
			Version: "1.1.1k",
			Vulns: []*storage.EmbeddedVulnerability{
				lowSeverityCVEs[0],
				moderateSeverityCVEs[0],
				moderateSeverityCVEs[1],
				importantSeverityCVEs[0],
				criticalSeverityCVEs[2],
			},
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2},
		},
	}
)

// mock implementation for v1.ImageServiceServer
type mockImageServiceServer struct {
	v1.ImageServiceServer
	components []*storage.EmbeddedImageScanComponent
}

func (m *mockImageServiceServer) ScanImage(ctx context.Context, in *v1.ScanImageRequest) (*storage.Image, error) {
	img := &storage.Image{
		Scan: &storage.ImageScan{
			Components: m.components,
		},
		Metadata: &storage.ImageMetadata{V1: &storage.V1Metadata{
			Layers: []*storage.ImageLayer{
				{
					Instruction: "layer1",
					Value:       "1",
				},
				{
					Instruction: "layer2",
					Value:       "2",
				},
				{
					Instruction: "layer3",
					Value:       "3",
				},
			},
		}},
	}
	return img, nil
}

func TestImageScanCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(imageScanTestSuite))
}

type imageScanTestSuite struct {
	suite.Suite
	defaultImageScanCommand imageScanCommand
}

type closeFunction = func()

// createGRPCMockImageService will create an in-memory gRPC server serving a mockImageServiceServer
// which will respond with the injected components. A valid grpc.ClientConn for the grpc server will be
// returned as well as a closeFunction to stop the server and in-memory listener
// NOTE: Ensure that you ALWAYS call the closeFunction to clean up the test setup
func (s *imageScanTestSuite) createGRPCMockImageService(components []*storage.EmbeddedImageScanComponent) (*grpc.ClientConn, closeFunction) {
	// create an in-memory listener that does not require exposing any ports on the host
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	v1.RegisterImageServiceServer(server, &mockImageServiceServer{components: components})

	// start the server
	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithInsecure())
	s.Require().NoError(err)

	closeF := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, closeF
}

func (s *imageScanTestSuite) newTestMockEnvironmentWithConn(conn *grpc.ClientConn) (environment.Environment, *bytes.Buffer) {
	mockEnv := mocks.NewMockEnvironment(gomock.NewController(s.T()))

	testIO, _, testStdOut, _ := environment.TestIO()
	mockEnv.EXPECT().InputOutput().AnyTimes().Return(testIO)
	mockEnv.EXPECT().GRPCConnection().AnyTimes().Return(conn, nil)
	return mockEnv, testStdOut
}

func (s *imageScanTestSuite) SetupTest() {
	s.defaultImageScanCommand = imageScanCommand{
		image:      "nginx:test",
		retryDelay: 3,
		timeout:    1 * time.Minute,
	}
}

func (s *imageScanTestSuite) TestConstruct() {
	jsonFactory := printer.NewJSONPrinterFactory(false, false)
	jsonPrinter, err := jsonFactory.CreatePrinter("json")
	s.Require().NoError(err)

	validObjPrinterFactory, err := printer.NewObjectPrinterFactory("json", jsonFactory)
	s.Require().NoError(err)

	invalidObjPrinterFactory, err := printer.NewObjectPrinterFactory("json", jsonFactory)
	s.Require().NoError(err)
	invalidObjPrinterFactory.OutputFormat = "table"

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Duration("timeout", 1*time.Minute, "")
	cases := map[string]struct {
		legacyFormat       string
		printerFactory     *printer.ObjectPrinterFactory
		standardizedFormat bool
		printer            printer.ObjectPrinter
		shouldFail         bool
		error              error
	}{
		"new output format and valid default values": {
			printerFactory:     validObjPrinterFactory,
			standardizedFormat: true,
			printer:            jsonPrinter,
		},
		"legacy output format should never create printers": {
			legacyFormat:   "json",
			printerFactory: validObjPrinterFactory,
		},
		"invalid printer factory should return an error": {
			printerFactory: invalidObjPrinterFactory,
			shouldFail:     true,
			error:          errorhelpers.ErrInvalidArgs,
		},
		"legacy output format should never throw an error when invalid object printer factory is used": {
			legacyFormat:   "json",
			printerFactory: invalidObjPrinterFactory,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			imgScanCmd := s.defaultImageScanCommand
			imgScanCmd.format = c.legacyFormat

			err := imgScanCmd.Construct(nil, cmd, c.printerFactory)
			if c.shouldFail {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, c.error)
			} else {
				s.Require().NoError(err)
			}

			s.Assert().Equal(c.printer, imgScanCmd.printer)
			s.Assert().Equal(c.standardizedFormat, imgScanCmd.standardizedFormat)
			s.Assert().Equal(1*time.Minute, imgScanCmd.timeout)
		})
	}
}

func (s *imageScanTestSuite) TestValidate() {
	jsonPrinter, err := printer.NewJSONPrinterFactory(false, false).CreatePrinter("json")
	s.Require().NoError(err)

	cases := map[string]struct {
		image        string
		legacyFormat string
		printer      printer.ObjectPrinter
		shouldFail   bool
		error        error
	}{
		"valid legacy output format and image name should not result in an error": {
			image:        s.defaultImageScanCommand.image,
			legacyFormat: "json",
		},
		"valid new output format and image name should not result in an error": {
			image:   s.defaultImageScanCommand.image,
			printer: jsonPrinter,
		},
		"invalid image name should result in an error": {
			image: "c:",
			error: errorhelpers.ErrInvalidArgs,
		},
		"wrong legacy output format should result in an error when new output format IS NOT used": {
			image:        s.defaultImageScanCommand.image,
			legacyFormat: "table",
			shouldFail:   true,
			error:        errorhelpers.ErrInvalidArgs,
		},
		"wrong legacy output format should NOT result in an error when new output format IS used": {
			image:        s.defaultImageScanCommand.image,
			legacyFormat: "table",
			printer:      jsonPrinter,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			imgScanCmd := s.defaultImageScanCommand
			imgScanCmd.image = c.image
			imgScanCmd.format = c.legacyFormat
			imgScanCmd.printer = c.printer

			err := imgScanCmd.Validate()
			if c.shouldFail {
				s.Require().Error(err)
				s.Assert().ErrorIs(err, c.error)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

type outputFormatTest struct {
	components     []*storage.EmbeddedImageScanComponent
	expectedOutput string
}

func (s *imageScanTestSuite) TestScan_TableOutput() {
	cases := map[string]outputFormatTest{
		"should render default output with merged cells and additional verbose output": {
			components: testComponents,
			expectedOutput: `Scan results for image: nginx:test
(TOTAL-COMPONENTS: 5, TOTAL-VULNERABILITIES: 17, LOW: 3, MODERATE: 0, IMPORTANT: 4, CRITICAL: 5)

+-----------+------------+--------------+-----------+--------------------+
| COMPONENT |  VERSION   |     CVE      | SEVERITY  |        LINK        |
+-----------+------------+--------------+-----------+--------------------+
| apt       |        1.0 | CVE-789-CRIT | CRITICAL  | <some-link-to-nvd> |
+           +            +--------------+-----------+                    +
|           |            | CVE-123-LOW  | LOW       |                    |
+           +            +--------------+           +                    +
|           |            | CVE-789-LOW  |           |                    |
+-----------+------------+--------------+-----------+                    +
| bash      |        4.2 | CVE-123-CRIT | CRITICAL  |                    |
+           +            +--------------+           +                    +
|           |            | CVE-456-CRIT |           |                    |
+           +            +--------------+           +                    +
|           |            | CVE-789-CRIT |           |                    |
+-----------+------------+--------------+-----------+                    +
| curl      | 7.0-rc1    | CVE-123-IMP  | IMPORTANT |                    |
+           +            +--------------+           +                    +
|           |            | CVE-456-IMP  |           |                    |
+           +            +--------------+           +                    +
|           |            | CVE-789-IMP  |           |                    |
+-----------+------------+--------------+-----------+                    +
| openssl   | 1.1.1k     | CVE-789-CRIT | CRITICAL  |                    |
+           +            +--------------+-----------+                    +
|           |            | CVE-123-IMP  | IMPORTANT |                    |
+           +            +--------------+-----------+                    +
|           |            | CVE-123-MED  | MODERATE  |                    |
+           +            +--------------+           +                    +
|           |            | CVE-456-MED  |           |                    |
+           +            +--------------+-----------+                    +
|           |            | CVE-123-LOW  | LOW       |                    |
+-----------+------------+--------------+-----------+                    +
| systemd   | 1.3-debu49 | CVE-123-MED  | MODERATE  |                    |
+           +            +--------------+           +                    +
|           |            | CVE-456-MED  |           |                    |
+           +            +--------------+           +                    +
|           |            | CVE-789-MED  |           |                    |
+-----------+------------+--------------+-----------+--------------------+
WARN: A total of 17 vulnerabilities were found in 5 components
`,
		},
		"should print only headers with empty components in image scan": {
			expectedOutput: `Scan results for image: nginx:test
(TOTAL-COMPONENTS: 0, TOTAL-VULNERABILITIES: 0, LOW: 0, MODERATE: 0, IMPORTANT: 0, CRITICAL: 0)

+-----------+---------+-----+----------+------+
| COMPONENT | VERSION | CVE | SEVERITY | LINK |
+-----------+---------+-----+----------+------+
+-----------+---------+-----+----------+------+
`,
		},
	}

	factory, err := printer.NewObjectPrinterFactory("table", supportedObjectPrinters...)
	s.Require().NoError(err)
	tablePrinter, err := factory.CreatePrinter()
	s.Require().NoError(err)

	s.runOutputTests(cases, tablePrinter, false)
}

func (s *imageScanTestSuite) TestScan_JSONOutput() {
	cases := map[string]outputFormatTest{
		"should render default output non compact without additional verbose output": {
			components: testComponents,
			expectedOutput: `{
  "result": {
    "summary": {
      "CRITICAL": 5,
      "IMPORTANT": 4,
      "LOW": 3,
      "MODERATE": 5,
      "TOTAL-COMPONENTS": 5,
      "TOTAL-VULNERABILITIES": 17
    },
    "vulnerabilities": [
      {
        "cveId": "CVE-789-CRIT",
        "cveSeverity": "CRITICAL",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "apt",
        "componentVersion": "1.0",
        "componentFixedVersion": "1.3"
      },
      {
        "cveId": "CVE-123-LOW",
        "cveSeverity": "LOW",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "apt",
        "componentVersion": "1.0",
        "componentFixedVersion": "1.1"
      },
      {
        "cveId": "CVE-789-LOW",
        "cveSeverity": "LOW",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "apt",
        "componentVersion": "1.0",
        "componentFixedVersion": "1.3"
      },
      {
        "cveId": "CVE-123-CRIT",
        "cveSeverity": "CRITICAL",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "bash",
        "componentVersion": "4.2",
        "componentFixedVersion": "1.1"
      },
      {
        "cveId": "CVE-456-CRIT",
        "cveSeverity": "CRITICAL",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "bash",
        "componentVersion": "4.2",
        "componentFixedVersion": "1.2"
      },
      {
        "cveId": "CVE-789-CRIT",
        "cveSeverity": "CRITICAL",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "bash",
        "componentVersion": "4.2",
        "componentFixedVersion": "1.3"
      },
      {
        "cveId": "CVE-123-IMP",
        "cveSeverity": "IMPORTANT",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "curl",
        "componentVersion": "7.0-rc1",
        "componentFixedVersion": "1.1"
      },
      {
        "cveId": "CVE-456-IMP",
        "cveSeverity": "IMPORTANT",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "curl",
        "componentVersion": "7.0-rc1",
        "componentFixedVersion": "1.2"
      },
      {
        "cveId": "CVE-789-IMP",
        "cveSeverity": "IMPORTANT",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "curl",
        "componentVersion": "7.0-rc1",
        "componentFixedVersion": "1.3"
      },
      {
        "cveId": "CVE-789-CRIT",
        "cveSeverity": "CRITICAL",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "openssl",
        "componentVersion": "1.1.1k",
        "componentFixedVersion": "1.3"
      },
      {
        "cveId": "CVE-123-IMP",
        "cveSeverity": "IMPORTANT",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "openssl",
        "componentVersion": "1.1.1k",
        "componentFixedVersion": "1.1"
      },
      {
        "cveId": "CVE-123-MED",
        "cveSeverity": "MODERATE",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "openssl",
        "componentVersion": "1.1.1k",
        "componentFixedVersion": "1.1"
      },
      {
        "cveId": "CVE-456-MED",
        "cveSeverity": "MODERATE",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "openssl",
        "componentVersion": "1.1.1k",
        "componentFixedVersion": "1.2"
      },
      {
        "cveId": "CVE-123-LOW",
        "cveSeverity": "LOW",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "openssl",
        "componentVersion": "1.1.1k",
        "componentFixedVersion": "1.1"
      },
      {
        "cveId": "CVE-123-MED",
        "cveSeverity": "MODERATE",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "systemd",
        "componentVersion": "1.3-debu49",
        "componentFixedVersion": "1.1"
      },
      {
        "cveId": "CVE-456-MED",
        "cveSeverity": "MODERATE",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "systemd",
        "componentVersion": "1.3-debu49",
        "componentFixedVersion": "1.2"
      },
      {
        "cveId": "CVE-789-MED",
        "cveSeverity": "MODERATE",
        "cveInfo": "<some-link-to-nvd>",
        "componentName": "systemd",
        "componentVersion": "1.3-debu49",
        "componentFixedVersion": "1.3"
      }
    ]
  }
}
`,
		},
		"should print nothing with empty components in image scan": {
			components: nil,
			expectedOutput: `{
  "result": {
    "summary": {
      "CRITICAL": 0,
      "IMPORTANT": 0,
      "LOW": 0,
      "MODERATE": 0,
      "TOTAL-COMPONENTS": 0,
      "TOTAL-VULNERABILITIES": 0
    }
  }
}
`,
		},
	}

	factory, err := printer.NewObjectPrinterFactory("json", supportedObjectPrinters...)
	s.Require().NoError(err)
	jsonPrinter, err := factory.CreatePrinter()
	s.Require().NoError(err)

	s.runOutputTests(cases, jsonPrinter, true)
}

func (s *imageScanTestSuite) TestScan_CSVOutput() {
	cases := map[string]outputFormatTest{
		"should render default output without additional verbose output": {
			components: testComponents,
			expectedOutput: `COMPONENT,VERSION,CVE,SEVERITY,LINK
apt,1.0,CVE-789-CRIT,CRITICAL,<some-link-to-nvd>
apt,1.0,CVE-123-LOW,LOW,<some-link-to-nvd>
apt,1.0,CVE-789-LOW,LOW,<some-link-to-nvd>
bash,4.2,CVE-123-CRIT,CRITICAL,<some-link-to-nvd>
bash,4.2,CVE-456-CRIT,CRITICAL,<some-link-to-nvd>
bash,4.2,CVE-789-CRIT,CRITICAL,<some-link-to-nvd>
curl,7.0-rc1,CVE-123-IMP,IMPORTANT,<some-link-to-nvd>
curl,7.0-rc1,CVE-456-IMP,IMPORTANT,<some-link-to-nvd>
curl,7.0-rc1,CVE-789-IMP,IMPORTANT,<some-link-to-nvd>
openssl,1.1.1k,CVE-789-CRIT,CRITICAL,<some-link-to-nvd>
openssl,1.1.1k,CVE-123-IMP,IMPORTANT,<some-link-to-nvd>
openssl,1.1.1k,CVE-123-MED,MODERATE,<some-link-to-nvd>
openssl,1.1.1k,CVE-456-MED,MODERATE,<some-link-to-nvd>
openssl,1.1.1k,CVE-123-LOW,LOW,<some-link-to-nvd>
systemd,1.3-debu49,CVE-123-MED,MODERATE,<some-link-to-nvd>
systemd,1.3-debu49,CVE-456-MED,MODERATE,<some-link-to-nvd>
systemd,1.3-debu49,CVE-789-MED,MODERATE,<some-link-to-nvd>
`,
		},
		"should print only headers with empty components in image scan": {
			components: nil,
			expectedOutput: `COMPONENT,VERSION,CVE,SEVERITY,LINK
`,
		},
	}

	factory, err := printer.NewObjectPrinterFactory("csv", supportedObjectPrinters...)
	s.Require().NoError(err)
	csvPrinter, err := factory.CreatePrinter()
	s.Require().NoError(err)

	s.runOutputTests(cases, csvPrinter, true)
}

func (s *imageScanTestSuite) TestScan_LegacyCSVOutput() {
	cases := map[string]outputFormatTest{
		"should print legacy CSV output if format is set": {
			components:     testComponents,
			expectedOutput: "CVE,CVSS Score,Severity Rating,Summary,Component,Version,Fixed By,Layer Instruction\r\nCVE-789-CRIT,9.5,Critical,This is a crit CVE 3,apt,1.0,1.3,layer1 1\r\nCVE-123-LOW,2,Low,This is a low CVE 1,apt,1.0,1.1,layer1 1\r\nCVE-789-LOW,2.5,Low,This is a low CVE 3,apt,1.0,1.3,layer1 1\r\nCVE-123-IMP,7,Important,This is a imp CVE 1,curl,7.0-rc1,1.1,layer2 2\r\nCVE-456-IMP,6.8,Important,This is a imp CVE 2,curl,7.0-rc1,1.2,layer2 2\r\nCVE-789-IMP,7,Important,This is a imp CVE 3,curl,7.0-rc1,1.3,layer2 2\r\nCVE-123-MED,4.5,Moderate,This is a mod CVE 1,systemd,1.3-debu49,1.1,layer2 2\r\nCVE-456-MED,4.9,Moderate,This is a mod CVE 2,systemd,1.3-debu49,1.2,layer2 2\r\nCVE-789-MED,5.2,Moderate,This is a mod CVE 3,systemd,1.3-debu49,1.3,layer2 2\r\nCVE-123-CRIT,8.5,Critical,This is a crit CVE 1,bash,4.2,1.1,layer3 3\r\nCVE-456-CRIT,9,Critical,This is a crit CVE 2,bash,4.2,1.2,layer3 3\r\nCVE-789-CRIT,9.5,Critical,This is a crit CVE 3,bash,4.2,1.3,layer3 3\r\nCVE-789-CRIT,9.5,Critical,This is a crit CVE 3,openssl,1.1.1k,1.3,layer3 3\r\nCVE-123-IMP,7,Important,This is a imp CVE 1,openssl,1.1.1k,1.1,layer3 3\r\nCVE-123-MED,4.5,Moderate,This is a mod CVE 1,openssl,1.1.1k,1.1,layer3 3\r\nCVE-456-MED,4.9,Moderate,This is a mod CVE 2,openssl,1.1.1k,1.2,layer3 3\r\nCVE-123-LOW,2,Low,This is a low CVE 1,openssl,1.1.1k,1.1,layer3 3\r\n",
		},
	}

	s.runLegacyOutputTests(cases, "csv")
}

func (s *imageScanTestSuite) TestScan_LegacyJSONOutput() {
	cases := map[string]outputFormatTest{
		"should print legacy JSON output if format is set": {
			components: testComponents,
			expectedOutput: `{
  "metadata": {
    "v1": {
      "layers": [
        {
          "instruction": "layer1",
          "value": "1"
        },
        {
          "instruction": "layer2",
          "value": "2"
        },
        {
          "instruction": "layer3",
          "value": "3"
        }
      ]
    }
  },
  "scan": {
    "components": [
      {
        "name": "apt",
        "version": "1.0",
        "vulns": [
          {
            "cve": "CVE-123-LOW",
            "cvss": 2,
            "summary": "This is a low CVE 1",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.1",
            "severity": "LOW_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-789-LOW",
            "cvss": 2.5,
            "summary": "This is a low CVE 3",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.3",
            "severity": "LOW_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-789-CRIT",
            "cvss": 9.5,
            "summary": "This is a crit CVE 3",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.3",
            "severity": "CRITICAL_VULNERABILITY_SEVERITY"
          }
        ],
        "layerIndex": 0,
        "fixedBy": "1.4"
      },
      {
        "name": "systemd",
        "version": "1.3-debu49",
        "vulns": [
          {
            "cve": "CVE-123-MED",
            "cvss": 4.5,
            "summary": "This is a mod CVE 1",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.1",
            "severity": "MODERATE_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-456-MED",
            "cvss": 4.9,
            "summary": "This is a mod CVE 2",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.2",
            "severity": "MODERATE_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-789-MED",
            "cvss": 5.2,
            "summary": "This is a mod CVE 3",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.3",
            "severity": "MODERATE_VULNERABILITY_SEVERITY"
          }
        ],
        "layerIndex": 1,
        "fixedBy": "1.3-debu102"
      },
      {
        "name": "curl",
        "version": "7.0-rc1",
        "vulns": [
          {
            "cve": "CVE-123-IMP",
            "cvss": 7,
            "summary": "This is a imp CVE 1",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.1",
            "severity": "IMPORTANT_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-456-IMP",
            "cvss": 6.8,
            "summary": "This is a imp CVE 2",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.2",
            "severity": "IMPORTANT_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-789-IMP",
            "cvss": 7,
            "summary": "This is a imp CVE 3",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.3",
            "severity": "IMPORTANT_VULNERABILITY_SEVERITY"
          }
        ],
        "layerIndex": 1,
        "fixedBy": "7.1-rc2"
      },
      {
        "name": "bash",
        "version": "4.2",
        "vulns": [
          {
            "cve": "CVE-123-CRIT",
            "cvss": 8.5,
            "summary": "This is a crit CVE 1",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.1",
            "severity": "CRITICAL_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-456-CRIT",
            "cvss": 9,
            "summary": "This is a crit CVE 2",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.2",
            "severity": "CRITICAL_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-789-CRIT",
            "cvss": 9.5,
            "summary": "This is a crit CVE 3",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.3",
            "severity": "CRITICAL_VULNERABILITY_SEVERITY"
          }
        ],
        "layerIndex": 2,
        "fixedBy": "4.3"
      },
      {
        "name": "openssl",
        "version": "1.1.1k",
        "vulns": [
          {
            "cve": "CVE-123-LOW",
            "cvss": 2,
            "summary": "This is a low CVE 1",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.1",
            "severity": "LOW_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-123-MED",
            "cvss": 4.5,
            "summary": "This is a mod CVE 1",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.1",
            "severity": "MODERATE_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-456-MED",
            "cvss": 4.9,
            "summary": "This is a mod CVE 2",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.2",
            "severity": "MODERATE_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-123-IMP",
            "cvss": 7,
            "summary": "This is a imp CVE 1",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.1",
            "severity": "IMPORTANT_VULNERABILITY_SEVERITY"
          },
          {
            "cve": "CVE-789-CRIT",
            "cvss": 9.5,
            "summary": "This is a crit CVE 3",
            "link": "\u003csome-link-to-nvd\u003e",
            "fixedBy": "1.3",
            "severity": "CRITICAL_VULNERABILITY_SEVERITY"
          }
        ],
        "layerIndex": 2
      }
    ]
  }
}
`,
		},
	}

	s.runLegacyOutputTests(cases, "json")
}

// helpers to run output formats tests either for legacy formats or printer.ObjectPrinter supported formats

func (s *imageScanTestSuite) runOutputTests(cases map[string]outputFormatTest, printer printer.ObjectPrinter,
	standardizedFormat bool) {
	for name, c := range cases {
		s.Run(name, func() {
			var out *bytes.Buffer
			conn, closeF := s.createGRPCMockImageService(c.components)
			defer closeF()

			imgScanCmd := s.defaultImageScanCommand
			imgScanCmd.printer = printer
			imgScanCmd.standardizedFormat = standardizedFormat
			imgScanCmd.env, out = s.newTestMockEnvironmentWithConn(conn)

			err := imgScanCmd.Scan()
			s.Require().NoError(err)
			s.Assert().Equal(c.expectedOutput, out.String())
		})
	}
}

func (s *imageScanTestSuite) runLegacyOutputTests(cases map[string]outputFormatTest, format string) {
	for name, c := range cases {
		s.Run(name, func() {
			var out *bytes.Buffer
			conn, closeF := s.createGRPCMockImageService(c.components)
			defer closeF()

			imgScanCmd := s.defaultImageScanCommand
			imgScanCmd.format = format
			imgScanCmd.env, out = s.newTestMockEnvironmentWithConn(conn)

			err := imgScanCmd.Scan()
			s.Require().NoError(err)
			s.Assert().Equal(c.expectedOutput, out.String())
		})
	}
}
