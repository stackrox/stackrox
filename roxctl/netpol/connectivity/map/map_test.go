package connectivitymap

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stretchr/testify/suite"
)

func TestConnectivityMap(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(connectivityMapTestSuite))
}

type connectivityMapTestSuite struct {
	suite.Suite
}

func (d *connectivityMapTestSuite) TestAnalyzeNetpol() {
	tmpOutFileName := d.T().TempDir() + "/out"
	outFileTxt := tmpOutFileName + ".txt"
	outFileJSON := tmpOutFileName + ".json"
	outFileMD := tmpOutFileName + ".md"
	outFileCSV := tmpOutFileName + ".csv"
	outFileDOT := tmpOutFileName + ".dot"
	cases := map[string]struct {
		inputFolderPath       string
		treatWarningsAsErrors bool
		stopOnFirstErr        bool
		outFile               string
		outputToFile          bool
		focusWorkload         string
		outputFormat          string
		removeOutputPath      bool
		exposure              bool
		explain               bool

		expectedErrors   []string
		expectedWarnings []string
	}{
		"Not existing inputFolderPath should print error about path not existing but attempt analysis": {
			inputFolderPath:       "/tmp/xxx",
			stopOnFirstErr:        false,
			treatWarningsAsErrors: false,
			expectedErrors: []string{
				"the path \"/tmp/xxx\" does not exist",
				"no relevant Kubernetes workload resources found"},
			expectedWarnings: []string{"no relevant Kubernetes network policy resources found"},
		},
		"errors with no resources found": {
			inputFolderPath: "testdata/empty-yamls",
			expectedErrors:  []string{"no relevant Kubernetes workload resources found"},
			expectedWarnings: []string{
				"unable to decode \"testdata/empty-yamls/empty.yaml\": Object 'Kind' is missing in",
				"unable to decode \"testdata/empty-yamls/empty2.yaml\": Object 'Kind' is missing in",
				"no relevant Kubernetes network policy resources found",
			},
		},
		"treating warnings as errors": {
			inputFolderPath:       "testdata/minimal-with-invalid-doc",
			treatWarningsAsErrors: true,
			expectedErrors:        []string{},
			expectedWarnings:      []string{"unable to decode \"testdata/minimal-with-invalid-doc/resources.yaml\""},
		},
		"warnings not indicated without strict": {
			inputFolderPath:  "testdata/minimal-with-invalid-doc",
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal-with-invalid-doc/resources.yaml\": Object 'Kind' is missing in"},
		},
		"stopOnFistError": {
			inputFolderPath: "testdata/dirty",
			stopOnFirstErr:  true,
			expectedErrors:  []string{"no relevant Kubernetes workload resources found"},
			expectedWarnings: []string{
				"error parsing testdata/dirty/backend.yaml: error converting YAML to JSON",
				"error parsing testdata/dirty/frontend.yaml: error converting YAML to JSON",
				"no relevant Kubernetes network policy resources found",
			},
		},
		"should override existing file": {
			inputFolderPath:  "testdata/minimal",
			outFile:          outFileTxt,
			removeOutputPath: true,
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},
		},
		"output should be written to default txt output file": {
			inputFolderPath:  "testdata/minimal",
			outputToFile:     true,
			outputFormat:     defaultOutputFormat,
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},
		},
		"output should be focused to a workload": {
			inputFolderPath:  "testdata/minimal",
			focusWorkload:    "default/backend",
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},
		},
		"not supported output format": {
			inputFolderPath:  "testdata/minimal",
			outputFormat:     "docx",
			expectedErrors:   []string{"docx output format is not supported."},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},
		},
		"generate output in json format": {
			inputFolderPath:  "testdata/minimal",
			outputFormat:     "json",
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},

			outFile: outFileJSON,
		},
		"generate output in md format": {
			inputFolderPath:  "testdata/minimal",
			outputFormat:     "md",
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},

			outFile: outFileMD,
		},
		"generate output in csv format": {
			inputFolderPath:  "testdata/minimal",
			outputFormat:     "csv",
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},

			outFile: outFileCSV,
		},
		"generate output in dot format": {
			inputFolderPath:  "testdata/minimal",
			outputFormat:     "dot",
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},

			outFile: outFileDOT,
		},
		"openshift resources are recognized by the serializer with k8s resources": {
			inputFolderPath: "testdata/frontend-security",
			expectedErrors:  []string{},
			expectedWarnings: []string{
				"Route resource frontend/asset-cache specified workload frontend/asset-cache[Deployment] as a backend, but network policies are blocking ingress connections from an arbitrary in-cluster source to this workload. Connectivity map will not include a possibly allowed connection between the ingress controller and this workload.",
				"Route resource frontend/webapp specified workload frontend/webapp[Deployment] as a backend, but network policies are blocking ingress connections from an arbitrary in-cluster source to this workload. Connectivity map will not include a possibly allowed connection between the ingress controller and this workload.",
				"Connectivity analysis found no allowed connectivity between pairs from the configured workloads or external IP-blocks",
			},
		},
		"output should be written to default json output file": {
			inputFolderPath:  "testdata/minimal",
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},

			outputToFile: true,
			outputFormat: "json",
		},
		"generate connections list with ingress controller": {
			inputFolderPath:  "testdata/acs-security-demos",
			expectedErrors:   []string{},
			expectedWarnings: []string{},
			outputToFile:     true,
		},
		"generate connections list with exposure analysis": {
			inputFolderPath:  "testdata/acs-security-demos",
			expectedErrors:   []string{},
			expectedWarnings: []string{},
			outputToFile:     true,
			exposure:         true,
		},
		"generate explainability report for connections": {
			inputFolderPath:  "testdata/minimal",
			expectedErrors:   []string{},
			expectedWarnings: []string{"unable to decode \"testdata/minimal/output.json\""},
			outputToFile:     true,
			explain:          true,
		},
	}

	for name, tt := range cases {
		d.Run(name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			analyzeNetpolCmd := Cmd{
				stopOnFirstError:      tt.stopOnFirstErr,
				treatWarningsAsErrors: tt.treatWarningsAsErrors,
				inputFolderPath:       "", // set through construct
				outputFilePath:        tt.outFile,
				removeOutputPath:      tt.removeOutputPath,
				outputToFile:          tt.outputToFile,
				focusWorkload:         tt.focusWorkload,
				outputFormat:          tt.outputFormat,
				exposure:              tt.exposure,
				explain:               tt.explain,
				env:                   env,
			}

			analyzer, err := analyzeNetpolCmd.construct([]string{tt.inputFolderPath})
			d.Assert().NoError(err)
			d.Assert().NoError(analyzeNetpolCmd.validate())
			warns, errs := analyzeNetpolCmd.analyze(analyzer)
			npg.AssertErrorsContain(d.T(), tt.expectedErrors, errs, "errors")
			npg.AssertErrorsContain(d.T(), tt.expectedWarnings, warns, "warnings")

			if tt.outFile != "" && len(tt.expectedErrors) == 0 {
				_, err := os.Stat(tt.outFile)
				d.Assert().NoError(err) // out file should exist
			}

			if tt.outputToFile && tt.outFile == "" && len(tt.expectedErrors) == 0 && len(tt.expectedWarnings) == 0 {
				defaultFile := analyzeNetpolCmd.getDefaultFileName()
				formatSuffix := ""
				if tt.outputFormat != "" {
					d.Assert().Contains(defaultFile, tt.outputFormat)
					formatSuffix = tt.outputFormat
				} else {
					formatSuffix = defaultOutputFormat
				}
				output, err := os.ReadFile(defaultFile)
				d.Assert().NoError(err)
				expectedOutputFileName := "output." + formatSuffix
				if tt.exposure {
					expectedOutputFileName = "exposure_output." + formatSuffix
				}
				if tt.explain {
					d.Equal(formatSuffix, defaultOutputFormat)
					expectedOutputFileName = "explain_output.txt"
				}
				expectedOutput, err := os.ReadFile(path.Join(tt.inputFolderPath, expectedOutputFileName))
				d.Assert().NoError(err)
				d.Equal(strings.TrimRight(string(expectedOutput), "\n\r"), strings.TrimRight(string(output), "\n\r"))

				d.Assert().NoError(os.Remove(defaultFile))
			}
		})
	}

}
