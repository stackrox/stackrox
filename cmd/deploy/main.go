package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/cmd/deploy/central"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/version"
	zipPkg "github.com/stackrox/rox/pkg/zip"
)

var (
	clairifyTag   = "0.4"
	clairifyImage = "clairify:" + clairifyTag
	preventTag    = getVersion()
	preventImage  = "prevent:" + preventTag
)

func getVersion() string {
	v, err := version.GetVersion()
	if err != nil {
		panic(err)
	}
	return v
}

func outputZip(config central.Config) error {
	buf := new(bytes.Buffer)
	zipW := zip.NewWriter(buf)

	d, ok := central.Deployers[config.ClusterType]
	if !ok {
		return fmt.Errorf("undefined cluster deployment generator: %s", config.ClusterType)
	}

	files, err := d.Render(config)
	if err != nil {
		return fmt.Errorf("could not render files: %s", err)
	}
	for _, f := range files {
		if err := zipPkg.AddFile(zipW, f); err != nil {
			return fmt.Errorf("failed to write '%s': %s", f.Name, err)
		}
	}

	// Generate the private key that we will use to sign JWTs for API keys.
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("couldn't generate private key: %s", err)
	}
	err = zipPkg.AddFile(zipW, zipPkg.NewFile("jwt-key.der", string(x509.MarshalPKCS1PrivateKey(privateKey)), false))
	if err != nil {
		return fmt.Errorf("failed to write jwt key: %s", err)
	}

	// Add MTLS files
	req := csr.CertificateRequest{
		CN:         "StackRox Prevent Certificate Authority",
		KeyRequest: csr.NewBasicKeyRequest(),
	}
	cert, _, key, err := initca.New(&req)
	if err != nil {
		return fmt.Errorf("could not generate keypair: %s", err)
	}
	if err := zipPkg.AddFile(zipW, zipPkg.NewFile("ca.pem", string(cert), false)); err != nil {
		return fmt.Errorf("failed to write cert.pem: %s", err)
	}
	if err := zipPkg.AddFile(zipW, zipPkg.NewFile("ca-key.pem", string(key), false)); err != nil {
		return fmt.Errorf("failed to write key.pem: %s", err)
	}

	err = zipW.Close()
	if err != nil {
		return fmt.Errorf("couldn't close zip writer: %s", err)
	}

	_, err = os.Stdout.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("couldn't write zip file: %s", err)
	}
	return err
}

func root() *cobra.Command {
	c := &cobra.Command{
		Use:          "root",
		SilenceUsage: true,
	}
	c.PersistentFlags().Var(&featureValue{&cfg.Features}, "flags", "Feature flags to enable")
	c.PersistentFlags().MarkHidden("flags")
	c.AddCommand(interactive())
	c.AddCommand(cmd())
	return c
}

func interactive() *cobra.Command {
	return &cobra.Command{
		Use: "interactive",
		RunE: func(c *cobra.Command, args []string) error {
			c = cmd()
			c.SilenceUsage = true
			return runInteractive(c)
		},
		SilenceUsage: true,
	}
}

func cmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy generates deployment files for StackRox Prevent Central",
		Long: `Deploy generates deployment files for StackRox Prevent Central.
Output is a zip file printed to stdout.`,
		Run: func(*cobra.Command, []string) {
			printToStderr("Orchestrator is required\n")
		},
	}
	c.AddCommand(k8s())
	c.AddCommand(openshift())
	c.AddCommand(dockerBasedOrchestrator("swarm", "Docker Swarm", v1.ClusterType_SWARM_CLUSTER))
	return c
}

func runInteractive(cmd *cobra.Command) error {
	// Overwrite os.Args because cobra uses them
	os.Args = walkTree(cmd)
	return cmd.Execute()
}

func main() {
	root().Execute()
}
