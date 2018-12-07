package renderer

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"
	"text/template"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/zip"
)

var (
	log = logging.LoggerForModule()

	dockerAuthPath = "/data/assets/docker-auth.sh"
)

// ExternalPersistence holds the data for a volume that is already created (e.g. docker volume, PV, etc)
type ExternalPersistence struct {
	Name         string
	StorageClass string
}

// HostPathPersistence describes the parameters for a bind mount
type HostPathPersistence struct {
	HostPath          string
	NodeSelectorKey   string
	NodeSelectorValue string
}

// WithNodeSelector is a helper function for the templater that returns if node selectors are used
func (h *HostPathPersistence) WithNodeSelector() bool {
	if h == nil {
		return false
	}
	return h.NodeSelectorKey != ""
}

// CommonConfig contains the common config between orchestrators that cannot be placed at the top level
// Image is an example as it can be parameterized per orchestrator with different defaults so it cannot be placed
// at the top level
type CommonConfig struct {
	MainImage     string
	ClairifyImage string
}

// MonitoringType is the enum for the place monitoring is hosted
type MonitoringType int

// Types of monitoring
const (
	OnPrem = iota
	None
	StackRoxHosted
)

// String returns the string form of the enum
func (m MonitoringType) String() string {
	switch m {
	case OnPrem:
		return "on-prem"
	case None:
		return "none"
	case StackRoxHosted:
		return "stackrox-hosted"
	}
	return "unknown"
}

// OnPrem is true if the monitoring is hosted on prem
func (m MonitoringType) OnPrem() bool {
	return m == OnPrem
}

// StackRoxHosted is true if the monitoring is hosted by StackRox
func (m MonitoringType) StackRoxHosted() bool {
	return m == StackRoxHosted
}

// None returns true if there is no monitoring solution
func (m MonitoringType) None() bool {
	return m == None
}

// K8sConfig contains k8s fields
type K8sConfig struct {
	CommonConfig
	ConfigType v1.DeploymentFormat

	// k8s fields
	Namespace string
	Registry  string

	// These variables are not prompted for by Cobra, but are set based on
	// provided inputs for use in templating.
	MainImageTag     string
	ClairifyImageTag string

	MonitoringEndpoint         string
	MonitoringImage            string
	MonitoringType             MonitoringType
	MonitoringLoadBalancerType v1.LoadBalancerType
	MonitoringPassword         string
	MonitoringPasswordAuto     bool

	DeploymentFormat v1.DeploymentFormat
	LoadBalancerType v1.LoadBalancerType

	// Command is either oc or kubectl depending on the value of cluster type
	Command string
}

// SwarmConfig contains swarm fields
type SwarmConfig struct {
	CommonConfig

	// swarm fields
	NetworkMode string
	PublicPort  int
}

// Config configures the deployer for the central service.
type Config struct {
	ClusterType v1.ClusterType
	K8sConfig   *K8sConfig
	SwarmConfig *SwarmConfig

	External *ExternalPersistence
	HostPath *HostPathPersistence
	Features []features.FeatureFlag

	Password     string
	PasswordAuto bool

	SecretsByteMap   map[string][]byte
	SecretsBase64Map map[string]string

	Environment map[string]string
}

type deployer interface {
	Render(Config) ([]*zip.File, error)
	Instructions(Config) string
}

// Deployers contains all implementations for central deployment generators.
var Deployers = make(map[v1.ClusterType]deployer)

func executeRawTemplate(raw string, c *Config) ([]byte, error) {
	t, err := template.New("temp").Parse(raw)
	if err != nil {
		return nil, err
	}
	return executeTemplate(t, c)
}

func executeTemplate(temp *template.Template, c *Config) ([]byte, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := temp.Execute(buf, c)
	if err != nil {
		log.Errorf("Template execution failed: %s", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func generateMonitoringImage(mainImage string) string {
	img := types.Wrapper{Image: utils.GenerateImageFromString(mainImage)}
	remote := img.Namespace() + "/monitoring"
	// This handles the case where there is no namespace. e.g. stackrox.io/collector:latest
	if img.Repo() == "" {
		remote = "monitoring"
	}
	return fmt.Sprintf("%s/%s:%s", img.GetName().GetRegistry(), remote, img.GetName().GetTag())
}

func wrapFiles(files []*zip.File, c *Config, staticFilenames ...string) ([]*zip.File, error) {
	files = files[:len(files):len(files)]
	for _, staticFilename := range staticFilenames {
		var flags zip.FileFlags
		if path.Ext(staticFilename) == ".sh" {
			flags |= zip.Executable
		}
		f, err := zip.NewFromFile(staticFilename, path.Base(staticFilename), flags)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	readmeText, err := generateReadme(c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("README", []byte(standardizeWhitespace(readmeText)), 0))

	return files, nil
}

func renderFilenames(filenames []string, c *Config, staticFilenames ...string) ([]*zip.File, error) {
	var files []*zip.File
	for _, f := range filenames {
		t, err := templates.ReadFileAndTemplate(f)
		if err != nil {
			return nil, err
		}
		d, err := executeTemplate(t, c)
		if err != nil {
			return nil, err
		}

		var flags zip.FileFlags
		if strings.HasSuffix(f, ".sh") {
			flags |= zip.Executable
		}

		// Trim the first section off of the path because it defines the orchestrator
		path := f[strings.Index(f, "/")+1:]
		files = append(files, zip.NewFile(path, d, flags))
	}
	return wrapFiles(files, c, staticFilenames...)
}

// WriteInstructions writes the instructions for the configured cluster
// to the provided writer.
func (c Config) WriteInstructions(w io.Writer) {
	fmt.Fprint(w, standardizeWhitespace(Deployers[c.ClusterType].Instructions(c)))

	if c.PasswordAuto {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Use username '%s' and the following auto-generated password for administrator login (also stored in the 'password' file):\n", basic.DefaultUsername)
		fmt.Fprintf(w, " %s\n", c.Password)
	}
	if c.K8sConfig != nil && c.K8sConfig.MonitoringPasswordAuto {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Use the following auto-generated password for accessing monitoring (also stored in the 'monitoring/password' file):")
		fmt.Fprintf(w, " %s\n", c.K8sConfig.MonitoringPassword)
	}
}

func standardizeWhitespace(instructions string) string {
	return strings.TrimSpace(instructions) + "\n"
}
