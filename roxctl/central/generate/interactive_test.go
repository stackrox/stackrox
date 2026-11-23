package generate

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/common/environment"
	io2 "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registryPromptRegexp matches prompts that ask for anything related to registry credentials.
var registryPromptRegexp = regexp.MustCompile(`(?i)registry|docker|credential|pull secret|username`)

// maxPrompts is the upper bound of prompts in one interactive run. It prevents a test from hanging when the scripted
// answers do not satisfy a prompt that asks again until it gets a valid answer.
const maxPrompts = 100

// scriptedAnswers maps a case-insensitive substring of a prompt to the answers given to prompts containing it.
// The answers are consumed in order; once they are used up (or if no substring matches), the empty answer is given,
// which makes the interactive mode accept the default.
type scriptedAnswers map[string][]string

func (s scriptedAnswers) next(prompt string) string {
	prompt = strings.ToLower(prompt)
	for substring, answers := range s {
		if !strings.Contains(prompt, strings.ToLower(substring)) || len(answers) == 0 {
			continue
		}
		s[substring] = answers[1:]
		return answers[0]
	}
	return ""
}

// runInteractivePrompts runs the interactive prompting part of "roxctl central generate interactive" with the
// given scripted answers and returns the resulting command line arguments and all prompts that were shown.
// The prompts are read from stderr and each of them is answered on stdin, because the prompting code
// creates a new buffered reader for every prompt, so all answers cannot be written to stdin up front.
func runInteractivePrompts(t *testing.T, answers scriptedAnswers) (args []string, prompts []string) {
	t.Helper()

	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)
	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	origStdin, origStderr := os.Stdin, os.Stderr
	os.Stdin, os.Stderr = stdinR, stderrW
	defer func() {
		os.Stdin, os.Stderr = origStdin, origStderr
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stderrR.Read(buf)
			if n > 0 {
				// Every prompt is written with a single write and ends with ": ". Other output is not a prompt.
				if text := string(buf[:n]); strings.HasSuffix(text, ": ") {
					prompts = append(prompts, text)
					if len(prompts) > maxPrompts {
						panic(fmt.Sprintf("more than %d prompts, the scripted answers do not lead out of a prompt loop, last prompt: %q", maxPrompts, text))
					}
					_, _ = io.WriteString(stdinW, answers.next(text)+"\n")
				}
			}
			if err != nil {
				return
			}
		}
	}()

	cmd := Command(environment.NewTestCLIEnvironment(t, nil, printer.NoColorPrinter()))
	args = walkTree(cmd)

	require.NoError(t, stderrW.Close())
	wg.Wait()
	require.NoError(t, stdinW.Close())
	require.NoError(t, stdinR.Close())
	require.NoError(t, stderrR.Close())
	return args, prompts
}

// generateBundle runs "roxctl central generate" with the given arguments and returns the files of the bundle.
func generateBundle(t *testing.T, args []string) map[string]string {
	t.Helper()
	// Keep the bundle in memory.
	t.Setenv("ROX_ROXCTL_IN_MAIN_IMAGE", "true")

	// The command writes into a package-level config, which must not leak between runs.
	cfg = renderer.Config{}
	t.Cleanup(func() { cfg = renderer.Config{} })

	testIO, _, out, _ := io2.TestIO()
	cmd := Command(environment.NewTestCLIEnvironment(t, testIO, printer.NoColorPrinter()))
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	require.NoError(t, cmd.Execute())

	r, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
	require.NoError(t, err)
	files := make(map[string]string, len(r.File))
	for _, f := range r.File {
		rc, err := f.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		files[f.Name] = string(content)
	}
	return files
}

func TestInteractiveRegistryCredentials(t *testing.T) {
	testutils.SetExampleVersion(t)
	// Keep the bundle in memory, which also removes the output directory prompt.
	t.Setenv("ROX_ROXCTL_IN_MAIN_IMAGE", "true")

	// answersFor returns the answers for the prompts that decide what is generated. All other prompts get the default.
	answersFor := func(orchestrator, flavor, format, volume string) scriptedAnswers {
		return scriptedAnswers{
			"orchestrator":     {orchestrator},
			"container images": {flavor},
			"deployment tool":  {format},
			"volume type":      {volume},
		}
	}

	cases := map[string]struct {
		answers scriptedAnswers
		// The flavor that has to be passed on to the generate command.
		expectedFlavor string
		// The path of the setup script in the bundle.
		scriptPath string
		// Whether the setup script has to configure credentials for expectedAuthRegistry.
		expectAuth           bool
		expectedAuthRegistry string
		// Number of times the flavor prompt is expected to be shown.
		expectedFlavorPrompts int
	}{
		"opensource, k8s, kubectl, pvc": {
			answers:               answersFor("k8s", "opensource", "kubectl", "pvc"),
			expectedFlavor:        "opensource",
			scriptPath:            "central/scripts/setup.sh",
			expectedFlavorPrompts: 1,
		},
		"opensource, openshift, kubectl, hostpath": {
			answers:               answersFor("openshift", "opensource", "kubectl", "hostpath"),
			expectedFlavor:        "opensource",
			scriptPath:            "central/scripts/setup.sh",
			expectedFlavorPrompts: 1,
		},
		"opensource, k8s, helm": {
			answers:               answersFor("k8s", "opensource", "helm", "pvc"),
			expectedFlavor:        "opensource",
			scriptPath:            "scripts/setup.sh",
			expectedFlavorPrompts: 1,
		},
		"opensource, k8s, helm-values": {
			answers:               answersFor("k8s", "opensource", "helm-values", "pvc"),
			expectedFlavor:        "opensource",
			scriptPath:            "scripts/setup.sh",
			expectedFlavorPrompts: 1,
		},
		"opensource after an invalid flavor": {
			answers: func() scriptedAnswers {
				a := answersFor("k8s", "", "kubectl", "pvc")
				a["container images"] = []string{"no-such-flavor", "opensource"}
				return a
			}(),
			expectedFlavor:        "opensource",
			scriptPath:            "central/scripts/setup.sh",
			expectedFlavorPrompts: 2,
		},
		"opensource with the main image from a private registry": {
			answers: func() scriptedAnswers {
				a := answersFor("k8s", "opensource", "kubectl", "pvc")
				a["main image"] = []string{"registry.example.com/stackrox/main:1.2.3"}
				return a
			}(),
			expectedFlavor:        "opensource",
			scriptPath:            "central/scripts/setup.sh",
			expectAuth:            true,
			expectedAuthRegistry:  "registry.example.com",
			expectedFlavorPrompts: 1,
		},
		"rhacs, k8s, kubectl": {
			answers:               answersFor("k8s", "rhacs", "kubectl", "pvc"),
			expectedFlavor:        "rhacs",
			scriptPath:            "central/scripts/setup.sh",
			expectAuth:            true,
			expectedAuthRegistry:  "registry.redhat.io",
			expectedFlavorPrompts: 1,
		},
		"rhacs, k8s, helm": {
			answers:               answersFor("k8s", "rhacs", "helm", "pvc"),
			expectedFlavor:        "rhacs",
			scriptPath:            "scripts/setup.sh",
			expectAuth:            true,
			expectedAuthRegistry:  "registry.redhat.io",
			expectedFlavorPrompts: 1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg = renderer.Config{}
			args, prompts := runInteractivePrompts(t, tc.answers)
			assert.Contains(t, args, "--image-defaults="+tc.expectedFlavor)

			// The interactive mode never asks for registry credentials, for any flavor.
			require.NotEmpty(t, prompts)
			flavorPrompts := 0
			for _, prompt := range prompts {
				// The default values shown in the prompts legitimately contain registry names.
				question, _, _ := strings.Cut(prompt, "(default:")
				assert.NotRegexp(t, registryPromptRegexp, question)
				if strings.Contains(strings.ToLower(prompt), "container images") {
					flavorPrompts++
				}
			}
			assert.Equal(t, tc.expectedFlavorPrompts, flavorPrompts)

			files := generateBundle(t, args[1:])

			script, ok := files[tc.scriptPath]
			require.True(t, ok, "%s not found in the bundle, got %v", tc.scriptPath, fileNames(files))
			assert.Contains(t, script, `create namespace "$NAMESPACE"`)
			readme, ok := files["README"]
			require.True(t, ok, "README not found in the bundle, got %v", fileNames(files))
			if tc.expectAuth {
				assert.Contains(t, script, "docker-auth.sh")
				assert.Contains(t, script, tc.expectedAuthRegistry)
				assert.Contains(t, script, "kubernetes.io/dockerconfigjson")
			} else {
				assert.NotContains(t, script, "docker-auth.sh")
				assert.NotContains(t, script, "kubernetes.io/dockerconfigjson")
				assert.NotContains(t, readme, "Docker credentials")
			}
		})
	}
}

func fileNames(files map[string]string) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	return names
}
