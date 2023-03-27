package create

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"gopkg.in/yaml.v3"
)

func authProviderCommand(cliEnvironment environment.Environment) *cobra.Command {
	authProviderCmd := &authProviderCmd{authProvider: &declarativeconfig.AuthProvider{}, env: cliEnvironment}

	cmd := &cobra.Command{
		Use: authProviderCmd.authProvider.Type(),
	}

	cmd.PersistentFlags().StringVar(&authProviderCmd.authProvider.Name, "name", "", "name of the auth provider")
	cmd.PersistentFlags().StringVar(&authProviderCmd.authProvider.MinimumRoleName, "minimum-access-role", "",
		"minimum access role of the auth provider. This can be left empty if the minimum access role should not"+
			" be configured via declarative configuration")
	cmd.PersistentFlags().StringVar(&authProviderCmd.authProvider.UIEndpoint, "ui-endpoint", "", "UI Endpoint "+
		"from which the auth provider is used (this is typically the public endpoint where RHACS is exposed). The "+
		"expected format is <endpoint>:<port>")
	cmd.PersistentFlags().StringSliceVar(&authProviderCmd.authProvider.ExtraUIEndpoints, "extra-ui-endpoints", []string{},
		"Additional UI endpoints from which the auth provider is used. The expected format is <endpoint>:<port>")

	cmd.PersistentFlags().StringToStringVar(&authProviderCmd.requiredAttributes, "required-attributes",
		map[string]string{}, `list of attributes that are required to be returned by the auth provider during
authentication, e.g. --required-attributes "my_org=sample-org"`)

	// pflag.FlagSet currently lacks to provide a tuple of repeated key value pairs, hence we need to resort to
	// providing three separate flags which hold the values required to construct a single group:
	// --groups-key, --groups-value, --group-role.
	// They can be repeated as such within the CLI flags:
	// --groups-key "email" --groups-value "my@domain.com" --groups-value"Admin" \
	// --groups-key "user" --groups-value "id" --groups-role "Analyst"
	cmd.PersistentFlags().StringSliceVar(&authProviderCmd.groupsKeys, "--groups-key", []string{},
		`keys of the groups to add within the auth provider. Note that the tuple of key, value, role should
be of the same length. Example of a group: --groups-key "email" --groups-value "my@domain.com" --groups-role "Admin"`)

	cmd.PersistentFlags().StringSliceVar(&authProviderCmd.groupsValues, "--groups-value", []string{},
		`values of the groups to add within the auth provider. Note that the tuple of key, value, role should
be of the same length. Example of a group: --groups-key "email" --groups-value "my@domain.com" --groups-role "Admin"`)
	cmd.PersistentFlags().StringSliceVar(&authProviderCmd.groupsKeys, "--groups-role", []string{},
		`role of the groups to add within the auth provider. Note that the tuple of key, value, role should
be of the same length. Example of a group: --groups-key "email" --groups-value "my@domain.com" --groups-role "Admin"`)

	cmd.MarkFlagsRequiredTogether("name", "ui-endpoint")

	cmd.AddCommand(
		authProviderCmd.oidcCommand(),
		authProviderCmd.samlCommand(),
		authProviderCmd.iapCommand(),
		authProviderCmd.userPKICommand(),
		authProviderCmd.openShiftCommand(),
	)

	return cmd
}

type authProviderCmd struct {
	authProvider *declarativeconfig.AuthProvider

	requiredAttributes map[string]string
	claimMapping       map[string]string

	groupsKeys   []string
	groupsValues []string
	groupsRoles  []string

	// Custom mappings for SAML command.
	samlIDPCertFile string

	// Custom mapping for User PKI command.
	userPKICAFile string

	env environment.Environment
}

func (a *authProviderCmd) oidcCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "oidc",
		Args: cobra.NoArgs,
		RunE: a.RunE(),
	}

	cmd.Flags().StringVar(&a.authProvider.OIDCConfig.Issuer, "issuer", "",
		"issuer of the OIDC client")
	cmd.Flags().StringVar(&a.authProvider.OIDCConfig.CallbackMode, "mode", "auto",
		"The callback mode to use. Possible values are: auto, post, query, fragment")
	cmd.Flags().StringVar(&a.authProvider.OIDCConfig.ClientID, "client-id", "",
		"Client ID of the OIDC client")
	cmd.Flags().StringVar(&a.authProvider.OIDCConfig.ClientSecret, "client-secret", "",
		"Client Secret of the OIDC client")
	cmd.Flags().BoolVar(&a.authProvider.OIDCConfig.DisableOfflineAccessScope, "disable-offline-access", false,
		"Disable requesting the scope offline_access from the OIDC identity provider. This should only be set "+
			"if there are any limitations from your OIDC IdP about the number of sessions with the offline_access scope")

	cmd.PersistentFlags().StringToStringVar(&a.claimMapping, "claim-mappings", map[string]string{},
		`list of non-standard claims from the IdP token that should be available within auth provider rules, e.g.
--claim-mappings "my_claim_on_the_idp_token=claim_name_on_the_rox_token"`)

	return cmd
}

func (a *authProviderCmd) samlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "saml",
		Args: cobra.NoArgs,
		RunE: a.RunE(),
	}

	cmd.Flags().StringVar(&a.authProvider.SAMLConfig.SpIssuer, "sp-issuer", "", "service provider "+
		"issuer")
	cmd.Flags().StringVar(&a.authProvider.SAMLConfig.MetadataURL, "metadata-url", "", "metadata "+
		"URL of the service provider")
	cmd.Flags().StringVar(&a.samlIDPCertFile, "idp-cert", "", "file containing the SAML IdP "+
		"certificate in PEM format")
	cmd.Flags().StringVar(&a.authProvider.SAMLConfig.SsoURL, "sso-url", "", "URL of the IdP")
	cmd.Flags().StringVar(&a.authProvider.SAMLConfig.NameIDFormat, "name-id-format", "",
		"Name ID format")
	cmd.Flags().StringVar(&a.authProvider.SAMLConfig.IDPIssuer, "idp-issuer", "", "issuer of the IdP")

	return cmd
}

func (a *authProviderCmd) iapCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "iap",
		Args: cobra.NoArgs,
		RunE: a.RunE(),
	}

	cmd.Flags().StringVar(&a.authProvider.IAPConfig.Audience, "audience", "", "audience that should "+
		"be validated")

	utils.Must(cmd.MarkFlagRequired("audience"))

	return cmd
}

func (a *authProviderCmd) userPKICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "user-pki",
		Args: cobra.NoArgs,
		RunE: a.RunE(),
	}

	cmd.Flags().StringVar(&a.userPKICAFile, "ca-file", "", "file containing the certificate "+
		"authorities in PEM format")

	utils.Must(cmd.MarkFlagRequired("ca-file"))

	return cmd
}

func (a *authProviderCmd) openShiftCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "openshift-auth",
		Args: cobra.NoArgs,
		RunE: a.RunE(),
	}

	return cmd
}

func (a *authProviderCmd) RunE() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := a.Validate(cmd.Use); err != nil {
			return errors.Wrap(err, "validating auth provider")
		}
		return a.PrintYAML()
	}
}

func (a *authProviderCmd) Validate(providerType string) error {
	requiredAttributes := make([]declarativeconfig.RequiredAttribute, 0, len(a.requiredAttributes))
	for key, value := range a.requiredAttributes {
		requiredAttributes = append(requiredAttributes, declarativeconfig.RequiredAttribute{
			AttributeKey:   key,
			AttributeValue: value,
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
			a.authProvider.SAMLConfig.Cert = samlCert
		}
	case "user-pki":
		ca, err := readFileContents(a.userPKICAFile)
		if err != nil {
			return errors.Wrap(err, "reading user PKI CA file")
		}
		a.authProvider.UserpkiConfig.CertificateAuthorities = ca
	case "openshift-auth":
		a.authProvider.OpenshiftConfig.Enable = true
	case "oid":
		claimMappings := make([]declarativeconfig.ClaimMapping, 0, len(a.claimMapping))
		for path, name := range a.claimMapping {
			claimMappings = append(claimMappings, declarativeconfig.ClaimMapping{
				Path: path,
				Name: name,
			})
		}
		a.authProvider.ClaimMappings = claimMappings
	}

	t := transform.New()
	_, err = t.Transform(a.authProvider)
	return errors.Wrap(err, "validating auth provider")
}

func (a *authProviderCmd) validateGroups() ([]declarativeconfig.Group, error) {
	expectedGroups := len(a.groupsKeys)

	if len(a.groupsKeys) != expectedGroups || len(a.groupsValues) != expectedGroups || len(a.groupsRoles) != expectedGroups {
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
	enc := yaml.NewEncoder(a.env.InputOutput().Out())
	return errors.Wrap(enc.Encode(a.authProvider), "creating the YAML output")
}
