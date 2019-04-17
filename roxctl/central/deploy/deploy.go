package deploy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	cflog "github.com/cloudflare/cfssl/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/zip"
	"github.com/stackrox/rox/roxctl/central/deploy/renderer"
)

func init() {
	// The cfssl library prints logs at Info level when it processes a
	// Certificate Signing Request (CSR) or issues a new certificate.
	// These logs do not help the user understand anything, so here
	// we adjust the log level to exclude them.
	cflog.Level = cflog.LevelWarning
}

var (
	isInteractive              bool
	flagsHiddenWhenInteractive = []string{
		"monitoring-persistence-type",
	}
)

func generateJWTSigningKey(fileMap map[string][]byte) error {
	// Generate the private key that we will use to sign JWTs for API keys.
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return errors.Wrap(err, "couldn't generate private key")
	}
	fileMap["jwt-key.der"] = x509.MarshalPKCS1PrivateKey(privateKey)
	return nil
}

func generateMTLSFiles(fileMap map[string][]byte) (caCert, caKey []byte, err error) {
	// Add MTLS files
	req := csr.CertificateRequest{
		CN:         "StackRox Certificate Authority",
		KeyRequest: csr.NewBasicKeyRequest(),
	}
	caCert, _, caKey, err = initca.New(&req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not generate keypair")
	}
	fileMap["ca.pem"] = caCert
	fileMap["ca-key.pem"] = caKey

	cert, err := mtls.IssueNewCertFromCA(mtls.CentralSubject, caCert, caKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not issue central cert")
	}
	fileMap["cert.pem"] = cert.CertPEM
	fileMap["key.pem"] = cert.KeyPEM
	return caCert, caKey, nil
}

func generateMonitoringFiles(fileMap map[string][]byte, caCert, caKey []byte) error {
	monitoringCert, err := mtls.IssueNewCertFromCA(mtls.Subject{ServiceType: storage.ServiceType_MONITORING_DB_SERVICE, Identifier: "Monitoring DB"},
		caCert, caKey)
	if err != nil {
		return err
	}
	fileMap["monitoring-db-cert.pem"] = monitoringCert.CertPEM
	fileMap["monitoring-db-key.pem"] = monitoringCert.KeyPEM

	monitoringCert, err = mtls.IssueNewCertFromCA(mtls.Subject{ServiceType: storage.ServiceType_MONITORING_UI_SERVICE, Identifier: "Monitoring UI"},
		caCert, caKey)
	if err != nil {
		return err
	}

	fileMap["monitoring-ui-cert.pem"] = monitoringCert.CertPEM
	fileMap["monitoring-ui-key.pem"] = monitoringCert.KeyPEM

	monitoringCert, err = mtls.IssueNewCertFromCA(mtls.Subject{ServiceType: storage.ServiceType_MONITORING_CLIENT_SERVICE, Identifier: "Monitoring Client"},
		caCert, caKey)
	if err != nil {
		return err
	}

	fileMap["monitoring-client-cert.pem"] = monitoringCert.CertPEM
	fileMap["monitoring-client-key.pem"] = monitoringCert.KeyPEM

	return nil
}

func outputZip(config renderer.Config) error {
	fmt.Fprint(os.Stderr, "Generating deployment bundle... ")

	wrapper := zip.NewWrapper()

	d, ok := renderer.Deployers[config.ClusterType]
	if !ok {
		return fmt.Errorf("undefined cluster deployment generator: %s", config.ClusterType)
	}

	config.SecretsByteMap = make(map[string][]byte)
	if err := generateJWTSigningKey(config.SecretsByteMap); err != nil {
		return err
	}

	if len(config.LicenseData) > 0 {
		config.SecretsByteMap["central-license"] = config.LicenseData
	}

	config.Environment = make(map[string]string)
	for _, flag := range features.Flags {
		config.Environment[flag.EnvVar()] = strconv.FormatBool(flag.(features.Feature).Enabled())
	}

	htpasswd, err := renderer.GenerateHtpasswd(&config)
	if err != nil {
		return err
	}

	for _, setting := range env.Settings {
		if _, ok := os.LookupEnv(setting.EnvVar()); ok {
			config.Environment[setting.EnvVar()] = setting.Setting()
		}
	}

	config.SecretsByteMap["htpasswd"] = htpasswd
	wrapper.AddFiles(zip.NewFile("password", []byte(config.Password+"\n"), zip.Sensitive))

	cert, key, err := generateMTLSFiles(config.SecretsByteMap)
	if err != nil {
		return err
	}

	if config.K8sConfig != nil && config.K8sConfig.Monitoring.Type.OnPrem() {
		if err := generateMonitoringFiles(config.SecretsByteMap, cert, key); err != nil {
			return err
		}

		if config.K8sConfig.Monitoring.Password == "" {
			config.K8sConfig.Monitoring.Password = renderer.CreatePassword()
			config.K8sConfig.Monitoring.PasswordAuto = true
		}

		config.SecretsByteMap["monitoring-password"] = []byte(config.K8sConfig.Monitoring.Password)
		wrapper.AddFiles(zip.NewFile("monitoring/password", []byte(config.K8sConfig.Monitoring.Password+"\n"), zip.Sensitive))
	}

	config.SecretsBase64Map = make(map[string]string)
	for k, v := range config.SecretsByteMap {
		config.SecretsBase64Map[k] = base64.StdEncoding.EncodeToString(v)
	}

	files, err := d.Render(config)
	if err != nil {
		return errors.Wrap(err, "could not render files")
	}
	wrapper.AddFiles(files...)

	var outputPath string
	if docker.IsContainerized() {
		bytes, err := wrapper.Zip()
		if err != nil {
			return errors.Wrap(err, "error generating zip file")
		}
		_, err = os.Stdout.Write(bytes)
		if err != nil {
			return errors.Wrap(err, "couldn't write zip file")
		}
	} else {
		var err error
		outputPath, err = wrapper.Directory(config.OutputDir)
		if err != nil {
			return errors.Wrap(err, "error generating directory for Central output")
		}
	}

	fmt.Fprintln(os.Stderr, "Done!")
	fmt.Fprintln(os.Stderr)

	if outputPath != "" {
		fmt.Fprintf(os.Stderr, "Wrote central bundle to %q\n", outputPath)
		fmt.Fprintln(os.Stderr)
	}

	if err := config.WriteInstructions(os.Stderr); err != nil {
		return err
	}
	return nil
}

func interactive() *cobra.Command {
	return &cobra.Command{
		Use:   "interactive",
		Short: "Interactive runs the CLI in interactive mode with user prompts",
		RunE: func(_ *cobra.Command, args []string) error {
			c := Command()
			c.SilenceUsage = true
			return runInteractive(c)
		},
		SilenceUsage: true,
	}
}

// Command defines the deploy command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate creates the required YAML files to deploy StackRox Central.",
		Long:  "Generate creates the required YAML files to deploy StackRox Central.",
		Run: func(c *cobra.Command, _ []string) {
			if !isInteractive {
				_ = c.Help()
			}
		},
	}
	c.PersistentFlags().StringVar(&cfg.Password, "password", "", "administrator password (default: autogenerated)")
	c.PersistentFlags().Var(&licenseVar{
		data: &cfg.LicenseData,
	}, "license", "license data or filename (default: none, - to read stdin)")

	c.PersistentFlags().Var(&featureValue{&cfg.Features}, "flags", "Feature flags to enable")
	utils.Must(c.PersistentFlags().MarkHidden("flags"))
	c.AddCommand(interactive())

	c.AddCommand(k8s())
	c.AddCommand(openshift())
	return c
}

func markFlagAsHidden(cmd *cobra.Command, flagName string) {
	_ = cmd.PersistentFlags().MarkHidden(flagName)
	for _, c := range cmd.Commands() {
		markFlagAsHidden(c, flagName)
	}
}

func runInteractive(cmd *cobra.Command) error {
	for _, f := range flagsHiddenWhenInteractive {
		markFlagAsHidden(cmd, f)
	}
	isInteractive = true
	// Overwrite os.Args because cobra uses them
	os.Args = walkTree(cmd)
	return cmd.Execute()
}
