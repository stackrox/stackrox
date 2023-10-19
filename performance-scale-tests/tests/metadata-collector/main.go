package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	ocpMetadata "github.com/cloud-bulldozer/go-commons/ocp-metadata"
	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/tools/clientcmd"
)

type fieldType int

const (
	integerFieldType fieldType = iota
	stringFieldType
)

type MetadataField struct {
	Name        string
	Description string

	FieldType  fieldType
	IntDefault int
	StrDefault string

	AutomaticallyGathered bool
}

var (
	metadataFields []MetadataField

	// Please keep in alphabetic order of the name field.
	acsEnabledFeatures           = newMetadataStringField("acsEnabledFeatures", false, "comma-separated list of the ACS features enabled on the test cluster")
	acsVersion                   = newMetadataStringField("acsVersion", false, "ACS version used on the test cluster")
	clusterType                  = newMetadataStringField("clusterType", true, "test cluster type (OpenShift or Kubernetes)")
	configsPerDeploymentCount    = newMetadataIntField("configsPerDeploymentCount", false, "number of configs per deployment created by the test")
	cpuArchitecture              = newMetadataStringField("cpuArchitecture", false, "CPU architecture used on the test cluster nodes")
	deploymentsPerNamespaceCount = newMetadataIntField("deploymentsPerNamespaceCount", false, "number of deployments per namespace created by the test")
	infraNodesCount              = newMetadataIntField("infraNodesCount", true, "number of infra nodes on the test cluster")
	infraNodesKernelVersion      = newMetadataStringField("infraNodesKernelVersion", false, "kernel version used on the infra nodes on the test cluster")
	infraNodesType               = newMetadataStringField("infraNodesType", true, "type of the infra nodes used on the test cluster")
	k8sVersion                   = newMetadataStringField("k8sVersion", true, "Kubernetes API version used by the test cluster")
	masterNodesCount             = newMetadataIntField("masterNodesCount", true, "number of master nodes on the test cluster")
	masterNodesKernelVersion     = newMetadataStringField("masterNodesKernelVersion", false, "kernel version used on the master nodes on the test cluster")
	masterNodesType              = newMetadataStringField("masterNodesType", true, "type of the master nodes used on the test cluster")
	namespacesCount              = newMetadataIntField("namespacesCount", false, "number of namespaces created by the test")
	ocpMajorVersion              = newMetadataStringField("ocpMajorVersion", true, "OpenShift major version used by the test cluster")
	ocpVersion                   = newMetadataStringField("ocpVersion", true, "OpenShift version used by the test cluster")
	otherNodesCount              = newMetadataIntField("otherNodesCount", true, "number of nodes in the test cluster that are neither master nodes, nor worker nodes, nor infra nodes")
	platform                     = newMetadataStringField("platform", true, "platform on which the test cluster is running (OpenShift, GKE, ...)")
	podsPerDeploymentCount       = newMetadataIntField("podsPerDeploymentCount", false, "number of pods per deployment created by the test")
	sdnType                      = newMetadataStringField("sdnType", true, "type of networking (SDN type) used in the test cluster")
	testWorkloadType             = newMetadataStringField("testWorkloadType", false, "name of the KubeBurner template used for the test")
	testWorkloadVersion          = newMetadataStringField("testWorkloadVersion", true, "version of the KubeBurner template used for the test")
	totalNodes                   = newMetadataIntField("totalNodes", true, "total number of nodes in the test cluster")
	workerNodesCount             = newMetadataIntField("workerNodesCount", true, "number of worker nodes on the test cluster")
	workerNodesKernelVersion     = newMetadataStringField("workerNodesKernelVersion", false, "kernel version used on the worker nodes on the test cluster")
	workerNodesType              = newMetadataStringField("workerNodesType", true, "type of the worker nodes used on the test cluster")

	allMetadata = make(map[string]interface{}, len(metadataFields))
)

func newMetadataField(
	name string,
	automaticallyCollected bool,
	fieldType fieldType,
	description string,
) MetadataField {
	field := MetadataField{
		Name:                  name,
		Description:           description,
		FieldType:             fieldType,
		AutomaticallyGathered: automaticallyCollected,
	}
	metadataFields = append(metadataFields, field)
	return field
}

func newMetadataIntField(
	name string,
	automaticallyCollected bool,
	description string,
) MetadataField {
	return newMetadataField(name, automaticallyCollected, integerFieldType, description)
}

func newMetadataStringField(
	name string,
	automaticallyCollected bool,
	description string,
) MetadataField {
	return newMetadataField(name, automaticallyCollected, stringFieldType, description)
}

func (f *MetadataField) getFlagName() string {
	var b strings.Builder
	lastIsLower := false
	for _, c := range f.Name {
		if !unicode.IsLetter(c) && !unicode.IsNumber(c) {
			continue
		}
		if unicode.IsLetter(c) && unicode.IsUpper(c) {
			if lastIsLower {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(c))
			lastIsLower = false
		} else {
			b.WriteRune(c)
			lastIsLower = true
		}
	}
	return b.String()
}

func (f *MetadataField) getDescription() string {
	if !f.AutomaticallyGathered {
		return f.Description
	}
	return f.Description + " - This field is automatically collected by default"
}

func main() {
	outputPath := "run_metadata.yaml"
	var kubeConfigPath string

	kubeConfigPathDefault := os.Getenv("KUBECONFIG")
	if len(kubeConfigPathDefault) == 0 {
		kubeConfigPathDefault = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	cmd := &cobra.Command{
		Use:   "metadata-collector",
		Short: "Generates test run metadata YAML file",
		Long:  "Generates test run metadata YAML file for kube-burner tests",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errors.New("expected no arguments; please check usage")
			}

			collectDataErr := collectAutomaticData(kubeConfigPath)
			if collectDataErr != nil {
				fmt.Fprintf(os.Stderr, "error during the automated data collection: %v\n", collectDataErr)
			}

			for _, field := range metadataFields {
				if !c.Flags().Lookup(field.getFlagName()).Changed {
					continue
				}
				switch field.FieldType {
				case integerFieldType:
					val, _ := c.Flags().GetInt(field.getFlagName())
					allMetadata[field.Name] = val
				case stringFieldType:
					val, _ := c.Flags().GetString(field.getFlagName())
					allMetadata[field.Name] = val
				}
			}

			writeErr := WriteMetadata(outputPath, allMetadata)
			if writeErr != nil {
				return writeErr
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&outputPath, "output-file", "run_metadata.yaml", "location where the consolidated metadata should be written to")
	cmd.Flags().StringVar(&kubeConfigPath, "kubeconf-path", kubeConfigPathDefault, "location of the kubernetes config file used for cluster infromation retrieval")

	for _, field := range metadataFields {
		switch field.FieldType {
		case integerFieldType:
			cmd.Flags().Int(field.getFlagName(), field.IntDefault, field.getDescription())
		case stringFieldType:
			cmd.Flags().String(field.getFlagName(), field.StrDefault, field.getDescription())
		}
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func WriteMetadata(outputPath string, values map[string]any) error {
	bytes, encodeErr := yaml.Marshal(values)
	if encodeErr != nil {
		return encodeErr
	}
	writeErr := os.WriteFile(outputPath, bytes, 0600)
	if writeErr != nil {
		return writeErr
	}
	return nil
}

func collectAutomaticData(kubeConfigPath string) error {
	var err *multierror.Error
	gitRevision, gitLookupErr := getGitRevision()
	if gitLookupErr != nil {
		err = multierror.Append(err, gitLookupErr)
	} else {
		allMetadata[testWorkloadVersion.Name] = gitRevision
	}

	clusterInfoErr := collectClusterInformation(kubeConfigPath)
	if clusterInfoErr != nil {
		err = multierror.Append(err, clusterInfoErr)
	}

	return err.ErrorOrNil()
}

func getGitRevision() (string, error) {
	currentWorkingDir, getWorkingDirErr := os.Getwd()
	if getWorkingDirErr != nil {
		return "", getWorkingDirErr
	}
	var gitRepo *git.Repository
	var repoOpenErr error
	cwdLen := len(currentWorkingDir)
	for i := 0; i < cwdLen; i++ {
		index := cwdLen - i - 1
		lastChar := currentWorkingDir[index]
		if !os.IsPathSeparator(lastChar) {
			continue
		}
		gitRepo, repoOpenErr = git.PlainOpen(currentWorkingDir[0:index])
		if repoOpenErr == nil {
			break
		}
	}
	if repoOpenErr != nil {
		return "", repoOpenErr
	}
	head, headLookupErr := gitRepo.Head()
	if headLookupErr != nil {
		return "", headLookupErr
	}
	hash := head.Hash()
	hashString := hash.String()
	revision := hashString + "-dirty"
	workTree, workTreeErr := gitRepo.Worktree()
	if workTreeErr != nil {
		return revision, nil
	}
	workTreeStatus, workTreeStatusErr := workTree.Status()
	if workTreeStatusErr != nil {
		return revision, nil
	}
	if len(workTreeStatus) > 0 {
		return revision, nil
	}
	return hashString, nil
}

func collectClusterInformation(kubeConfigurationPath string) error {
	config, configErr := clientcmd.BuildConfigFromFlags("", kubeConfigurationPath)
	if configErr != nil {
		return configErr
	}
	metadataFetcher, instantiationErr := ocpMetadata.NewMetadata(config)
	if instantiationErr != nil {
		return instantiationErr
	}
	clusterMetadata, infoFetchErr := metadataFetcher.GetClusterMetadata()
	if infoFetchErr != nil {
		return infoFetchErr
	}

	allMetadata[clusterType.Name] = clusterMetadata.ClusterType
	allMetadata[infraNodesCount.Name] = clusterMetadata.InfraNodesCount
	allMetadata[infraNodesType.Name] = clusterMetadata.InfraNodesType
	allMetadata[k8sVersion.Name] = clusterMetadata.K8SVersion
	allMetadata[masterNodesCount.Name] = clusterMetadata.MasterNodesCount
	allMetadata[masterNodesType.Name] = clusterMetadata.MasterNodesType
	allMetadata[ocpMajorVersion.Name] = clusterMetadata.OCPMajorVersion
	allMetadata[ocpVersion.Name] = clusterMetadata.OCPVersion
	allMetadata[otherNodesCount.Name] = clusterMetadata.OtherNodesCount
	allMetadata[platform.Name] = clusterMetadata.Platform
	allMetadata[sdnType.Name] = clusterMetadata.SDNType
	allMetadata[totalNodes.Name] = clusterMetadata.TotalNodes
	allMetadata[workerNodesCount.Name] = clusterMetadata.WorkerNodesCount
	allMetadata[workerNodesType.Name] = clusterMetadata.WorkerNodesType

	return nil
}
