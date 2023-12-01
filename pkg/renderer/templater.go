package renderer

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/zip"
)

var (
	log = logging.LoggerForModule()
)

// ExternalPersistenceInstance holds the data for a volume that is already created (e.g. docker volume, PV, etc)
type ExternalPersistenceInstance struct {
	Name         string
	StorageClass string
	Size         uint32
}

// ExternalPersistence is wrapper around the definitions for data held in a volume
type ExternalPersistence struct {
	Central *ExternalPersistenceInstance
	DB      *ExternalPersistenceInstance
}

// HostPathPersistenceInstance describes the parameters for a bind mount
type HostPathPersistenceInstance struct {
	HostPath          string
	NodeSelectorKey   string
	NodeSelectorValue string
}

// HostPathPersistence wraps the instances of bind mounts for Central and Central DB
type HostPathPersistence struct {
	Central *HostPathPersistenceInstance
	DB      *HostPathPersistenceInstance
}

// WithNodeSelector is a helper function for the templater that returns if node selectors are used
func (h *HostPathPersistenceInstance) WithNodeSelector() bool {
	if h == nil {
		return false
	}
	return h.NodeSelectorKey != ""
}

// CommonConfig contains the common config between orchestrators that cannot be placed at the top level
// Image is an example as it can be parameterized per orchestrator with different defaults so it cannot be placed
// at the top level
type CommonConfig struct {
	MainImage      string
	CentralDBImage string
	ScannerImage   string
	ScannerDBImage string
}

// TelemetryConfig contains config to set up the transimission of telemtry and diagnostic data.
type TelemetryConfig struct {
	Enabled         bool
	StorageEndpoint string
	StorageKey      string
}

// MonitoringConfig contains config to set up monitoring infrastructure.
type MonitoringConfig struct {
	OpenShiftMonitoring *bool
}

// DeclarativeConfigMounts contains mounts to config maps holding configuration to create resources in a declarative
// manner.
type DeclarativeConfigMounts struct {
	ConfigMaps []string
	Secrets    []string
}

// PersistenceType describes the type of persistence
type PersistenceType string

// StringToPersistentTypes is a map from the persistenttype string value to its object
var StringToPersistentTypes = make(map[string]PersistenceType)

// String returns the string form of the enum
func (m PersistenceType) String() string {
	return string(m)
}

// K8sConfig contains k8s fields
type K8sConfig struct {
	CommonConfig

	// Telemetry holds the configuration for telemetry.
	Telemetry TelemetryConfig

	// Monitoring holds the monitoring configuration.
	Monitoring MonitoringConfig

	// DeclarativeConfigMounts holds the mounts for specifying resources to be created in a declarative manner.
	DeclarativeConfigMounts DeclarativeConfigMounts

	// ImageFlavorName is the name of the flavor selected by the user with CLI parameters
	ImageFlavorName string

	// K8s Application name
	AppName string

	// k8s fields
	Registry string

	ScannerRegistry string
	// If the scanner registry is different from the central registry get a separate secret
	ScannerSecretName string

	// These variables are not prompted for by Cobra, but are set based on
	// provided inputs for use in templating.
	MainImageTag string

	DeploymentFormat v1.DeploymentFormat
	LoadBalancerType v1.LoadBalancerType

	// Command is either oc or kubectl depending on the value of cluster type
	Command string

	OfflineMode bool

	// IstioVersion is the version of Istio to render for (if any)
	IstioVersion string

	ImageOverrides map[string]interface{}

	EnableCentralDB bool
}

// Config configures the deployer for the central service.
type Config struct {
	ClusterType storage.ClusterType
	OutputDir   string

	K8sConfig *K8sConfig

	External *ExternalPersistence
	HostPath *HostPathPersistence

	Password     string
	PasswordAuto bool

	LicenseData []byte

	DefaultTLSCertPEM []byte
	DefaultTLSKeyPEM  []byte
	BackupBundle      string

	SecretsByteMap   map[string][]byte
	SecretsBase64Map map[string]string

	Environment map[string]string

	Version string

	ConfigFileOverrides map[string]string

	RenderOpts *helmUtil.Options // additional render options, if any (only legal in non-Helm mode).
	HelmImage  *image.Image

	EnablePodSecurityPolicies bool
}

// HasCentralHostPath returns if a Central is configured with host path
func (c Config) HasCentralHostPath() bool {
	return c.HostPath != nil && c.HostPath.Central != nil
}

// HasCentralDBHostPath returns if a Central DB is configured with host path
func (c Config) HasCentralDBHostPath() bool {
	return c.HostPath != nil && c.HostPath.DB != nil
}

// HasCentralExternal returns if a Central is configured with an external volume
func (c Config) HasCentralExternal() bool {
	return c.External != nil && c.External.Central != nil
}

// HasCentralDBExternal returns if a Central DB is configured with an external volume
func (c Config) HasCentralDBExternal() bool {
	return c.External != nil && c.External.DB != nil
}

func generateReadmeFile(c *Config, mode mode) (*zip.File, error) {
	instructions, err := generateReadme(c, mode)
	if err != nil {
		return nil, err
	}
	return zip.NewFile("README", []byte(instructions), 0), nil
}

// WriteInstructions writes the instructions for the configured cluster
// to the provided writer.
func (c Config) WriteInstructions(w io.Writer) error {
	instructions, err := generateReadme(&c, renderAll)
	if err != nil {
		return err
	}
	fmt.Fprint(w, standardizeWhitespace(instructions))

	if c.PasswordAuto {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Use username '%s' and the following auto-generated password for administrator login (also stored in the 'password' file):\n", basic.DefaultUsername)
		fmt.Fprintf(w, " %s\n", c.Password)
	}
	return nil
}

// GetConfigOverride retrieves the contents of the config override file `name`.
func (c Config) GetConfigOverride(name string) (string, error) {
	tgtFile := c.ConfigFileOverrides[name]
	if tgtFile == "" {
		return "", nil
	}

	f, err := os.Open(tgtFile)
	if err != nil {
		return "", errors.Wrapf(err, "opening override file %s for config file %s", tgtFile, name)
	}
	defer utils.IgnoreError(f.Close)

	contents, err := io.ReadAll(f)
	if err != nil {
		return "", errors.Wrapf(err, "reading contents of file %s", tgtFile)
	}

	return string(contents), nil
}

func standardizeWhitespace(instructions string) string {
	return strings.TrimSpace(instructions) + "\n"
}

// EnvironmentMap returns the values of Environment in the form of a map[string]interface{}
// that can be used with sprig template functions.
func (c Config) EnvironmentMap() map[string]interface{} {
	result := make(map[string]interface{}, len(c.Environment))
	for k, v := range c.Environment {
		result[k] = v
	}
	return result
}
