package generate

import (
	"encoding/pem"
	"os"
	"path"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/backup"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/roxctl"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/zip"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	io2 "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/mode"
	"github.com/stackrox/rox/roxctl/common/util"
	"k8s.io/utils/pointer"
)

func generateJWTSigningKey(fileMap map[string][]byte) error {
	// Generate the private key that we will use to sign JWTs for API keys.
	privateKey, err := certgen.GenerateJWTSigningKey()
	if err != nil {
		return errors.Wrap(err, "couldn't generate private key")
	}
	certgen.AddJWTSigningKeyToFileMap(fileMap, privateKey)
	return nil
}

func restoreJWTSigningKey(fileMap map[string][]byte, backupBundle string) error {
	z, err := zip.NewReader(backupBundle)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(z.Close)

	switch {
	case z.ContainsFile(path.Join(backup.KeysBaseFolder, backup.JwtKeyInDer)):
		jwtKey, _ := z.ReadFrom(path.Join(backup.KeysBaseFolder, backup.JwtKeyInDer))
		fileMap[certgen.JWTKeyPEMFileName] = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: jwtKey,
		})
	case z.ContainsFile(path.Join(backup.KeysBaseFolder, backup.JwtKeyInPem)):
		jwtKeyPem, err := z.ReadFrom(path.Join(backup.KeysBaseFolder, backup.JwtKeyInPem))
		if err != nil {
			return err
		}
		fileMap[certgen.JWTKeyPEMFileName] = jwtKeyPem
		decode, _ := pem.Decode(jwtKeyPem)
		if decode == nil {
			return errors.Errorf("Unable to decode key in %s:\n%s", backup.JwtKeyInPem, string(jwtKeyPem))
		}

	default:
		return errors.New("cannot find jwt key in backup bundle.")
	}
	return nil
}

func restoreCA(backupBundle string) (mtls.CA, error) {
	z, err := zip.NewReader(backupBundle)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(z.Close)

	caCert, err := z.ReadFrom(path.Join(backup.KeysBaseFolder, backup.CaCertPem))
	if err != nil {
		return nil, err
	}

	caKey, err := z.ReadFrom(path.Join(backup.KeysBaseFolder, backup.CaKeyPem))
	if err != nil {
		return nil, err
	}

	return mtls.LoadCAForSigning(caCert, caKey)
}

func restoreCentralDBPassword(fileMap map[string][]byte, backupBundle string) error {
	z, err := zip.NewReader(backupBundle)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(z.Close)

	passPath := path.Join(backup.DatabaseBaseFolder, backup.DatabasePassword)

	// If an older backup, file may not be included
	if !z.ContainsFile(passPath) {
		return nil
	}

	centralDBPass, err := z.ReadFrom(passPath)
	if err != nil {
		return err
	}

	fileMap["central-db-password"] = centralDBPass

	return nil
}

func populateMTLSFiles(fileMap map[string][]byte, backupBundle string) error {
	var ca mtls.CA
	var err error
	switch backupBundle {
	case "":
		if ca, err = certgen.GenerateCA(); err != nil {
			return err
		}
	default:
		if ca, err = restoreCA(backupBundle); err != nil {
			return err
		}

		if err = restoreCentralDBPassword(fileMap, backupBundle); err != nil {
			return err
		}
	}
	certgen.AddCAToFileMap(fileMap, ca)

	if err := certgen.IssueCentralCert(fileMap, ca); err != nil {
		return err
	}
	if err := certgen.IssueOtherServiceCerts(fileMap, ca, []mtls.Subject{mtls.CentralDBSubject}); err != nil {
		return err
	}
	fileMap["central-db-password"] = []byte(renderer.CreatePassword())

	if err := certgen.IssueScannerCerts(fileMap, ca); err != nil {
		return err
	}

	fileMap["scanner-db-password"] = []byte(renderer.CreatePassword())

	return nil
}

func updateConfig(config *renderer.Config) error {

	if config.ClusterType == storage.ClusterType_GENERIC_CLUSTER {
		return errox.InvalidArgs.Newf("invalid cluster type: %s", config.ClusterType)
	}

	config.SecretsByteMap = make(map[string][]byte)
	if config.BackupBundle == "" {
		if err := generateJWTSigningKey(config.SecretsByteMap); err != nil {
			return err
		}
	} else if err := restoreJWTSigningKey(config.SecretsByteMap, config.BackupBundle); err != nil {
		return err
	}

	if len(config.LicenseData) > 0 {
		config.SecretsByteMap["central-license"] = config.LicenseData
	}

	if len(config.DefaultTLSCertPEM) > 0 {
		config.SecretsByteMap["default-tls.crt"] = config.DefaultTLSCertPEM
		config.SecretsByteMap["default-tls.key"] = config.DefaultTLSKeyPEM
	}

	config.Environment = make(map[string]string)
	// Feature flags can only be overridden on release builds.
	if !buildinfo.ReleaseBuild {
		for _, flag := range features.Flags {
			if value := os.Getenv(flag.EnvVar()); value != "" {
				config.Environment[flag.EnvVar()] = strconv.FormatBool(flag.Enabled())
			}
		}
		if flags.IsDebug() {
			config.HelmImage = flags.GetDebugHelmImage()
		}
	}

	htpasswd, err := renderer.GenerateHtpasswd(config)
	if err != nil {
		return err
	}

	for _, setting := range env.Settings {
		if _, ok := os.LookupEnv(setting.EnvVar()); ok {
			config.Environment[setting.EnvVar()] = setting.Setting()
		}
	}
	if config.K8sConfig != nil {
		config.Environment[env.OfflineModeEnv.EnvVar()] = strconv.FormatBool(config.K8sConfig.OfflineMode)

		if config.K8sConfig.Telemetry.Enabled &&
			(version.IsReleaseVersion() || env.TelemetryStorageKey.Setting() != "") {
			config.K8sConfig.Telemetry.StorageKey = env.TelemetryStorageKey.Setting()
			config.K8sConfig.Telemetry.StorageEndpoint = env.TelemetryEndpoint.Setting()
		} else {
			config.K8sConfig.Telemetry.StorageKey = phonehome.DisabledKey
			config.K8sConfig.Telemetry.Enabled = false
		}

		if config.K8sConfig.Monitoring.OpenShiftMonitoring == nil {
			isOpenShift4 := config.ClusterType == storage.ClusterType_OPENSHIFT4_CLUSTER
			config.K8sConfig.Monitoring.OpenShiftMonitoring = pointer.Bool(isOpenShift4)
		} else if *config.K8sConfig.Monitoring.OpenShiftMonitoring &&
			config.ClusterType != storage.ClusterType_OPENSHIFT4_CLUSTER {
			return errox.InvalidArgs.Newf("OpenShift monitoring integration requires OpenShift 4, got %q", config.ClusterType.String())
		}
	}

	config.SecretsByteMap["htpasswd"] = htpasswd
	return nil
}

func createBundle(config *renderer.Config) (*zip.Wrapper, error) {

	wrapper := zip.NewWrapper()

	wrapper.AddFiles(zip.NewFile("password", []byte(config.Password+"\n"), zip.Sensitive))

	if err := populateMTLSFiles(config.SecretsByteMap, config.BackupBundle); err != nil {
		return nil, err
	}

	flavor, err := defaults.GetImageFlavorByName(config.K8sConfig.ImageFlavorName, buildinfo.ReleaseBuild)
	if err != nil {
		return nil, common.ErrInvalidCommandOption.CausedByf("'--%s': %v", flags.ImageDefaultsFlagName, err)
	}

	files, err := renderer.Render(*config, flavor)
	if err != nil {
		return nil, errors.Wrap(err, "could not render files")
	}
	wrapper.AddFiles(files...)

	return wrapper, nil
}

// OutputZip renders a deployment bundle. The deployment bundle can either be
// written directly into a directory, or as a zipfile to STDOUT.
func OutputZip(logger logger.Logger, io io2.IO, config renderer.Config) error {
	logger.InfofLn("Generating deployment bundle...")

	common.LogInfoPsp(logger, config.EnablePodSecurityPolicies)

	err := updateConfig(&config)
	if err != nil {
		return err
	}
	if config.K8sConfig.Telemetry.Enabled {
		logger.InfofLn("StackRox Kubernetes Security Platform collects " +
			"and transmits anonymous usage and system configuration " +
			"information. If you want to OPT OUT from this, re-generate " +
			"the deployment bundle with the '--enable-telemetry=false' " +
			"flag.")
	}

	wrapper, err := createBundle(&config)
	if err != nil {
		return err
	}

	var outputPath string
	if roxctl.InMainImage() {
		bytes, err := wrapper.Zip()
		if err != nil {
			return errors.Wrap(err, "error generating zip file")
		}
		_, err = io.Out().Write(bytes)
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

	logger.InfofLn("Done!")

	if outputPath != "" {
		logger.InfofLn("Wrote central bundle to %q.", outputPath)
	}

	if err := config.WriteInstructions(io.ErrOut()); err != nil {
		return err
	}
	return nil
}

func (cmd *centralGenerateCommand) interactive() *cobra.Command {
	return &cobra.Command{
		Use:   "interactive",
		Short: "Run the command in interactive mode where it prompts you to provide additional information.",
		Long:  "Run the command in interactive mode where it prompts you to provide additional information. You can press `Enter` to accept the default value for the prompts.",
		RunE: util.RunENoArgs(func(*cobra.Command) error {
			c := Command(cmd.env)
			c.SilenceUsage = true
			return runInteractive(c)
		}),
		SilenceUsage: true,
	}
}

type centralGenerateCommand struct {
	// Properties that are bound to cobra flags.
	rendererConfig *renderer.Config

	// Properties that are injected or constructed.
	env environment.Environment
}

// Command defines the generate command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {

	centralGenerateCmd := &centralGenerateCommand{rendererConfig: &cfg, env: cliEnvironment}

	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate the required YAML configuration files containing the orchestrator objects to deploy StackRox Central.",
	}
	c.PersistentFlags().StringVarP(&centralGenerateCmd.rendererConfig.Password, "password", "p", "", "administrator password (default: autogenerated)")
	utils.Must(c.PersistentFlags().SetAnnotation("password", flags.PasswordKey, []string{"true"}))

	c.PersistentFlags().Var(&flags.FileContentsVar{
		Data: &centralGenerateCmd.rendererConfig.DefaultTLSCertPEM,
	}, "default-tls-cert", "PEM cert bundle file")
	utils.Must(c.PersistentFlags().SetAnnotation("default-tls-cert", flags.OptionalKey, []string{"true"}))

	c.PersistentFlags().Var(&flags.FileContentsVar{
		Data: &centralGenerateCmd.rendererConfig.DefaultTLSKeyPEM,
	}, "default-tls-key", "PEM private key file")
	utils.Must(
		c.PersistentFlags().SetAnnotation("default-tls-key", flags.DependenciesKey, []string{"default-tls-cert"}),
		c.PersistentFlags().SetAnnotation("default-tls-key", flags.MandatoryKey, []string{"true"}),
	)
	c.PersistentFlags().StringVar(&centralGenerateCmd.rendererConfig.BackupBundle, "backup-bundle", "", "path to the backup bundle from which to restore keys and certificates")
	utils.Must(
		c.PersistentFlags().SetAnnotation("backup-bundle", flags.OptionalKey, []string{"true"}),
	)

	c.PersistentFlags().VarPF(
		flags.ForSetting(env.PlaintextEndpoints, cliEnvironment.Logger()), "plaintext-endpoints", "",
		"The ports or endpoints to use for plaintext (unencrypted) exposure; comma-separated list.")
	utils.Must(
		c.PersistentFlags().SetAnnotation("plaintext-endpoints", flags.NoInteractiveKey, []string{"true"}))

	c.PersistentFlags().Var(&flags.FileMapVar{
		FileMap: &centralGenerateCmd.rendererConfig.ConfigFileOverrides,
	}, "with-config-file", "Use the given local file(s) to override default config files")
	utils.Must(
		c.PersistentFlags().MarkHidden("with-config-file"))

	if !buildinfo.ReleaseBuild {
		flags.AddHelmChartDebugSetting(c)
	}
	c.PersistentFlags().BoolVar(&centralGenerateCmd.rendererConfig.EnablePodSecurityPolicies, "enable-pod-security-policies", true, "Create PodSecurityPolicy resources (for pre-v1.25 Kubernetes)")

	c.AddCommand(centralGenerateCmd.interactive())
	c.AddCommand(k8s(cliEnvironment))
	c.AddCommand(openshift(cliEnvironment))
	return c
}

func runInteractive(cmd *cobra.Command) error {
	mode.SetInteractiveMode()
	// Overwrite os.Args because cobra uses them
	os.Args = walkTree(cmd)
	return cmd.Execute()
}
