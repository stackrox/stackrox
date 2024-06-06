package scan

import (
	"bytes"
	"context"
	"net"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	v1.UnimplementedImageServiceServer

	components []*storage.EmbeddedImageScanComponent
}

func (m *mockImageServiceServer) ScanImage(_ context.Context, _ *v1.ScanImageRequest) (*storage.Image, error) {
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
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)

	closeF := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, closeF
}

func (s *imageScanTestSuite) newTestMockEnvironmentWithConn(conn *grpc.ClientConn) (environment.Environment, *bytes.Buffer, *bytes.Buffer) {
	return mocks.NewEnvWithConn(conn, s.T())
}

func (s *imageScanTestSuite) SetupTest() {
	s.defaultImageScanCommand = imageScanCommand{
		image:      "nginx:test",
		retryDelay: 3,
		retryCount: 3,
		timeout:    1 * time.Minute,
		severities: []string{
			lowCVESeverity.String(),
			moderateCVESeverity.String(),
			importantCVESeverity.String(),
			criticalCVESeverity.String(),
		},
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

	emptyOutputFormatPrinterFactory, err := printer.NewObjectPrinterFactory("json", jsonFactory)
	s.Require().NoError(err)
	emptyOutputFormatPrinterFactory.OutputFormat = ""

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Duration("timeout", 1*time.Minute, "")
	cmd.Flags().Duration("retry-timeout", 1*time.Minute, "")
	cmd.Flags().String("output", "", "")

	cases := map[string]struct {
		printerFactory     *printer.ObjectPrinterFactory
		standardizedFormat bool
		printer            printer.ObjectPrinter
		shouldFail         bool
		error              error
	}{
		"valid default values": {
			printerFactory:     validObjPrinterFactory,
			standardizedFormat: true,
			printer:            jsonPrinter,
		},
		"invalid printer factory should return an error": {
			printerFactory: invalidObjPrinterFactory,
			shouldFail:     true,
			error:          errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			imgScanCmd := s.defaultImageScanCommand
			imgScanCmd.env, _, _ = s.newTestMockEnvironmentWithConn(nil)

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
		image      string
		printer    printer.ObjectPrinter
		shouldFail bool
		error      error
	}{
		"no output format and image name should not result in an error": {
			image: s.defaultImageScanCommand.image,
		},
		"valid output printer and image name should not result in an error": {
			image:   s.defaultImageScanCommand.image,
			printer: jsonPrinter,
		},
		"invalid image name should result in an error": {
			image: "c:",
			error: errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			imgScanCmd := s.defaultImageScanCommand
			imgScanCmd.image = c.image
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
	components                   []*storage.EmbeddedImageScanComponent
	failOnFinding                bool
	error                        error
	expectedOutput               string
	expectedErrorOutput          string
	expectedErrorOutputColorized string
}

func (s *imageScanTestSuite) TestScan_TableOutput() {
	cases := map[string]outputFormatTest{
		"should render default output with merged cells and additional verbose output": {
			components:                   testComponents,
			expectedOutput:               "testComponents.txt",
			expectedErrorOutput:          "WARN:\tA total of 11 unique vulnerabilities were found in 5 components\n",
			expectedErrorOutputColorized: "\x1b[95mWARN:\tA total of 11 unique vulnerabilities were found in 5 components\n\x1b[0m",
		},
		"should return a vulnerability found error": {
			components:                   testComponents,
			failOnFinding:                true,
			error:                        errVulnerabilityFound,
			expectedOutput:               "testComponents.txt",
			expectedErrorOutput:          "WARN:\tA total of 11 unique vulnerabilities were found in 5 components\n",
			expectedErrorOutputColorized: "\x1b[95mWARN:\tA total of 11 unique vulnerabilities were found in 5 components\n\x1b[0m",
		},
		"should print only headers with empty components in image scan": {
			expectedOutput: "empty.txt",
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
			components:     testComponents,
			expectedOutput: "testComponents.json",
		},
		"should return a vulnerability found error": {
			components:     testComponents,
			failOnFinding:  true,
			error:          errVulnerabilityFound,
			expectedOutput: "testComponents.json",
		},
		"should print nothing with empty components in image scan": {
			components:     nil,
			expectedOutput: "empty.json",
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
			components:     testComponents,
			expectedOutput: "testComponents.csv",
		},
		"should return a vulnerability found error": {
			components:     testComponents,
			failOnFinding:  true,
			error:          errVulnerabilityFound,
			expectedOutput: "testComponents.csv",
		},
		"should print only headers with empty components in image scan": {
			components:     nil,
			expectedOutput: "empty.csv",
		},
	}

	factory, err := printer.NewObjectPrinterFactory("csv", supportedObjectPrinters...)
	s.Require().NoError(err)
	csvPrinter, err := factory.CreatePrinter()
	s.Require().NoError(err)

	s.runOutputTests(cases, csvPrinter, true)
}

// helpers to run printer.ObjectPrinter supported formats

func (s *imageScanTestSuite) runOutputTests(cases map[string]outputFormatTest, printer printer.ObjectPrinter,
	standardizedFormat bool,
) {
	const colorTestPrefix = "color_"
	for name, c := range cases {
		s.Run(name, func() {
			out, errOut, closeF, imgScanCmd := s.createImgScanCmd(c, printer, standardizedFormat)
			defer closeF()

			err := imgScanCmd.Scan()
			s.Assert().ErrorIs(err, c.error)
			expectedOutput, err := os.ReadFile(path.Join("testdata", c.expectedOutput))
			s.Require().NoError(err)
			s.Assert().Equal(string(expectedOutput), out.String())
			s.Assert().Equal(c.expectedErrorOutput, errOut.String())
		})
		s.Run(colorTestPrefix+name, func() {
			if runtime.GOOS == "windows" {
				s.T().Skip("Windows has different color sequences than Linux/Mac.")
			}
			color.NoColor = false
			defer func() { color.NoColor = true }()
			out, errOut, closeF, imgScanCmd := s.createImgScanCmd(c, printer, standardizedFormat)
			defer closeF()

			err := imgScanCmd.Scan()
			s.Assert().ErrorIs(err, c.error)
			expectedOutput, err := os.ReadFile(path.Join("testdata", colorTestPrefix+c.expectedOutput))
			s.Require().NoError(err)
			s.Assert().Equal(string(expectedOutput), out.String())
			s.Assert().Equal(c.expectedErrorOutputColorized, errOut.String())
		})
	}
}

func (s *imageScanTestSuite) createImgScanCmd(c outputFormatTest, printer printer.ObjectPrinter, standardizedFormat bool) (*bytes.Buffer, *bytes.Buffer, closeFunction, imageScanCommand) {
	var out, errOut *bytes.Buffer
	conn, closeF := s.createGRPCMockImageService(c.components)

	imgScanCmd := s.defaultImageScanCommand
	imgScanCmd.printer = printer
	imgScanCmd.standardizedFormat = standardizedFormat
	imgScanCmd.env, out, errOut = s.newTestMockEnvironmentWithConn(conn)
	imgScanCmd.failOnFinding = c.failOnFinding
	return out, errOut, closeF, imgScanCmd
}

func (s *imageScanTestSuite) TestScan_IncludeSnoozed() {
	s.Run("disabled by default", func() {
		envMock, _, _ := s.newTestMockEnvironmentWithConn(nil)
		cobraCommand := Command(envMock)
		s.Equal("false", cobraCommand.Flag("include-snoozed").Value.String())
	})
}
