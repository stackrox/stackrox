package generate

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/common"
	env "github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stretchr/testify/suite"
)

type emulatedError int

const (
	testImageDefault = "opensource"

	testHostPathPath      = "/var/lib/hostpath-path"
	testNodeSelectorValue = "host"
	testNodeSelectorKey   = "kubernetes.io/hostname"

	testPvcName         = "pvcname"
	testPvcSize         = 33
	testPvcStorageClass = "storageclass"
)

const (
	noError emulatedError = iota
	downloadError
	renderError
)

type testCaseType struct {
	description       string
	addError          emulatedError
	args              []string
	cmdUse            string
	containsCommands  []string
	allCommands       []string
	containsFlags     []string
	errContains       string
	checkUsage        bool
	expectedImage     string
	expectedOutputDir string
	expectedPVC       *renderer.ExternalPersistenceInstance
	expectedHostPath  *renderer.HostPathPersistenceInstance
}

func (c *testCaseType) shouldCheckUsage() bool {
	return c.checkUsage || len(c.args) != 0 && c.args[len(c.args)-1] == "-h"
}

func (c *testCaseType) shouldVerifyConfig() bool {
	// Do not check config in case of error
	if c.addError != noError || c.errContains != "" {
		return false
	}
	// Do not check config in case of usage
	if c.shouldCheckUsage() {
		return false
	}

	// Only verify config when the command is complete.
	if !set.NewFrozenStringSet("none", "pvc", "hostpath").Contains(c.cmdUse) {
		return false
	}
	return true
}

func TestCentralDBGenerateCli(t *testing.T) {
	suite.Run(t, new(centralDBGenerateCliTestSuite))
}

type centralDBGenerateCliTestSuite struct {
	suite.Suite

	testOutputDir string
}

func (s *centralDBGenerateCliTestSuite) SetupTest() {
	testutils.SetMainVersion(s.T(), "3.74.0.0")
	s.testOutputDir = s.T().TempDir()
}

func (s *centralDBGenerateCliTestSuite) TeardownTest() {
	_ = os.RemoveAll(s.testOutputDir)
}

func (s *centralDBGenerateCliTestSuite) TestCentralDBGenerateCli() {
	testCases := []testCaseType{
		{
			description:      "generate usage",
			args:             []string{"-h"},
			cmdUse:           "generate",
			containsCommands: []string{"k8s", "openshift"},
			containsFlags:    []string{"--enable-pod-security-policies"},
		},
		{
			description: "generate wrong cluster type",
			args:        []string{"ks"},
			cmdUse:      "generate",
			errContains: "Error: unknown command \"ks\" for \"generate\"",
		},
	}
	for _, clusterType := range []string{"k8s", "openshift"} {
		testCases = append(testCases,
			testCaseType{
				description:   fmt.Sprintf("generate %s usage", clusterType),
				args:          []string{clusterType, "-h"},
				cmdUse:        clusterType,
				allCommands:   []string{"hostpath", "pvc"},
				containsFlags: []string{"--central-db-image", "--image-defaults", "--output-dir"},
			},
			testCaseType{
				description:   fmt.Sprintf("generate %s pvc usage", clusterType),
				args:          []string{clusterType, "pvc", "-h"},
				cmdUse:        "pvc",
				allCommands:   []string{},
				containsFlags: []string{"--name", "--size", "--storage-class"},
			},
			testCaseType{
				description:   fmt.Sprintf("generate %s hostpath usage", clusterType),
				args:          []string{clusterType, "hostpath", "-h"},
				cmdUse:        "hostpath",
				allCommands:   []string{},
				containsFlags: []string{"--hostpath", "--node-selector-key", "--node-selector-value"},
			},
			testCaseType{
				description: fmt.Sprintf("generate %s none usage", clusterType),
				args:        []string{clusterType, "none", "-h"},
				cmdUse:      "none",
				allCommands: []string{},
			},
			testCaseType{
				description: fmt.Sprintf("generate %s", clusterType),
				args:        []string{clusterType},
				cmdUse:      clusterType,
				allCommands: []string{},
				errContains: "Error: storage type must be specified",
			},
			testCaseType{
				description: fmt.Sprintf("generate %s unknown", clusterType),
				args:        []string{clusterType, "--enable-pod-security-policies=false", "unknown"},
				cmdUse:      clusterType,
				errContains: "Error: unexpected storage type \"unknown\"",
				checkUsage:  true,
			},
			testCaseType{
				description:       fmt.Sprintf("generate %s pvc", clusterType),
				args:              []string{clusterType, "pvc"},
				cmdUse:            "pvc",
				expectedImage:     "quay.io/rhacs-eng/central-db:3.74.0.0",
				expectedOutputDir: defaultCentralDBBundle,
				expectedPVC:       &renderer.ExternalPersistenceInstance{Name: "central-db", Size: 100},
			},
			testCaseType{
				description:       fmt.Sprintf("generate %s hostpath", clusterType),
				args:              []string{clusterType, "hostpath"},
				cmdUse:            "hostpath",
				expectedImage:     "quay.io/rhacs-eng/central-db:3.74.0.0",
				expectedOutputDir: defaultCentralDBBundle,
				expectedHostPath: &renderer.HostPathPersistenceInstance{
					HostPath: defaultHostPathPath,
				},
			},
			testCaseType{
				description:       fmt.Sprintf("generate %s none", clusterType),
				args:              []string{clusterType, "none"},
				cmdUse:            "none",
				checkUsage:        true,
				expectedImage:     "quay.io/rhacs-eng/central-db:3.74.0.0",
				expectedOutputDir: defaultCentralDBBundle,
			},
			testCaseType{
				description: fmt.Sprintf("generate %s wrong storage type", clusterType),
				args:        []string{clusterType, "unknown"},
				cmdUse:      clusterType,
				checkUsage:  true,
				errContains: "Error: unexpected storage type \"unknown\"",
			},
			testCaseType{
				description: fmt.Sprintf("generate %s pvc central download error", clusterType),
				addError:    downloadError,
				args:        []string{clusterType, "pvc"},
				cmdUse:      "pvc",
				errContains: "Error: download error",
			},
			testCaseType{
				description: fmt.Sprintf("generate %s hostpath render error", clusterType),
				addError:    renderError,
				args:        []string{clusterType, "hostpath"},
				cmdUse:      "hostpath",
				errContains: "Error: render error",
			},
			testCaseType{
				description:       fmt.Sprintf("generate %s one disabled pod security", clusterType),
				args:              []string{clusterType, "none", "--enable-pod-security-policies=false"},
				cmdUse:            "none",
				expectedImage:     "quay.io/rhacs-eng/central-db:3.74.0.0",
				expectedOutputDir: defaultCentralDBBundle,
			},
			testCaseType{
				description:       fmt.Sprintf("generate %s none output", clusterType),
				args:              []string{"k8s", "none", "--output-dir", s.testOutputDir},
				cmdUse:            "none",
				expectedImage:     "quay.io/rhacs-eng/central-db:3.74.0.0",
				expectedOutputDir: s.testOutputDir,
			},
			testCaseType{
				description:       fmt.Sprintf("generate %s pvc image", clusterType),
				args:              []string{"k8s", "pvc", "--central-db-image", "quay.io/rhacs-eng/central-db:3.72.0.0"},
				cmdUse:            "pvc",
				expectedImage:     "quay.io/rhacs-eng/central-db:3.72.0.0",
				expectedOutputDir: defaultCentralDBBundle,
				expectedPVC:       &renderer.ExternalPersistenceInstance{Name: "central-db", Size: 100},
			},
			testCaseType{
				description:       fmt.Sprintf("generate %s hostpath image default", clusterType),
				args:              []string{"k8s", "hostpath", "--image-defaults", testImageDefault},
				cmdUse:            "hostpath",
				expectedImage:     "quay.io/stackrox-io/central-db:3.74.0.0",
				expectedOutputDir: defaultCentralDBBundle,
				expectedHostPath:  &renderer.HostPathPersistenceInstance{HostPath: defaultHostPathPath},
			},
			testCaseType{
				description:       fmt.Sprintf("generate %s pvc config", clusterType),
				args:              []string{"k8s", "pvc", "--name", testPvcName, "--size", fmt.Sprint(testPvcSize), "--storage-class", testPvcStorageClass},
				cmdUse:            "pvc",
				expectedImage:     "quay.io/rhacs-eng/central-db:3.74.0.0",
				expectedOutputDir: defaultCentralDBBundle,
				expectedPVC: &renderer.ExternalPersistenceInstance{
					Name:         testPvcName,
					Size:         uint32(testPvcSize),
					StorageClass: testPvcStorageClass,
				},
			},
			testCaseType{
				description:       fmt.Sprintf("generate %s hostpath config", clusterType),
				args:              []string{"k8s", "hostpath", "--hostpath", testHostPathPath, "--node-selector-key", testNodeSelectorKey, "--node-selector-value", testNodeSelectorValue},
				cmdUse:            "hostpath",
				expectedImage:     "quay.io/rhacs-eng/central-db:3.74.0.0",
				expectedOutputDir: defaultCentralDBBundle,
				expectedHostPath: &renderer.HostPathPersistenceInstance{
					HostPath:          testHostPathPath,
					NodeSelectorKey:   testNodeSelectorKey,
					NodeSelectorValue: testNodeSelectorValue,
				},
			},
		)
	}

	for _, testCase := range testCases {
		c := testCase
		cfg = renderer.Config{}
		s.Run(c.description, func() {
			s.Require().NoError(os.RemoveAll(defaultCentralDBBundle), os.RemoveAll(s.testOutputDir))
			rootCmd := s.createRootCommand(c)
			cmd, output, errOut, err := executeCommand(rootCmd, c.args...)
			s.Assert().NotNil(cmd)
			s.Assert().Equal(c.cmdUse, cmd.Use)
			if c.shouldCheckUsage() {
				_, cmds, flags := s.parseUsage(output)
				for _, expectCmd := range c.containsCommands {
					s.Assert().Contains(cmds, expectCmd)
				}
				for _, flag := range c.containsFlags {
					s.Assert().Contains(flags, flag)
				}
				if c.allCommands != nil {
					for _, contain := range c.allCommands {
						s.Assert().Contains(cmds, contain)
					}
					s.Assert().Len(cmds, len(c.allCommands))
				}
			}
			if c.errContains != "" {
				s.Assert().Error(err)
				s.Assert().Contains(errOut, c.errContains)
			} else {
				s.Assert().Empty(errOut)
				s.Assert().NoError(err)
			}
		})
	}
}

func (s *centralDBGenerateCliTestSuite) createRootCommand(testCase testCaseType) *cobra.Command {
	genCmd := Command(env.CLIEnvironment())
	common.PatchPersistentPreRunHooks(genCmd)
	if testCase.addError == downloadError {
		genCmd.PersistentPreRunE = func(*cobra.Command, []string) error {
			return errors.New("download error")
		}
	} else {
		genCmd.PersistentPreRunE = func(*cobra.Command, []string) error {
			cfg.SecretsByteMap = map[string][]byte{
				"ca.pem":              []byte("ca.pem"),
				"central-db-cert.pem": []byte("central-db-cert.pem"),
				"central-db-key.pem":  []byte("central-db-key.pem"),
				"central-db-password": []byte("password"),
			}
			return nil
		}
	}

	s.setRunE(testCase, genCmd, "k8s", "pvc")
	s.setRunE(testCase, genCmd, "k8s", "hostpath")
	s.setRunE(testCase, genCmd, "k8s", "none")
	s.setRunE(testCase, genCmd, "openshift", "pvc")
	s.setRunE(testCase, genCmd, "openshift", "hostpath")
	s.setRunE(testCase, genCmd, "openshift", "none")

	return genCmd
}

func (s *centralDBGenerateCliTestSuite) setRunE(testCase testCaseType, command *cobra.Command, subCmds ...string) {
	for _, subCmd := range subCmds {
		command = s.lookUpCommand(command.Commands(), subCmd)
	}
	runE := command.RunE
	command.RunE = func(cmd *cobra.Command, args []string) error {
		// Run original command but tolerate the error.
		_ = runE(cmd, args)
		if testCase.addError == renderError {
			return errors.New("render error")
		}
		s.verifyConfig(testCase, &cfg)
		return nil
	}
}

func (s *centralDBGenerateCliTestSuite) lookUpCommand(cmds []*cobra.Command, target string) *cobra.Command {
	for _, cmd := range cmds {
		if cmd.Use == target {
			return cmd
		}
	}
	s.Require().Fail("Cannot find command %s", target)
	return nil
}

func (s *centralDBGenerateCliTestSuite) parseUsage(output string) (usage string, commands map[string][]string, flags map[string][]string) {
	lines := strings.Split(output, "\n")
	commands = make(map[string][]string)
	flags = make(map[string][]string)
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch line {
		case "Usage:":
			i++
			s.Assert().Less(i, len(lines))
			usage = lines[i]
		case "Available Commands:":
			i++
			s.Assert().Less(i, len(lines))
			for ; i < len(lines) && strings.TrimSpace(lines[i]) != ""; i++ {
				words := strings.Fields(lines[i])
				s.Assert().Greater(len(words), 1)
				commands[words[0]] = words[1:]
			}
		case "Flags:":
			i++
			s.Assert().Less(i, len(lines))
			for ; i < len(lines) && strings.TrimSpace(lines[i]) != ""; i++ {
				words := strings.Fields(lines[i])
				s.Assert().Greater(len(words), 1)
				flags[words[0]] = words[1:]
			}
		}
	}
	return
}

func (s *centralDBGenerateCliTestSuite) verifyConfig(c testCaseType, config *renderer.Config) {
	if !c.shouldVerifyConfig() {
		return
	}
	args := set.NewFrozenStringSet(c.args...)

	// Verify generate and k8s settings.
	s.Assert().Len(config.SecretsByteMap, 4)
	s.Assert().Equal(args.Contains("--enable-pod-security-policies=false"), !config.EnablePodSecurityPolicies)
	s.Assert().True(config.K8sConfig.EnableCentralDB)
	s.Assert().Equal(c.expectedOutputDir, config.OutputDir)
	if buildinfo.BuildFlavor == "development" {
		s.Assert().Equal(config.K8sConfig.CentralDBImage, c.expectedImage)
	}

	if args.Contains("--image-defaults") {
		s.Assert().Equal(config.K8sConfig.ImageFlavorName, testImageDefault)
	}

	// Verify settings for each storage type
	if c.expectedPVC != nil {
		s.Assert().NotNil(config.External)
		s.Assert().Equal(c.expectedPVC, config.External.DB)
	} else {
		s.Assert().Nil(config.External)
	}

	if c.expectedHostPath != nil {
		s.Assert().NotNil(config.HostPath)
		s.Assert().Equal(c.expectedHostPath, config.HostPath.DB)
	} else {
		s.Assert().Nil(config.HostPath)
	}
}

func executeCommand(root *cobra.Command, args ...string) (*cobra.Command, string, string, error) {
	outputBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outputBuf)
	root.SetErr(errBuf)
	root.SetArgs(args)

	c, err := root.ExecuteC()

	return c, outputBuf.String(), errBuf.String(), err
}
