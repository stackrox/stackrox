package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/central"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/spf13/cobra"
)

var (
	logger = logging.New("main")
)

func clusterType(ct string) v1.ClusterType {
	switch ct {
	case "kubernetes", "k8s":
		return v1.ClusterType_KUBERNETES_CLUSTER
	case "openshift":
		return v1.ClusterType_OPENSHIFT_CLUSTER
	case "swarm":
		return v1.ClusterType_SWARM_CLUSTER
	case "ee", "dockeree":
		return v1.ClusterType_DOCKER_EE_CLUSTER
	default:
		return v1.ClusterType_GENERIC_CLUSTER
	}
}

// ServeHTTP serves a ZIP file for the cluster upon request.
func outputZip(config central.Config, clusterType v1.ClusterType) error {
	buf := new(bytes.Buffer)
	zipW := zip.NewWriter(buf)

	d, ok := central.Deployers[clusterType]
	if !ok {
		return fmt.Errorf("Undefined cluster deployment generator: %s", clusterType)
	}
	dep, err := d.Deployment(config)
	if err != nil {
		return fmt.Errorf("Could not generate deployment: %s", err)
	}
	if err := addFile(zipW, "deploy.yaml", dep); err != nil {
		return fmt.Errorf("Failed to write deploy.yaml: %s", err)
	}
	cmd, err := d.Command(config)
	if err != nil {
		logger.Fatalf("Could not generate command: %s", err)
	}
	if err := addExecutableFile(zipW, "deploy.sh", cmd); err != nil {
		return fmt.Errorf("Failed to write deploy.sh: %s", err)
	}

	// Add MTLS files
	req := csr.CertificateRequest{
		CN:         "StackRox Mitigate Certificate Authority",
		KeyRequest: csr.NewBasicKeyRequest(),
	}
	cert, _, key, err := initca.New(&req)
	if err != nil {
		return fmt.Errorf("Could not generate keypair: %s", err)
	}
	if err := addFile(zipW, "ca.pem", string(cert)); err != nil {
		return fmt.Errorf("Failed to write cert.pem: %s", err)
	}
	if err := addFile(zipW, "ca-key.pem", string(key)); err != nil {
		return fmt.Errorf("Failed to write key.pem: %s", err)
	}

	err = zipW.Close()
	if err != nil {
		return fmt.Errorf("Couldn't close zip writer: %s", err)
	}

	_, err = os.Stdout.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("Couldn't write zip file: %s", err)
	}
	return err
}

func addFile(zipW *zip.Writer, name, contents string) error {
	f, err := zipW.Create(name)
	if err != nil {
		return fmt.Errorf("file creation: %s", err)
	}
	_, err = f.Write([]byte(contents))
	if err != nil {
		return fmt.Errorf("file writing: %s", err)
	}
	return nil
}

func addExecutableFile(zipW *zip.Writer, name, contents string) error {
	hdr := &zip.FileHeader{
		Name: name,
	}
	hdr.SetMode(os.ModePerm & 0755)
	f, err := zipW.CreateHeader(hdr)
	if err != nil {
		return fmt.Errorf("file creation: %s", err)
	}
	_, err = f.Write([]byte(contents))
	if err != nil {
		return fmt.Errorf("file writing: %s", err)
	}
	return nil
}

func cmd() *cobra.Command {
	var cfg central.Config
	var clusterTypeInput string
	c := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy generates deployment files for StackRox Mitigate Central",
		Long: `Deploy generates deployment files for StackRox Mitigate Central.
Output is a zip file printed to stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cluster := clusterType(clusterTypeInput)
			if cluster == v1.ClusterType_GENERIC_CLUSTER {
				return fmt.Errorf("Unknown cluster type '%s'", clusterTypeInput)
			}
			return outputZip(cfg, cluster)
		},
	}
	c.Flags().StringVarP(&clusterTypeInput, "type", "t", "", "cluster type (kubernetes, k8s, openshift, swarm, ee, dockeree)")
	c.Flags().StringVarP(&cfg.Image, "image", "i", "stackrox.io/mitigate", "image to use") // TODO(cg): -X flag should provide version tag
	c.Flags().StringVarP(&cfg.Namespace, "namespace", "n", "stackrox", "namespace [Kubernetes/OpenShift]")
	c.Flags().IntVarP(&cfg.PublicPort, "port", "p", 443, "public port to expose [Docker Swarm/Docker EE]")
	return c
}

func main() {
	if err := cmd().Execute(); err != nil {
		logger.Errorf("unable to execute: %s", err)
	}
}
