package create

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"github.com/stackrox/rox/roxctl/declarativeconfig/lint"
	"gopkg.in/yaml.v3"
)

var (
	persistentFlagsToShow = []string{
		"name",
		"minimum-access-role",
		"ui-endpoint",
		"extra-ui-endpoints",
		"required-attributes",
		"groups-key",
		"groups-value",
		"groups-role",
		k8sobject.ConfigMapFlag,
		k8sobject.NamespaceFlag,
	}
)

func authProviderCommand(cliEnvironment environment.Environment) *cobra.Command {
	authProviderCmd := &authProviderCmd{authProvider: &declarativeconfig.AuthProvider{}, env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   authProviderCmd.authProvider.ConfigurationType(),
		Short: "Commands to create a declarative configuration for an auth provider",
	}

	cmd.PersistentFlags().StringVar(&authProviderCmd.authProvider.Name, "name", "", "name of the auth provider")
	cmd.PersistentFlags().StringVar(&authProviderCmd.authProvider.MinimumRoleName, "minimum-access-role", "",
		`minimum access role of the auth provider.
This can be left empty if the minimum access role should not be configured via declarative configuration`)
	cmd.PersistentFlags().StringVar(&authProviderCmd.authProvider.UIEndpoint, "ui-endpoint", "",
		`UI Endpoint from which the auth provider is used (this is typically the public endpoint where RHACS is exposed).
The expected format is <endpoint>:<port>`)
	cmd.PersistentFlags().StringSliceVar(&authProviderCmd.authProvider.ExtraUIEndpoints, "extra-ui-endpoints", []string{},
		`Additional UI endpoints from which the auth provider is used.
The expected format is <endpoint>:<port>`)

	cmd.PersistentFlags().StringToStringVar(&authProviderCmd.requiredAttributes, "required-attributes",
		map[string]string{}, `list of attributes that are required to be returned by the auth provider during authentication,
e.g. --required-attributes "my_org=sample-org"`)

	// pflag.FlagSet currently lacks to provide a tuple of repeated key value pairs, hence we need to resort to
	// providing three separate flags which hold the values required to construct a single group:
	// --groups-key, --groups-value, --group-role.
	// They can be repeated as such within the CLI flags:
	// --groups-key "email" --groups-value "my@domain.com" --groups-value"Admin" \
	// --groups-key "user" --groups-value "id" --groups-role "Analyst"
	cmd.PersistentFlags().StringSliceVar(&authProviderCmd.groupsKeys, "groups-key", []string{},
		`keys of the groups to add within the auth provider. Note that the tuple of key, value, role should
be of the same length.
Example of a group: --groups-key "email" --groups-value "my@domain.com" --groups-role "Admin"`)

	cmd.PersistentFlags().StringSliceVar(&authProviderCmd.groupsValues, "groups-value", []string{},
		`values of the groups to add within the auth provider. Note that the tuple of key, value, role should
be of the same length.
Example of a group: --groups-key "email" --groups-value "my@domain.com" --groups-role "Admin"`)
	cmd.PersistentFlags().StringSliceVar(&authProviderCmd.groupsRoles, "groups-role", []string{},
		`role of the groups to add within the auth provider. Note that the tuple of key, value, role should
be of the same length.
Example of a group: --groups-key "email" --groups-value "my@domain.com" --groups-role "Admin"`)

	cmd.MarkFlagsRequiredTogether("name", "ui-endpoint")

	cmd.AddCommand(
		authProviderCmd.oidcCommand(),
		authProviderCmd.samlCommand(),
		authProviderCmd.iapCommand(),
		authProviderCmd.userPKICommand(),
		authProviderCmd.openShiftCommand(),
	)

	flags.HideInheritedFlags(cmd, k8sobject.ConfigMapFlag, k8sobject.NamespaceFlag)

	return cmd
}

type authProviderCmd struct {
	authProvider *declarativeconfig.AuthProvider

	requiredAttributes map[string]string
	claimMapping       map[string]string

	groupsKeys   []string
	groupsValues []string
	groupsRoles  []string

	oidcConfig    *declarativeconfig.OIDCConfig
	samlConfig    *declarativeconfig.SAMLConfig
	iapConfig     *declarativeconfig.IAPConfig
	userPKIConfig *declarativeconfig.UserpkiConfig

	// Custom mappings for SAML command.
	samlIDPCertFile string

	// Custom mapping for User PKI command.
	userPKICAFile string

	env environment.Environment

	configMap string
	secret    string
	namespace string
}

func (a *authProviderCmd) oidcCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oidc",
		Args:  cobra.NoArgs,
		RunE:  a.RunE(),
		Short: "Create a declarative configuration for an OIDC auth provider",
	}
	a.oidcConfig = &declarativeconfig.OIDCConfig{}

	cmd.Flags().StringVar(&a.oidcConfig.Issuer, "issuer", "",
		"issuer of the OIDC client")
	cmd.Flags().StringVar(&a.oidcConfig.CallbackMode, "mode", "auto",
		"The callback mode to use. Possible values are: auto, post, query, fragment")
	cmd.Flags().StringVar(&a.oidcConfig.ClientID, "client-id", "",
		"Client ID of the OIDC client")
	cmd.Flags().StringVar(&a.oidcConfig.ClientSecret, "client-secret", "",
		"Client Secret of the OIDC client")
	cmd.Flags().BoolVar(&a.oidcConfig.DisableOfflineAccessScope, "disable-offline-access", false,
		"Disable requesting the scope offline_access from the OIDC identity provider. This should only be set "+
			"if there are any limitations from your OIDC IdP about the number of sessions with the offline_access scope")

	cmd.Flags().StringToStringVar(&a.claimMapping, "claim-mappings", map[string]string{},
		`list of non-standard claims from the IdP token that should be available within auth provider rules, e.g.
--claim-mappings "my_claim_on_the_idp_token=claim_name_on_the_rox_token"`)

	utils.Must(cmd.MarkFlagRequired("issuer"))
	utils.Must(cmd.MarkFlagRequired("client-id"))

	flags.HideInheritedFlags(cmd, persistentFlagsToShow...)
	return cmd
}

func (a *authProviderCmd) samlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "saml",
		Args:  cobra.NoArgs,
		RunE:  a.RunE(),
		Short: "Create a declarative configuration for a SAML auth provider",
	}
	a.samlConfig = &declarativeconfig.SAMLConfig{}

	cmd.Flags().StringVar(&a.samlConfig.SpIssuer, "sp-issuer", "", "service provider "+
		"issuer")
	cmd.Flags().StringVar(&a.samlConfig.MetadataURL, "metadata-url", "", "metadata "+
		"URL of the service provider")
	cmd.Flags().StringVar(&a.samlIDPCertFile, "idp-cert", "", "file containing the SAML IdP "+
		"certificate in PEM format")
	cmd.Flags().StringVar(&a.samlConfig.SsoURL, "sso-url", "", "URL of the IdP")
	cmd.Flags().StringVar(&a.samlConfig.NameIDFormat, "name-id-format", "",
		"Name ID format")
	cmd.Flags().StringVar(&a.samlConfig.IDPIssuer, "idp-issuer", "", "issuer of the IdP")

	utils.Must(cmd.MarkFlagRequired("sp-issuer"))
	cmd.MarkFlagsRequiredTogether("idp-cert", "sso-url", "idp-issuer")
	cmd.MarkFlagsMutuallyExclusive("metadata-url", "sso-url")

	flags.HideInheritedFlags(cmd, persistentFlagsToShow...)
	return cmd
}

func (a *authProviderCmd) iapCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "iap",
		Args: cobra.NoArgs,
		RunE: a.RunE(),
	}
	a.iapConfig = &declarativeconfig.IAPConfig{}

	cmd.Flags().StringVar(&a.iapConfig.Audience, "audience", "", "audience that should "+
		"be validated")

	utils.Must(cmd.MarkFlagRequired("audience"))

	flags.HideInheritedFlags(cmd, persistentFlagsToShow...)
	return cmd
}

func (a *authProviderCmd) userPKICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "userpki",
		Args:  cobra.NoArgs,
		RunE:  a.RunE(),
		Short: "Create a declarative configuration for an user PKI auth provider",
	}
	a.userPKIConfig = &declarativeconfig.UserpkiConfig{}

	cmd.Flags().StringVar(&a.userPKICAFile, "ca-file", "", "file containing the certificate "+
		"authorities in PEM format")

	utils.Must(cmd.MarkFlagRequired("ca-file"))

	flags.HideInheritedFlags(cmd, persistentFlagsToShow...)
	return cmd
}

func (a *authProviderCmd) openShiftCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openshift-auth",
		RunE:  a.RunE(),
		Short: "Create a declarative configuration for an OpenShift-Auth auth provider",
	}

	flags.HideInheritedFlags(cmd, persistentFlagsToShow...)
	return cmd
}

func (a *authProviderCmd) RunE() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := a.Construct(cmd); err != nil {
			return err
		}
		if err := a.Validate(cmd.Use); err != nil {
			return err
		}
		return a.PrintYAML()
	}
}

func (a *authProviderCmd) Construct(cmd *cobra.Command) error {
	configMap, secret, namespace, err := k8sobject.ReadK8sObjectFlags(cmd)
	if err != nil {
		return errors.Wrap(err, "reading config map flag values")
	}
	a.configMap = configMap
	a.secret = secret
	a.namespace = namespace
	return nil
}

func (a *authProviderCmd) Validate(providerType string) error {
	requiredAttributes := make([]declarativeconfig.RequiredAttribute, 0, len(a.requiredAttributes))
	keys := maputil.Keys(a.requiredAttributes)
	sort.Strings(keys)
	for _, key := range keys {
		requiredAttributes = append(requiredAttributes, declarativeconfig.RequiredAttribute{
			AttributeKey:   key,
			AttributeValue: a.requiredAttributes[key],
		})
	}
	a.authProvider.RequiredAttributes = requiredAttributes

	groups, err := a.validateGroups()
	if err != nil {
		return errors.Wrap(err, "validating groups")
	}
	a.authProvider.Groups = groups

	switch providerType {
	case "saml":
		if a.samlIDPCertFile != "" {
			samlCert, err := readFileContents(a.samlIDPCertFile)
			if err != nil {
				return errors.Wrap(err, "reading SAML IdP cert file")
			}
			a.samlConfig.Cert = samlCert
		}
		a.authProvider.SAMLConfig = a.samlConfig
	case "userpki":
		ca, err := readFileContents(a.userPKICAFile)
		if err != nil {
			return errors.Wrap(err, "reading user PKI CA file")
		}
		a.userPKIConfig.CertificateAuthorities = ca
		a.authProvider.UserpkiConfig = a.userPKIConfig
	case "openshift-auth":
		a.authProvider.OpenshiftConfig = &declarativeconfig.OpenshiftConfig{Enable: true}
	case "oidc":
		claimMappings := make([]declarativeconfig.ClaimMapping, 0, len(a.claimMapping))
		paths := maputil.Keys(a.claimMapping)
		sort.Strings(paths)
		for _, path := range paths {
			claimMappings = append(claimMappings, declarativeconfig.ClaimMapping{
				Path: path,
				Name: a.claimMapping[path],
			})
		}
		a.authProvider.ClaimMappings = claimMappings
		a.authProvider.OIDCConfig = a.oidcConfig
	case "iap":
		a.authProvider.IAPConfig = a.iapConfig
	}
	return nil
}

func (a *authProviderCmd) validateGroups() ([]declarativeconfig.Group, error) {
	expectedGroups := len(a.groupsKeys)

	if len(a.groupsValues) != expectedGroups || len(a.groupsRoles) != expectedGroups {
		return nil, errox.InvalidArgs.Newf("the groups tuple of key, value, role should have the the same "+
			"number of entries, but found a mismatch [keys %d, values %d, roles %d]",
			len(a.groupsKeys), len(a.groupsValues), len(a.groupsRoles))
	}

	groups := make([]declarativeconfig.Group, 0, expectedGroups)
	for i := 0; i < expectedGroups; i++ {
		groups = append(groups, declarativeconfig.Group{
			AttributeKey:   a.groupsKeys[i],
			AttributeValue: a.groupsValues[i],
			RoleName:       a.groupsRoles[i],
		})
	}

	return groups, nil
}

func readFileContents(f string) (string, error) {
	contents, err := os.ReadFile(f)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errox.NotFound.CausedBy(err)
		}
		return "", errox.InvalidArgs.CausedBy(err)
	}
	return string(contents), nil
}

func (a *authProviderCmd) PrintYAML() error {
	yamlOut := &bytes.Buffer{}
	enc := yaml.NewEncoder(yamlOut)
	if err := enc.Encode(a.authProvider); err != nil {
		return errors.Wrap(err, "creating the YAML output")
	}
	if err := lint.Lint(yamlOut.Bytes()); err != nil {
		return errors.Wrap(err, "linting the YAML output")
	}
	if a.configMap != "" || a.secret != "" {
		return errors.Wrap(k8sobject.WriteToK8sObject(context.Background(), a.configMap, a.secret, a.namespace,
			fmt.Sprintf("%s-%s", a.authProvider.ConfigurationType(), a.authProvider.Name), yamlOut.Bytes()),
			"writing the YAML output to config map")
	}

	if _, err := a.env.InputOutput().Out().Write(yamlOut.Bytes()); err != nil {
		return errors.Wrap(err, "writing the YAML output")
	}
	return nil
}
