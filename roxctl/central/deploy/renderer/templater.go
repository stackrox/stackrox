package renderer

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/docker/distribution/reference"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/defaultimages"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/zip"
)

var (
	log = logging.LoggerForModule()

	dockerAuthFile = func() *zip.File {
		str, err := image.AssetBox.FindString("docker-auth.sh")
		if err != nil {
			log.Panicf("docker auth file could not be found: %v", err)
		}
		return zip.NewFile("docker-auth.sh", []byte(str), zip.Executable)
	}()
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
	MainImage       string
	ScannerImage    string
	MonitoringImage string
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

// PersistenceType describes the type of persistence
type PersistenceType string

// Types of persistence
var (
	PersistenceNone     = newPersistentType("none")
	PersistenceHostpath = newPersistentType("hostpath")
	PersistencePVC      = newPersistentType("pvc")
)

// StringToPersistentTypes is a map from the persistenttype string value to its object
var StringToPersistentTypes = make(map[string]PersistenceType)

func newPersistentType(t string) PersistenceType {
	pt := PersistenceType(t)
	StringToPersistentTypes[t] = pt
	return pt
}

// String returns the string form of the enum
func (m PersistenceType) String() string {
	return string(m)
}

// MonitoringConfig encapsulates the monitoring configuration
type MonitoringConfig struct {
	Endpoint         string
	Image            string
	Type             MonitoringType
	LoadBalancerType v1.LoadBalancerType
	Password         string
	PasswordAuto     bool

	PersistenceType PersistenceType
	External        *ExternalPersistence
	HostPath        *HostPathPersistence
}

// K8sConfig contains k8s fields
type K8sConfig struct {
	CommonConfig
	ConfigType v1.DeploymentFormat

	// k8s fields
	Registry string

	ScannerRegistry string
	// If the scanner registry is different from the central registry get a separate secret
	ScannerSecretName string

	// These variables are not prompted for by Cobra, but are set based on
	// provided inputs for use in templating.
	MainImageTag    string
	ScannerImageTag string

	DeploymentFormat v1.DeploymentFormat
	LoadBalancerType v1.LoadBalancerType

	// Command is either oc or kubectl depending on the value of cluster type
	Command string

	Monitoring MonitoringConfig

	OfflineMode bool
}

// Config configures the deployer for the central service.
type Config struct {
	ClusterType storage.ClusterType
	OutputDir   string

	K8sConfig *K8sConfig

	External *ExternalPersistence
	HostPath *HostPathPersistence
	Features []features.FeatureFlag

	Password     string
	PasswordAuto bool

	LicenseData []byte

	DefaultTLSCertPEM []byte
	DefaultTLSKeyPEM  []byte

	SecretsByteMap   map[string][]byte
	SecretsBase64Map map[string]string

	Environment map[string]string
}

type deployer interface {
	Render(Config) ([]*zip.File, error)
	Instructions(Config) string
}

// Deployers contains all implementations for central deployment generators.
var Deployers = make(map[storage.ClusterType]deployer)

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

func generateMonitoringImage(mainImage string, monitoringImage string) (string, error) {
	if monitoringImage != "" {
		_, err := reference.ParseAnyReference(monitoringImage)
		if err != nil {
			return "", err
		}
		return monitoringImage, nil
	}
	img, err := utils.GenerateImageFromString(mainImage)
	if err != nil {
		return "", err
	}
	imgName := img.GetName()
	monitoringImageName := defaultimages.GenerateNamedImageFromMainImage(imgName, imgName.GetTag(), defaultimages.Monitoring)
	return monitoringImageName.FullName, nil
}

func wrapFiles(files []*zip.File, c *Config) ([]*zip.File, error) {
	instructions, err := generateReadme(c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("README", []byte(instructions), 0))
	return files, nil
}

// WriteInstructions writes the instructions for the configured cluster
// to the provided writer.
func (c Config) WriteInstructions(w io.Writer) error {
	instructions, err := generateReadme(&c)
	if err != nil {
		return err
	}
	fmt.Fprint(w, standardizeWhitespace(instructions))

	if c.PasswordAuto {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Use username '%s' and the following auto-generated password for administrator login (also stored in the 'password' file):\n", basic.DefaultUsername)
		fmt.Fprintf(w, " %s\n", c.Password)
	}
	if c.K8sConfig != nil && c.K8sConfig.Monitoring.PasswordAuto {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Use the following auto-generated password for accessing monitoring (also stored in the 'monitoring/password' file):")
		fmt.Fprintf(w, " %s\n", c.K8sConfig.Monitoring.Password)
	}
	return nil
}

func standardizeWhitespace(instructions string) string {
	return strings.TrimSpace(instructions) + "\n"
}
