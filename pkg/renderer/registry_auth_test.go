package renderer

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/zip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Markers of the registry credential setup in the generated setup.sh.
	registryAuthScriptMarker = "docker-auth.sh"
	registryAuthSecretMarker = "kubernetes.io/dockerconfigjson"
	// Marker of the image pull secret instructions in the generated README.
	registryAuthReadmeMarker = "image pull secret setup"
)

func publicFlavorForTest(t *testing.T) defaults.ImageFlavor {
	f := flavorUtils.MakeImageFlavorForTest(t)
	f.ImagePullSecrets.AnonymousPull = true
	return f
}

func TestUsesPublicImages(t *testing.T) {
	t.Setenv("TEST_VERSIONS", "true")
	public := publicFlavorForTest(t)
	private := flavorUtils.MakeImageFlavorForTest(t)

	defaultImages := func() CommonConfig {
		return CommonConfig{
			MainImage:        public.MainImage(),
			CentralDBImage:   public.CentralDBImage(),
			ScannerV4Image:   public.ScannerV4Image(),
			ScannerV4DBImage: public.ScannerV4DBImage(),
		}
	}
	with := func(modify func(*CommonConfig)) CommonConfig {
		c := defaultImages()
		modify(&c)
		return c
	}

	cases := map[string]struct {
		flavor   defaults.ImageFlavor
		images   CommonConfig
		expected bool
	}{
		"public flavor with default images": {
			flavor:   public,
			images:   defaultImages(),
			expected: true,
		},
		"private flavor with default images": {
			flavor:   private,
			images:   with(func(*CommonConfig) {}),
			expected: false,
		},
		"public flavor, main image from another registry": {
			flavor:   public,
			images:   with(func(c *CommonConfig) { c.MainImage = "quay.io/rhacs-eng/main:1.2.3" }),
			expected: false,
		},
		"public flavor, central-db image from another registry": {
			flavor:   public,
			images:   with(func(c *CommonConfig) { c.CentralDBImage = "registry.example.com/central-db:1.2.3" }),
			expected: false,
		},
		"public flavor, scanner-v4 image from another registry": {
			flavor:   public,
			images:   with(func(c *CommonConfig) { c.ScannerV4Image = "registry.example.com/scanner-v4:1.2.3" }),
			expected: false,
		},
		"public flavor, scanner-v4-db image from another registry": {
			flavor:   public,
			images:   with(func(c *CommonConfig) { c.ScannerV4DBImage = "registry.example.com/scanner-v4-db:1.2.3" }),
			expected: false,
		},
		"public flavor, registry name only shares a prefix": {
			flavor:   public,
			images:   with(func(c *CommonConfig) { c.MainImage = public.MainRegistry + "-mirror/main:1.2.3" }),
			expected: false,
		},
		"public flavor, override to another repository under the same registry may be private": {
			flavor:   public,
			images:   with(func(c *CommonConfig) { c.MainImage = public.MainRegistry + "/my-main:custom" }),
			expected: false,
		},
		"public flavor, only the tag overridden keeps the default repository": {
			flavor:   public,
			images:   with(func(c *CommonConfig) { c.MainImage = public.MainRegistry + "/" + public.MainImageName + ":custom" }),
			expected: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &Config{K8sConfig: &K8sConfig{CommonConfig: tc.images}}
			assert.Equal(t, tc.expected, usesPublicImages(c, tc.flavor))
		})
	}
}

func findFile(files []*zip.File, name string) *zip.File {
	for _, f := range files {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func TestSetupScriptRegistryAuth(t *testing.T) {
	t.Setenv("TEST_VERSIONS", "true")
	public := publicFlavorForTest(t)
	private := flavorUtils.MakeImageFlavorForTest(t)

	cases := map[string]struct {
		flavor defaults.ImageFlavor
		// Image overrides, applied on top of the flavor defaults.
		mainImage string
		format    v1.DeploymentFormat
		mode      mode
		// Path of the setup script in the generated bundle.
		scriptPath string
		// Whether the script is expected to configure registry credentials.
		expectAuth bool
	}{
		"kubectl, public flavor": {
			flavor: public, format: v1.DeploymentFormat_KUBECTL, mode: renderAll,
			scriptPath: "central/scripts/setup.sh", expectAuth: false,
		},
		"kubectl, public flavor, embedded helm bundle": {
			flavor: public, format: v1.DeploymentFormat_KUBECTL, mode: renderAll,
			scriptPath: "helm/scripts/setup.sh", expectAuth: false,
		},
		"kubectl, private flavor": {
			flavor: private, format: v1.DeploymentFormat_KUBECTL, mode: renderAll,
			scriptPath: "central/scripts/setup.sh", expectAuth: true,
		},
		"kubectl, private flavor, embedded helm bundle": {
			flavor: private, format: v1.DeploymentFormat_KUBECTL, mode: renderAll,
			scriptPath: "helm/scripts/setup.sh", expectAuth: true,
		},
		"kubectl, public flavor with main image from a private registry": {
			flavor: public, mainImage: "registry.example.com/stackrox/main:1.2.3",
			format: v1.DeploymentFormat_KUBECTL, mode: renderAll,
			scriptPath: "central/scripts/setup.sh", expectAuth: true,
		},
		"helm, public flavor": {
			flavor: public, format: v1.DeploymentFormat_HELM, mode: renderAll,
			scriptPath: "scripts/setup.sh", expectAuth: false,
		},
		"helm, private flavor": {
			flavor: private, format: v1.DeploymentFormat_HELM, mode: renderAll,
			scriptPath: "scripts/setup.sh", expectAuth: true,
		},
		"helm, public flavor with main image from a private registry": {
			flavor: public, mainImage: "registry.example.com/stackrox/main:1.2.3",
			format: v1.DeploymentFormat_HELM, mode: renderAll,
			scriptPath: "scripts/setup.sh", expectAuth: true,
		},
		"helm-values, public flavor": {
			flavor: public, format: v1.DeploymentFormat_HELM_VALUES, mode: renderAll,
			scriptPath: "scripts/setup.sh", expectAuth: false,
		},
		"helm-values, private flavor": {
			flavor: private, format: v1.DeploymentFormat_HELM_VALUES, mode: renderAll,
			scriptPath: "scripts/setup.sh", expectAuth: true,
		},
		"central-db only, public flavor": {
			flavor: public, format: v1.DeploymentFormat_KUBECTL, mode: centralDBOnly,
			scriptPath: "scripts/setup.sh", expectAuth: false,
		},
		"central-db only, private flavor": {
			flavor: private, format: v1.DeploymentFormat_KUBECTL, mode: centralDBOnly,
			scriptPath: "scripts/setup.sh", expectAuth: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			conf := Config{
				ClusterType: storage.ClusterType_KUBERNETES_CLUSTER,
				K8sConfig: &K8sConfig{
					CommonConfig:     CommonConfig{MainImage: tc.mainImage},
					DeploymentFormat: tc.format,
					EnableCentralDB:  true,
				},
				SecretsByteMap: map[string][]byte{},
			}
			if tc.mode == centralDBOnly {
				conf.HostPath = &HostPathPersistence{DB: &HostPathPersistenceInstance{HostPath: "/var/lib/stackrox-db"}}
			}

			files, err := render(conf, tc.mode, tc.flavor)
			require.NoError(t, err)

			script := findFile(files, tc.scriptPath)
			require.NotNil(t, script, "%s not found in the generated bundle", tc.scriptPath)
			content := string(script.Content)

			// The namespace is always prepared, whether or not credentials are needed.
			assert.Contains(t, content, `create namespace "$NAMESPACE"`)
			if tc.expectAuth {
				assert.Contains(t, content, registryAuthScriptMarker)
				assert.Contains(t, content, registryAuthSecretMarker)
			} else {
				assert.NotContains(t, content, registryAuthScriptMarker)
				assert.NotContains(t, content, registryAuthSecretMarker)
				assert.NotContains(t, content, "registry auth")
			}
			assert.NotContains(t, content, "{{", "unrendered template action")
			assertValidBashSyntax(t, content)
		})
	}
}

func assertValidBashSyntax(t *testing.T, script string) {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Log("bash not found, skipping the syntax check of the generated script")
		return
	}
	cmd := exec.Command(bash, "-n")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	assert.NoError(t, err, "generated script has invalid bash syntax: %s", out)
}

func TestHelmReadmeRegistryAuthInstructions(t *testing.T) {
	t.Setenv("TEST_VERSIONS", "true")
	public := publicFlavorForTest(t)
	private := flavorUtils.MakeImageFlavorForTest(t)

	cases := []struct {
		flavor     defaults.ImageFlavor
		mainImage  string
		format     v1.DeploymentFormat
		expectAuth bool
	}{
		{flavor: public, format: v1.DeploymentFormat_HELM, expectAuth: false},
		{flavor: public, format: v1.DeploymentFormat_HELM_VALUES, expectAuth: false},
		{flavor: private, format: v1.DeploymentFormat_HELM, expectAuth: true},
		{flavor: private, format: v1.DeploymentFormat_HELM_VALUES, expectAuth: true},
		{flavor: public, mainImage: "registry.example.com/main:1", format: v1.DeploymentFormat_HELM, expectAuth: true},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s/anonymousPull=%t/mainImage=%q", tc.format, tc.flavor.ImagePullSecrets.AnonymousPull, tc.mainImage), func(t *testing.T) {
			conf := Config{
				ClusterType: storage.ClusterType_KUBERNETES_CLUSTER,
				K8sConfig: &K8sConfig{
					CommonConfig:     CommonConfig{MainImage: tc.mainImage},
					DeploymentFormat: tc.format,
				},
				SecretsByteMap: map[string][]byte{},
			}
			files, err := render(conf, renderAll, tc.flavor)
			require.NoError(t, err)

			readme := findFile(files, "README")
			require.NotNil(t, readme)
			content := string(readme.Content)
			assert.Contains(t, content, "helm install")
			if tc.expectAuth {
				assert.Contains(t, content, registryAuthReadmeMarker)
				assert.Contains(t, content, "scripts/setup.sh")
			} else {
				assert.NotContains(t, content, registryAuthReadmeMarker)
				assert.NotContains(t, content, "Docker credentials")
				assert.NotContains(t, content, "imagePullSecrets")
			}
		})
	}
}
